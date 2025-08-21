package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"net"
	"strings"
	"sync"
	"time"

	"gothoom/eui"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	dark "github.com/thiagokokada/dark-mode-go"
	clipboard "golang.design/x/clipboard"
)

const lateRatio = 80
const gameAreaSizeX, gameAreaSizeY = 547, 540
const fieldCenterX, fieldCenterY = gameAreaSizeX / 2, gameAreaSizeY / 2
const defaultHandPictID = 6
const initialWindowW, initialWindowH = 1920, 1080

var uiMouseDown bool

// worldRT is the offscreen render target for the game world. It stays at an
// integer multiple of the native field size and is composited into the window.
var worldRT *ebiten.Image

// gameImageItem is the UI image item inside the game window that displays
// the rendered world, and gameImage is its backing texture.
var gameImageItem *eui.ItemData
var gameImage *ebiten.Image
var inAspectResize bool

// dimmedScreenBG holds the theme window background color dimmed by 25%.
// updateDimmedScreenBG refreshes this color when the theme changes.
var dimmedScreenBG = color.RGBA{0, 0, 0, 255}

func updateDimmedScreenBG() {
	c := color.RGBA{0, 0, 0, 255}
	if gameWin != nil && gameWin.Theme != nil {
		if tc := color.RGBA(gameWin.Theme.Window.BGColor); tc.A > 0 {
			c = tc
		}
	}
	dimmedScreenBG = color.RGBA{
		R: uint8(uint16(c.R) / 2),
		G: uint8(uint16(c.G) / 2),
		B: uint8(uint16(c.B) / 2),
		A: 255,
	}
}
func ensureWorldRT(w, h int) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	if worldRT == nil || worldRT.Bounds().Dx() != w || worldRT.Bounds().Dy() != h {
		// Use unmanaged images for faster off-screen rendering.
		worldRT = ebiten.NewImageWithOptions(image.Rect(0, 0, w, h), &ebiten.NewImageOptions{Unmanaged: true})
	}
}

// updateGameImageSize ensures the game image item exists and matches the
// current inner content size of the game window.
func updateGameImageSize() {
	if gameWin == nil {
		return
	}
	size := gameWin.GetSize()
	pad := float64(2 * gameWin.Padding)
	title := float64(gameWin.GetTitleSize())
	// Inner content size (exclude titlebar and inside padding)
	cw := int(float64(int(size.X)&^1) - pad)
	ch := int(float64(int(size.Y)&^1) - pad - title)
	// Leave a 2px margin on all sides for window edges
	w := cw - 4
	h := ch - 4
	if w <= 0 || h <= 0 {
		return
	}
	if gameImageItem == nil {
		it, img := eui.NewImageFastItem(w, h)
		gameImageItem = it
		gameImage = img
		gameImageItem.Position = eui.Point{X: 2, Y: 2}
		gameWin.AddItem(gameImageItem)
		return
	}
	// Resize backing image only when dimensions change
	iw, ih := 0, 0
	if gameImage != nil {
		b := gameImage.Bounds()
		iw, ih = b.Dx(), b.Dy()
	}
	if iw != w || ih != h {
		gameImage = ebiten.NewImage(w, h)
		gameImageItem.Image = gameImage
		gameImageItem.Size = eui.Point{X: float32(w), Y: float32(h)}
		gameImageItem.Position = eui.Point{X: 2, Y: 2}
		if gameWin != nil {
			gameWin.Dirty = true
		}
	}
}

// scaleForFiltering returns adjusted scale values for width and height to reduce
// filtering seams. If either dimension is zero, the original scale is returned
// unchanged to avoid division by zero on the half-texel offset.
func scaleForFiltering(scale float64, w, h int) (float64, float64) {
	if w == 0 || h == 0 {
		// Zero-sized image: keep the original scale.
		return scale, scale
	}

	ps, exact := exactScale(scale, 8, 1e-6) // denom ≤ 8, ε = 1e-6

	if exact {
		// Exact integer or exact small rational: no offset needed.
		return ps, ps
	}

	// Not exact → keep requested scale but nudge by half-texel to reduce seams.
	return scale + 0.5/float64(w), scale + 0.5/float64(h)
}

func exactScale(scale float64, maxDenom int, eps float64) (float64, bool) {
	// Exact integer?
	r := math.Round(scale)
	if math.Abs(scale-r) <= eps {
		return r, true
	}

	// Exact small rational num/den?
	// We look for a den <= maxDenom where num/den ≈ scale within eps.
	best := scale
	for den := 2; den <= maxDenom; den++ {
		num := math.Round(scale * float64(den))
		ideal := num / float64(den)
		if math.Abs(scale-ideal) <= eps {
			best = ideal
			return best, true
		}
	}
	return best, false
}

type inputState struct {
	mouseX, mouseY int16
	mouseDown      bool
}

var (
	latestInput inputState
	inputMu     sync.Mutex
)

var keyX, keyY int16
var walkToggled bool

var inputActive bool
var inputText []rune
var inputHistory []string
var historyPos int

var (
	recorder            *movieRecorder
	gPlayersListIsStale bool
	loginGameState      []byte
	loginMobileData     []byte
	loginPictureTable   []byte
	wroteLoginBlocks    bool
)

// gameWin represents the main playfield window. Its size corresponds to the
// classic client field box (547×540) defined in old_mac_client/client/source/
// GameWin_cl.cp and Public_cl.h (Layout.layoFieldBox).
var gameWin *eui.WindowData
var settingsWin *eui.WindowData
var debugWin *eui.WindowData
var qualityWin *eui.WindowData
var graphicsWin *eui.WindowData
var bubbleWin *eui.WindowData
var notificationsWin *eui.WindowData

var (
	lastDebugStatsUpdate   time.Time
	lastQualityPresetCheck time.Time
	lastMovieWinRefresh    time.Time
)

// Deprecated: sound settings window removed; kept other windows.
var gameCtx context.Context
var frameCounter int
var gameStarted = make(chan struct{})

const framems = 200

var (
	frameCh       = make(chan struct{}, 1)
	lastFrameTime time.Time
	frameInterval = framems * time.Millisecond
	intervalHist  = map[int]int{}
	frameMu       sync.Mutex
	serverFPS     float64
	netLatency    time.Duration
	lastInputSent time.Time
	latencyMu     sync.Mutex
)

// drawState tracks information needed by the Ebiten renderer.
type drawState struct {
	descriptors map[uint8]frameDescriptor
	pictures    []framePicture
	picShiftX   int
	picShiftY   int
	mobiles     map[uint8]frameMobile
	prevMobiles map[uint8]frameMobile
	prevDescs   map[uint8]frameDescriptor
	prevTime    time.Time
	curTime     time.Time

	bubbles []bubble

	hp, hpMax                   int
	sp, spMax                   int
	balance, balanceMax         int
	prevHP, prevHPMax           int
	prevSP, prevSPMax           int
	prevBalance, prevBalanceMax int
	ackCmd                      uint8
	lightingFlags               uint8

	// Prepared render caches populated only when a new game state arrives.
	// These avoid per-frame sorting and partitioning work in Draw.
	picsNeg  []framePicture
	picsZero []framePicture
	picsPos  []framePicture
	liveMobs []frameMobile
	deadMobs []frameMobile
	nameMobs []frameMobile
}

var (
	state = drawState{
		descriptors: make(map[uint8]frameDescriptor),
		mobiles:     make(map[uint8]frameMobile),
		prevMobiles: make(map[uint8]frameMobile),
		prevDescs:   make(map[uint8]frameDescriptor),
	}
	initialState drawState
	stateMu      sync.Mutex
)

// prepareRenderCacheLocked populates render-ready, sorted/partitioned slices.
// Call with stateMu held and only when a new game state is applied.
func prepareRenderCacheLocked() {
	// Mobiles: split into live and dead, sort by V then H, and prepare
	// a separate slice sorted right-to-left/top-to-bottom for name tags.
	state.liveMobs = state.liveMobs[:0]
	state.deadMobs = state.deadMobs[:0]
	for _, m := range state.mobiles {
		if m.State == poseDead {
			state.deadMobs = append(state.deadMobs, m)
		}
		state.liveMobs = append(state.liveMobs, m)
	}
	sortMobiles(state.deadMobs)
	sortMobiles(state.liveMobs)

	state.nameMobs = append(state.nameMobs[:0], state.liveMobs...)
	sortMobilesNameTags(state.nameMobs)

	// Pictures: sort once, then partition by plane while preserving order.
	// Work on a copy to avoid reordering the canonical state.pictures slice
	// which is also copied into snapshots.
	tmp := append([]framePicture(nil), state.pictures...)
	sortPictures(tmp)
	state.picsNeg = state.picsNeg[:0]
	state.picsZero = state.picsZero[:0]
	state.picsPos = state.picsPos[:0]
	for _, p := range tmp {
		switch {
		case p.Plane < 0:
			state.picsNeg = append(state.picsNeg, p)
		case p.Plane == 0:
			state.picsZero = append(state.picsZero, p)
		default:
			state.picsPos = append(state.picsPos, p)
		}
	}
}

// bubble stores temporary chat bubble information. Bubbles expire after a
// fixed number of game update frames from when they were created — no FPS
// correction or wall-clock timing is applied to keep playback simple.
const bubbleLifeFrames = (1000 / framems) * 4 // ~4s

type bubble struct {
	Index        uint8
	H, V         int16
	Far          bool
	NoArrow      bool
	Text         string
	Type         int
	CreatedFrame int
}

// drawSnapshot is a read-only copy of the current draw state.
type drawSnapshot struct {
	descriptors                 map[uint8]frameDescriptor
	pictures                    []framePicture
	picShiftX                   int
	picShiftY                   int
	mobiles                     []frameMobile // sorted right-to-left, top-to-bottom
	prevMobiles                 map[uint8]frameMobile
	prevDescs                   map[uint8]frameDescriptor
	prevTime                    time.Time
	curTime                     time.Time
	bubbles                     []bubble
	hp, hpMax                   int
	sp, spMax                   int
	balance, balanceMax         int
	prevHP, prevHPMax           int
	prevSP, prevSPMax           int
	prevBalance, prevBalanceMax int
	ackCmd                      uint8
	lightingFlags               uint8

	// Precomputed, sorted/partitioned data for rendering
	picsNeg  []framePicture
	picsZero []framePicture
	picsPos  []framePicture
	liveMobs []frameMobile
	deadMobs []frameMobile
}

// captureDrawSnapshot copies the shared draw state under a mutex.
func captureDrawSnapshot() drawSnapshot {
	stateMu.Lock()
	defer stateMu.Unlock()

	snap := drawSnapshot{
		descriptors:    make(map[uint8]frameDescriptor, len(state.descriptors)),
		pictures:       append([]framePicture(nil), state.pictures...),
		picShiftX:      state.picShiftX,
		picShiftY:      state.picShiftY,
		mobiles:        append([]frameMobile(nil), state.nameMobs...),
		prevTime:       state.prevTime,
		curTime:        state.curTime,
		hp:             state.hp,
		hpMax:          state.hpMax,
		sp:             state.sp,
		spMax:          state.spMax,
		balance:        state.balance,
		balanceMax:     state.balanceMax,
		prevHP:         state.prevHP,
		prevHPMax:      state.prevHPMax,
		prevSP:         state.prevSP,
		prevSPMax:      state.prevSPMax,
		prevBalance:    state.prevBalance,
		prevBalanceMax: state.prevBalanceMax,
		ackCmd:         state.ackCmd,
		lightingFlags:  state.lightingFlags,
		// prepared caches
		picsNeg:  append([]framePicture(nil), state.picsNeg...),
		picsZero: append([]framePicture(nil), state.picsZero...),
		picsPos:  append([]framePicture(nil), state.picsPos...),
		liveMobs: append([]frameMobile(nil), state.liveMobs...),
		deadMobs: append([]frameMobile(nil), state.deadMobs...),
	}

	for idx, d := range state.descriptors {
		snap.descriptors[idx] = d
	}
	if len(state.bubbles) > 0 {
		curFrame := frameCounter
		kept := state.bubbles[:0]
		for _, b := range state.bubbles {
			if (curFrame - b.CreatedFrame) < bubbleLifeFrames {
				if !b.Far {
					if m, ok := state.mobiles[b.Index]; ok {
						b.H, b.V = m.H, m.V
					}
				}
				kept = append(kept, b)
			}
		}
		last := make(map[uint8]int)
		for i, b := range kept {
			last[b.Index] = i
		}
		dedup := kept[:0]
		for i, b := range kept {
			if last[b.Index] == i {
				dedup = append(dedup, b)
			}
		}
		state.bubbles = dedup
		snap.bubbles = append([]bubble(nil), state.bubbles...)
	}
	if gs.MotionSmoothing || gs.BlendMobiles {
		snap.prevMobiles = make(map[uint8]frameMobile, len(state.prevMobiles))
		for idx, m := range state.prevMobiles {
			snap.prevMobiles[idx] = m
		}
	}
	if gs.BlendMobiles {
		snap.prevDescs = make(map[uint8]frameDescriptor, len(state.prevDescs))
		for idx, d := range state.prevDescs {
			snap.prevDescs[idx] = d
		}
	}
	return snap
}

// cloneDrawState makes a deep copy of a drawState.
func cloneDrawState(src drawState) drawState {
	dst := drawState{
		descriptors:    make(map[uint8]frameDescriptor, len(src.descriptors)),
		pictures:       append([]framePicture(nil), src.pictures...),
		picShiftX:      src.picShiftX,
		picShiftY:      src.picShiftY,
		mobiles:        make(map[uint8]frameMobile, len(src.mobiles)),
		prevMobiles:    make(map[uint8]frameMobile, len(src.prevMobiles)),
		prevDescs:      make(map[uint8]frameDescriptor, len(src.prevDescs)),
		prevTime:       src.prevTime,
		curTime:        src.curTime,
		bubbles:        append([]bubble(nil), src.bubbles...),
		hp:             src.hp,
		hpMax:          src.hpMax,
		sp:             src.sp,
		spMax:          src.spMax,
		balance:        src.balance,
		balanceMax:     src.balanceMax,
		prevHP:         src.prevHP,
		prevHPMax:      src.prevHPMax,
		prevSP:         src.prevSP,
		prevSPMax:      src.prevSPMax,
		prevBalance:    src.prevBalance,
		prevBalanceMax: src.prevBalanceMax,
		ackCmd:         src.ackCmd,
		lightingFlags:  src.lightingFlags,
	}
	for idx, d := range src.descriptors {
		dst.descriptors[idx] = d
	}
	for idx, m := range src.mobiles {
		dst.mobiles[idx] = m
	}
	for idx, m := range src.prevMobiles {
		dst.prevMobiles[idx] = m
	}
	for idx, d := range src.prevDescs {
		dst.prevDescs[idx] = d
	}
	return dst
}

// computeInterpolation returns the blend factors for frame interpolation and onion skinning.
// It returns separate fade values for mobiles and pictures based on their respective rates.
func computeInterpolation(prevTime, curTime time.Time, mobileRate, pictRate float64) (alpha float64, mobileFade, pictFade float32) {
	alpha = 1.0
	mobileFade = 1.0
	pictFade = 1.0
	if (gs.MotionSmoothing || gs.BlendMobiles || gs.BlendPicts) && !curTime.IsZero() && curTime.After(prevTime) {
		elapsed := time.Since(prevTime)
		interval := curTime.Sub(prevTime)
		if gs.MotionSmoothing {
			alpha = float64(elapsed) / float64(interval)
			if alpha < 0 {
				alpha = 0
			}
			if alpha > 1 {
				alpha = 1
			}
		}
		if gs.BlendMobiles {
			half := float64(interval) * mobileRate
			if half > 0 {
				mobileFade = float32(float64(elapsed) / float64(half))
			}
			if mobileFade < 0 {
				mobileFade = 0
			}
			if mobileFade > 1 {
				mobileFade = 1
			}
		}
		if gs.BlendPicts {
			half := float64(interval) * pictRate
			if half > 0 {
				pictFade = float32(float64(elapsed) / float64(half))
			}
			if pictFade < 0 {
				pictFade = 0
			}
			if pictFade > 1 {
				pictFade = 1
			}
		}
	}
	return alpha, mobileFade, pictFade
}

type Game struct{}

var once sync.Once

func (g *Game) Update() error {
	select {
	case <-gameCtx.Done():
		syncWindowSettings()
		return errors.New("shutdown")
	default:
	}
	eui.Update() //We really need this to return eaten clicks
	updateNotifications()
	updateThinkMessages()

	once.Do(func() {
		initGame()
	})

	if debugWin != nil && debugWin.IsOpen() {
		if time.Since(lastDebugStatsUpdate) >= time.Second {
			updateDebugStats()
			lastDebugStatsUpdate = time.Now()
		}
	}

	if inventoryDirty {
		updateInventoryWindow()
		updateHandsWindow()
		inventoryDirty = false
	}

	if playersDirty {
		updatePlayersWindow()
		playersDirty = false
	}

	if syncWindowSettings() {
		settingsDirty = true
	}

	if time.Since(lastQualityPresetCheck) >= time.Second {
		if settingsDirty && qualityPresetDD != nil {
			qualityPresetDD.Selected = detectQualityPreset()
		}
		lastQualityPresetCheck = time.Now()
	}

	if time.Since(lastSettingsSave) >= time.Second {
		if settingsDirty {
			saveSettings()
			settingsDirty = false
		}
		lastSettingsSave = time.Now()
	}

	if time.Since(lastPlayersSave) >= 10*time.Second {
		if playersDirty || playersPersistDirty {
			savePlayersPersist()
			playersPersistDirty = false
		}
		lastPlayersSave = time.Now()
	}

	if movieWin != nil && movieWin.IsOpen() {
		if time.Since(lastMovieWinRefresh) >= time.Second {
			movieWin.Refresh()
			lastMovieWinRefresh = time.Now()
		}
	}

	/* Console input */
	changedInput := false
	if inputActive {
		if newChars := ebiten.AppendInputChars(nil); len(newChars) > 0 {
			inputText = append(inputText, newChars...)
			changedInput = true
		}
		ctrl := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight)
		if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyV) {
			if txt := clipboard.Read(clipboard.FmtText); len(txt) > 0 {
				inputText = append(inputText, []rune(string(txt))...)
				changedInput = true
			}
		}
		if ctrl && inpututil.IsKeyJustPressed(ebiten.KeyC) {
			clipboard.Write(clipboard.FmtText, []byte(string(inputText)))
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			if len(inputHistory) > 0 {
				if historyPos > 0 {
					historyPos--
				} else {
					historyPos = 0
				}
				inputText = []rune(inputHistory[historyPos])
				changedInput = true
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			if len(inputHistory) > 0 {
				if historyPos < len(inputHistory)-1 {
					historyPos++
					inputText = []rune(inputHistory[historyPos])
					changedInput = true
				} else {
					historyPos = len(inputHistory)
					inputText = inputText[:0]
					changedInput = true
				}
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			if len(inputText) > 0 {
				inputText = inputText[:len(inputText)-1]
				changedInput = true
			}
		} else if d := inpututil.KeyPressDuration(ebiten.KeyBackspace); d > 30 && d%3 == 0 {
			if len(inputText) > 0 {
				inputText = inputText[:len(inputText)-1]
				changedInput = true
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			txt := strings.TrimSpace(string(inputText))
			if txt != "" {
				if strings.HasPrefix(txt, "/play ") {
					go playClanLordTune(strings.TrimSpace(txt[len("/play "):]))
				} else {
					pendingCommand = txt
					//consoleMessage("> " + txt)
				}
				inputHistory = append(inputHistory, txt)
			}
			inputActive = false
			inputText = inputText[:0]
			historyPos = len(inputHistory)
			changedInput = true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			inputActive = false
			inputText = inputText[:0]
			historyPos = len(inputHistory)
			changedInput = true
		}
	} else {
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			inputActive = true
			inputText = inputText[:0]
			historyPos = len(inputHistory)
			changedInput = true
		}
	}

	if changedInput {
		updateConsoleWindow()
		if consoleWin != nil {
			consoleWin.Refresh()
		}
	}

	/* WASD / ARROWS */

	var keyWalk bool
	if !inputActive {
		dx, dy := 0, 0
		if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) {
			dx--
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) {
			dx++
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) {
			dy--
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS) {
			dy++
		}
		if dx != 0 || dy != 0 {
			keyWalk = true
			speed := gs.KBWalkSpeed
			if ebiten.IsKeyPressed(ebiten.KeyShift) {
				speed = 1.0
			}
			keyX = int16(float64(dx) * float64(fieldCenterX) * speed)
			keyY = int16(float64(dy) * float64(fieldCenterY) * speed)
		} else {
			keyWalk = false
		}
	}

	mx, my := ebiten.CursorPosition()
	// Map mouse to world coordinates accounting for current draw scale/offset.
	origX, origY, worldScale := worldDrawInfo()
	baseX := int16(float64(mx-origX)/worldScale - float64(fieldCenterX))
	baseY := int16(float64(my-origY)/worldScale - float64(fieldCenterY))
	heldTime := inpututil.MouseButtonPressDuration(ebiten.MouseButtonLeft)
	click := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)

	if click && pointInUI(mx, my) {
		uiMouseDown = true
	}
	if uiMouseDown {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			uiMouseDown = false
		} else {
			click = false
			heldTime = 0
		}
	}

	x, y := baseX, baseY
	walk := false
	if !uiMouseDown {
		if keyWalk {
			x, y, walk = keyX, keyY, true
			walkToggled = false
		} else if gs.ClickToToggle && click {
			walkToggled = !walkToggled
			walk = walkToggled
		} else if !gs.ClickToToggle && heldTime > 1 && !click {
			walk = true
			walkToggled = false
		}

		if gs.ClickToToggle && walkToggled {
			walk = walkToggled
		}
	}

	/* Change Cursor */
	if walk && !keyWalk {
		ebiten.SetCursorShape(ebiten.CursorShapeCrosshair)
	} else {
		ebiten.SetCursorShape(ebiten.CursorShapeDefault)
	}

	inputMu.Lock()
	latestInput = inputState{mouseX: x, mouseY: y, mouseDown: walk}
	inputMu.Unlock()

	return nil
}

func updateGameWindowSize() {
	if gameWin == nil {
		return
	}
	size := gameWin.GetRawSize()
	desiredW := int(math.Round(float64(size.X)))
	desiredH := int(math.Round(float64(size.Y)))
	gameWin.SetSize(eui.Point{X: float32(desiredW), Y: float32(desiredH)})
}

func gameWindowOrigin() (int, int) {
	if gameWin == nil {
		return 0, 0
	}
	pos := gameWin.GetRawPos()
	frame := gameWin.Margin + gameWin.Border + gameWin.BorderPad + gameWin.Padding
	x := pos.X + frame
	y := pos.Y + frame + gameWin.GetRawTitleSize()
	return int(x), int(y)
}

// worldDrawInfo reports the on-screen origin (top-left) of the rendered world
// inside the game window, and the effective scale in pixels per world unit.
// This matches the draw-time composition logic so input stays aligned even
// when the window size or aspect ratio changes.
func worldDrawInfo() (int, int, float64) {
	gx, gy := gameWindowOrigin()
	if gameWin == nil {
		// Fallback to current game scale with no offset.
		if gs.GameScale <= 0 {
			return gx, gy, 1.0
		}
		return gx, gy, gs.GameScale
	}

	// Derive the inner content buffer size used for the game image.
	size := gameWin.GetSize()
	pad := float64(2 * gameWin.Padding)
	cw := int(float64(int(size.X)&^1) - pad) // content width
	ch := int(float64(int(size.Y)&^1) - pad) // content height
	// Leave a 2px margin on all sides (matches gameImageItem.Position and sizing).
	bufW := cw - 4
	bufH := ch - 4
	if bufW <= 0 || bufH <= 0 {
		if gs.GameScale <= 0 {
			return gx, gy, 1.0
		}
		return gx, gy, gs.GameScale
	}

	// Match Draw() scaling rules.
	const maxSuperSampleScale = 4
	worldW, worldH := gameAreaSizeX, gameAreaSizeY

	// Slider-desired scale.
	desired := int(math.Round(gs.GameScale))
	if desired < 1 {
		desired = 1
	}
	if desired > 10 {
		desired = 10
	}

	// Max integer fit into current buffer.
	fit := int(math.Floor(math.Min(float64(bufW)/float64(worldW), float64(bufH)/float64(worldH))))
	if fit < 1 {
		fit = 1
	}

	offIntScale := fit
	if desired > offIntScale {
		offIntScale = desired
	}
	if offIntScale > maxSuperSampleScale {
		offIntScale = maxSuperSampleScale
	}
	if offIntScale < 1 {
		offIntScale = 1
	}

	offW := worldW * offIntScale
	offH := worldH * offIntScale

	scaleDown := math.Min(float64(bufW)/float64(offW), float64(bufH)/float64(offH))

	drawW := float64(offW) * scaleDown
	drawH := float64(offH) * scaleDown
	tx := (float64(bufW) - drawW) / 2
	ty := (float64(bufH) - drawH) / 2

	// Add the 2px inner margin to the window origin to reach the game image.
	originX := gx + 2 + int(math.Round(tx))
	originY := gy + 2 + int(math.Round(ty))
	// Effective world scale on screen in pixels per world unit.
	effScale := float64(offIntScale) * scaleDown
	if effScale <= 0 {
		effScale = 1.0
	}
	return originX, originY, effScale
}

func (g *Game) Draw(screen *ebiten.Image) {

	//Reduce render load while seeking clMov
	if seekingMov {
		if time.Since(lastSeekPrev) < time.Millisecond*200 {
			return
		}
		lastSeekPrev = time.Now()
		gameImageItem.Disabled = true
	} else {
		gameImageItem.Disabled = false
	}
	if backgroundImg != nil {
		drawBackground(screen)
	} else {
		screen.Fill(dimmedScreenBG)
	}

	// Ensure the game image item/buffer exists and matches window content.
	updateGameImageSize()
	if gameImage == nil {
		// UI not ready yet
		eui.Draw(screen)
		return
	}

	// Determine offscreen render scale and composite scale.
	// A user-selected render scale (gs.GameScale) in 1x..10x acts as a
	// supersample factor. The window is always filled using linear filtering.
	bufW := gameImage.Bounds().Dx()
	bufH := gameImage.Bounds().Dy()
	const maxSuperSampleScale = 4
	worldW, worldH := gameAreaSizeX, gameAreaSizeY

	// Clamp desired render scale from settings (treat as integer steps)
	desired := int(math.Round(gs.GameScale))
	if desired < 1 {
		desired = 1
	}
	if desired > 10 {
		desired = 10
	}
	// Maximum scale that fits the current buffer without clipping
	fit := int(math.Floor(math.Min(float64(bufW)/float64(worldW), float64(bufH)/float64(worldH))))
	if fit < 1 {
		fit = 1
	}

	offIntScale := fit
	if desired > offIntScale {
		offIntScale = desired
	}
	if offIntScale > maxSuperSampleScale {
		offIntScale = maxSuperSampleScale
	}
	if offIntScale < 1 {
		offIntScale = 1
	}

	// Prepare variable-sized offscreen target (supersampled)
	offW := worldW * offIntScale
	offH := worldH * offIntScale
	ensureWorldRT(offW, offH)
	worldRT.Clear()

	// Render splash or live frame into worldRT using the offscreen scale
	var snap drawSnapshot
	var alpha float64
	var haveSnap bool
	if clmov == "" && tcpConn == nil && pcapPath == "" && !fake {
		prev := gs.GameScale
		gs.GameScale = float64(offIntScale)
		drawSplash(worldRT, 0, 0)
		gs.GameScale = prev
	} else {
		snap = captureDrawSnapshot()
		var mobileFade, pictFade float32
		alpha, mobileFade, pictFade = computeInterpolation(snap.prevTime, snap.curTime, gs.MobileBlendAmount, gs.BlendAmount)
		prev := gs.GameScale
		gs.GameScale = float64(offIntScale)
		drawScene(worldRT, 0, 0, snap, alpha, mobileFade, pictFade)
		if gs.nightEffect {
			drawNightOverlay(worldRT, 0, 0)
		}
		drawStatusBars(worldRT, 0, 0, snap, alpha)
		gs.GameScale = prev
		haveSnap = true
	}

	// Composite worldRT into the gameImage buffer: scale/center
	gameImage.Clear()
	scaleDown := math.Min(float64(bufW)/float64(offW), float64(bufH)/float64(offH))
	drawW := float64(offW) * scaleDown
	drawH := float64(offH) * scaleDown
	tx := (float64(bufW) - drawW) / 2
	ty := (float64(bufH) - drawH) / 2
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear, DisableMipmaps: true}
	op.GeoM.Scale(scaleDown, scaleDown)
	op.GeoM.Translate(tx, ty)
	gameImage.DrawImage(worldRT, op)
	if haveSnap {
		prev := gs.GameScale
		finalScale := float64(offIntScale) * scaleDown
		gs.GameScale = finalScale
		left := roundToInt(tx)
		top := roundToInt(ty)
		right := left + roundToInt(drawW)
		bottom := top + roundToInt(drawH)
		if right > bufW {
			right = bufW
		}
		if bottom > bufH {
			bottom = bufH
		}
		worldView := gameImage.SubImage(image.Rect(left, top, right, bottom)).(*ebiten.Image)
		drawMobileNameTags(worldView, snap, alpha)
		drawSpeechBubbles(worldView, snap, alpha)
		gs.GameScale = prev
	}

	// Finally, draw UI (which includes the game window image)
	eui.Draw(screen)
	if gs.ShowFPS {
		drawServerFPS(screen, screen.Bounds().Dx()-40, 4, serverFPS)
	}

	if seekingMov {
		x, y := float64(screen.Bounds().Dx())/2, float64(screen.Bounds().Dy())/2
		vector.DrawFilledRect(screen, float32(x+2), float32(y+2), 90, 40, color.Black, false)

		op := &text.DrawOptions{}
		op.GeoM.Translate(x, y)
		text.Draw(screen, "SEEKING...", mainFontBold, op)
	}
}

var lastSeekPrev time.Time

// drawScene renders all world objects for the current frame.
func drawScene(screen *ebiten.Image, ox, oy int, snap drawSnapshot, alpha float64, mobileFade, pictFade float32) {

	// Use cached descriptor map directly; no need to rebuild/sort it per frame.
	descMap := snap.descriptors

	// Use precomputed, sorted partitions
	negPics := snap.picsNeg
	zeroPics := snap.picsZero
	posPics := snap.picsPos
	live := snap.liveMobs
	dead := snap.deadMobs

	for _, p := range negPics {
		drawPicture(screen, ox, oy, p, alpha, pictFade, snap.mobiles, descMap, snap.prevMobiles, snap.picShiftX, snap.picShiftY)
	}

	if gs.hideMobiles {
		for _, p := range zeroPics {
			drawPicture(screen, ox, oy, p, alpha, pictFade, snap.mobiles, descMap, snap.prevMobiles, snap.picShiftX, snap.picShiftY)
		}
	} else {
		for _, m := range dead {
			drawMobile(screen, ox, oy, m, descMap, snap.prevMobiles, snap.prevDescs, snap.picShiftX, snap.picShiftY, alpha, mobileFade)
		}
		i, j := 0, 0
		maxInt := int(^uint(0) >> 1)
		for i < len(live) || j < len(zeroPics) {
			mV, mH := maxInt, maxInt
			if i < len(live) {
				mV = int(live[i].V)
				mH = int(live[i].H)
			}
			pV, pH := maxInt, maxInt
			if j < len(zeroPics) {
				pV = int(zeroPics[j].V)
				pH = int(zeroPics[j].H)
			}
			if mV < pV || (mV == pV && mH <= pH) {
				if live[i].State != poseDead {
					drawMobile(screen, ox, oy, live[i], descMap, snap.prevMobiles, snap.prevDescs, snap.picShiftX, snap.picShiftY, alpha, mobileFade)
				}
				i++
			} else {
				drawPicture(screen, ox, oy, zeroPics[j], alpha, pictFade, snap.mobiles, descMap, snap.prevMobiles, snap.picShiftX, snap.picShiftY)
				j++
			}
		}
	}

	for _, p := range posPics {
		drawPicture(screen, ox, oy, p, alpha, pictFade, snap.mobiles, descMap, snap.prevMobiles, snap.picShiftX, snap.picShiftY)
	}
}

// drawMobile renders a single mobile object with optional interpolation and onion skinning.
func drawMobile(screen *ebiten.Image, ox, oy int, m frameMobile, descMap map[uint8]frameDescriptor, prevMobiles map[uint8]frameMobile, prevDescs map[uint8]frameDescriptor, shiftX, shiftY int, alpha float64, fade float32) {
	h := float64(m.H)
	v := float64(m.V)
	if gs.MotionSmoothing {
		if pm, ok := prevMobiles[m.Index]; ok {
			dh := int(m.H) - int(pm.H) - shiftX
			dv := int(m.V) - int(pm.V) - shiftY
			if dh*dh+dv*dv <= maxMobileInterpPixels*maxMobileInterpPixels {
				h = float64(pm.H)*(1-alpha) + float64(m.H)*alpha
				v = float64(pm.V)*(1-alpha) + float64(m.V)*alpha
			}
		}
	}
	x := roundToInt((h + float64(fieldCenterX)) * gs.GameScale)
	y := roundToInt((v + float64(fieldCenterY)) * gs.GameScale)
	x += ox
	y += oy
	var img *ebiten.Image
	plane := 0
	var d frameDescriptor
	var colors []byte
	var state uint8
	if desc, ok := descMap[m.Index]; ok {
		d = desc
		colors = d.Colors
		playersMu.RLock()
		if p, ok := players[d.Name]; ok && len(p.Colors) > 0 {
			colors = append([]byte(nil), p.Colors...)
		}
		playersMu.RUnlock()
		state = m.State
		img = loadMobileFrame(d.PictID, state, colors)
		plane = d.Plane
	}
	var prevImg *ebiten.Image
	var prevColors []byte
	var prevPict uint16
	var prevState uint8
	if gs.BlendMobiles {
		if pm, ok := prevMobiles[m.Index]; ok {
			pd := descMap[m.Index]
			if d, ok := prevDescs[m.Index]; ok {
				pd = d
			}
			prevColors = pd.Colors
			playersMu.RLock()
			if p, ok := players[pd.Name]; ok && len(p.Colors) > 0 {
				prevColors = append([]byte(nil), p.Colors...)
			}
			playersMu.RUnlock()
			prevImg = loadMobileFrame(pd.PictID, pm.State, prevColors)
			prevPict = pd.PictID
			prevState = pm.State
		}
	}
	if img != nil {
		size := img.Bounds().Dx()
		blend := gs.BlendMobiles && prevImg != nil && fade > 0 && fade < 1
		var src *ebiten.Image
		drawSize := size
		if blend {
			steps := gs.MobileBlendFrames
			idx := int(fade * float32(steps))
			if idx <= 0 {
				idx = 1
			}
			if idx >= steps {
				idx = steps - 1
			}
			prevKey := makeMobileKey(prevPict, prevState, prevColors)
			curKey := makeMobileKey(d.PictID, state, colors)
			if b := mobileBlendFrame(prevKey, curKey, prevImg, img, idx, steps); b != nil {
				src = b
				drawSize = b.Bounds().Dx()
			} else {
				src = img
			}
		} else if gs.BlendMobiles && prevImg != nil {
			if fade <= 0 {
				src = prevImg
				drawSize = prevImg.Bounds().Dx()
			} else {
				src = img
			}
		} else {
			src = img
		}
		scale := gs.GameScale
		scaled := float64(roundToInt(float64(drawSize) * scale))
		scale = scaled / float64(drawSize)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest, DisableMipmaps: true}
		op.GeoM.Scale(scale, scale)
		tx := float64(x) - scaled/2
		ty := float64(y) - scaled/2
		op.GeoM.Translate(tx, ty)
		screen.DrawImage(src, op)
		if gs.imgPlanesDebug {
			metrics := mainFont.Metrics()
			lbl := fmt.Sprintf("%dm", plane)
			xPos := x - int(float64(size)*gs.GameScale/2)
			op := &text.DrawOptions{}
			op.GeoM.Translate(float64(xPos), float64(y)-float64(size)*gs.GameScale/2-metrics.HAscent)
			op.ColorScale.ScaleWithColor(color.RGBA{0, 255, 255, 255})
			text.Draw(screen, lbl, mainFont, op)
		}
	} else {
		// Fallback marker when image missing; no per-frame bounds check.
		vector.DrawFilledRect(screen, float32(float64(x)-3*gs.GameScale), float32(float64(y)-3*gs.GameScale), float32(6*gs.GameScale), float32(6*gs.GameScale), color.RGBA{0xff, 0, 0, 0xff}, false)
		if gs.imgPlanesDebug {
			metrics := mainFont.Metrics()
			lbl := fmt.Sprintf("%dm", plane)
			xPos := x - int(3*gs.GameScale)
			op := &text.DrawOptions{}
			op.GeoM.Translate(float64(xPos), float64(y)-3*gs.GameScale-metrics.HAscent)
			op.ColorScale.ScaleWithColor(color.White)
			text.Draw(screen, lbl, mainFont, op)
		}
	}
}

// drawPicture renders a single picture sprite.
func drawPicture(screen *ebiten.Image, ox, oy int, p framePicture, alpha float64, fade float32, mobiles []frameMobile, descMap map[uint8]frameDescriptor, prevMobiles map[uint8]frameMobile, shiftX, shiftY int) {
	if gs.hideMoving && p.Moving {
		return
	}
	offX := float64(int(p.PrevH)-int(p.H)) * (1 - alpha)
	offY := float64(int(p.PrevV)-int(p.V)) * (1 - alpha)
	if p.Moving && !gs.smoothMoving {
		if int(p.PrevH) == int(p.H)-shiftX && int(p.PrevV) == int(p.V)-shiftY {
			if gs.dontShiftNewSprites {
				offX = 0
				offY = 0
			}
		} else {
			offX = 0
			offY = 0
		}
	}

	frame := 0
	if clImages != nil {
		frame = clImages.FrameIndex(uint32(p.PictID), frameCounter)
	}
	plane := p.Plane

	w, h := 0, 0
	if clImages != nil {
		w, h = clImages.Size(uint32(p.PictID))
	}

	var mobileX, mobileY float64
	if w <= 64 && h <= 64 && gs.MotionSmoothing && gs.smoothMoving {
		if dx, dy, ok := pictureMobileOffset(p, mobiles, prevMobiles, alpha, shiftX, shiftY); ok {
			mobileX, mobileY = dx, dy
			offX = 0
			offY = 0
		}
	}

	x := roundToInt(((float64(p.H) + offX + mobileX) + float64(fieldCenterX)) * gs.GameScale)
	y := roundToInt(((float64(p.V) + offY + mobileY) + float64(fieldCenterY)) * gs.GameScale)
	x += ox
	y += oy

	img := loadImageFrame(p.PictID, frame)
	var prevImg *ebiten.Image
	var prevFrame int
	if gs.BlendPicts && clImages != nil {
		prevFrame = clImages.FrameIndex(uint32(p.PictID), frameCounter-1)
		if prevFrame != frame {
			prevImg = loadImageFrame(p.PictID, prevFrame)
		}
	}

	if img != nil {
		drawW, drawH := w, h
		blend := gs.BlendPicts && prevImg != nil && fade > 0 && fade < 1
		var src *ebiten.Image
		if blend {
			steps := gs.PictBlendFrames
			idx := int(fade * float32(steps))
			if idx <= 0 {
				idx = 1
			}
			if idx >= steps {
				idx = steps - 1
			}
			if b := pictBlendFrame(p.PictID, prevFrame, frame, prevImg, img, idx, steps); b != nil {
				src = b
			} else {
				src = img
				blend = false
			}
		} else if gs.BlendPicts && prevImg != nil {
			if fade <= 0 {
				src = prevImg
			} else {
				src = img
			}
		} else {
			src = img
		}
		if src != nil {
			drawW, drawH = src.Bounds().Dx(), src.Bounds().Dy()
		}
		sx, sy := scaleForFiltering(gs.GameScale, drawW, drawH)
		scaledW := float64(roundToInt(float64(drawW) * sx))
		scaledH := float64(roundToInt(float64(drawH) * sy))
		sx = scaledW / float64(drawW)
		sy = scaledH / float64(drawH)
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest, DisableMipmaps: true}
		op.GeoM.Scale(sx, sy)
		tx := float64(x) - scaledW/2
		ty := float64(y) - scaledH/2
		op.GeoM.Translate(tx, ty)
		if gs.pictAgainDebug && p.Again {
			op.ColorScale.Scale(0, 0, 1, 1)
		} else if src == img && gs.smoothingDebug && p.Moving {
			op.ColorScale.Scale(1, 0, 0, 1)
		}
		screen.DrawImage(src, op)

		if gs.pictIDDebug {
			metrics := mainFont.Metrics()
			lbl := fmt.Sprintf("%d", p.PictID)
			txtW, _ := text.Measure(lbl, mainFont, 0)
			xPos := x + int(float64(w)*gs.GameScale/2) - roundToInt(txtW)
			opTxt := &text.DrawOptions{}
			opTxt.GeoM.Translate(float64(xPos), float64(y)-float64(h)*gs.GameScale/2-metrics.HAscent)
			opTxt.ColorScale.ScaleWithColor(color.Black)
			text.Draw(screen, lbl, mainFont, opTxt)
		}

		if gs.imgPlanesDebug {
			metrics := mainFont.Metrics()
			lbl := fmt.Sprintf("%dp", plane)
			xPos := x - int(float64(w)*gs.GameScale/2)
			opTxt := &text.DrawOptions{}
			opTxt.GeoM.Translate(float64(xPos), float64(y)-float64(h)*gs.GameScale/2-metrics.HAscent)
			opTxt.ColorScale.ScaleWithColor(color.RGBA{255, 255, 0, 0})
			text.Draw(screen, lbl, mainFont, opTxt)
		}
	} else {
		clr := color.RGBA{0, 0, 0xff, 0xff}
		if gs.smoothingDebug && p.Moving {
			clr = color.RGBA{0xff, 0, 0, 0xff}
		}
		if gs.pictAgainDebug && p.Again {
			clr = color.RGBA{0, 0, 0xff, 0xff}
		}
		vector.DrawFilledRect(screen, float32(float64(x)-2*gs.GameScale), float32(float64(y)-2*gs.GameScale), float32(4*gs.GameScale), float32(4*gs.GameScale), clr, false)
		if gs.pictIDDebug {
			metrics := mainFont.Metrics()
			lbl := fmt.Sprintf("%d", p.PictID)
			txtW, _ := text.Measure(lbl, mainFont, 0)
			half := int(2 * gs.GameScale)
			xPos := x + half - roundToInt(txtW)
			opTxt := &text.DrawOptions{}
			opTxt.GeoM.Translate(float64(xPos), float64(y)-float64(half)-metrics.HAscent)
			opTxt.ColorScale.ScaleWithColor(color.RGBA{R: 1, A: 1})
			text.Draw(screen, lbl, mainFont, opTxt)
		}
		if gs.imgPlanesDebug {
			metrics := mainFont.Metrics()
			lbl := fmt.Sprintf("%dp", plane)
			xPos := x - int(2*gs.GameScale)
			opTxt := &text.DrawOptions{}
			opTxt.GeoM.Translate(float64(xPos), float64(y)-2*gs.GameScale-metrics.HAscent)
			opTxt.ColorScale.ScaleWithColor(color.RGBA{255, 255, 0, 0})
			text.Draw(screen, lbl, mainFont, opTxt)
		}
	}
}

// pictureMobileOffset returns the interpolated offset for a picture that
// aligns with a mobile which moved between frames.
func pictureMobileOffset(p framePicture, mobiles []frameMobile, prevMobiles map[uint8]frameMobile, alpha float64, shiftX, shiftY int) (float64, float64, bool) {
	for _, m := range mobiles {
		if m.H == p.H && m.V == p.V {
			if pm, ok := prevMobiles[m.Index]; ok {
				dh := int(m.H) - int(pm.H) - shiftX
				dv := int(m.V) - int(pm.V) - shiftY
				if dh != 0 || dv != 0 {
					if dh*dh+dv*dv <= maxMobileInterpPixels*maxMobileInterpPixels {
						h := float64(pm.H)*(1-alpha) + float64(m.H)*alpha
						v := float64(pm.V)*(1-alpha) + float64(m.V)*alpha
						return h - float64(m.H), v - float64(m.V), true
					}
				}
			}
			break
		}
	}
	return 0, 0, false
}

// drawMobileNameTags renders mobile name tags and color bars either at native
// resolution or scaled with the game world.
func drawMobileNameTags(screen *ebiten.Image, snap drawSnapshot, alpha float64) {
	if gs.hideMobiles {
		return
	}
	descMap := snap.descriptors
	for _, m := range snap.mobiles {
		h := float64(m.H)
		v := float64(m.V)
		if gs.MotionSmoothing {
			if pm, ok := snap.prevMobiles[m.Index]; ok {
				dh := int(m.H) - int(pm.H) - snap.picShiftX
				dv := int(m.V) - int(pm.V) - snap.picShiftY
				if dh*dh+dv*dv <= maxMobileInterpPixels*maxMobileInterpPixels {
					h = float64(pm.H)*(1-alpha) + float64(m.H)*alpha
					v = float64(pm.V)*(1-alpha) + float64(m.V)*alpha
				}
			}
		}
		x := roundToInt((h + float64(fieldCenterX)) * gs.GameScale)
		y := roundToInt((v + float64(fieldCenterY)) * gs.GameScale)
		if d, ok := descMap[m.Index]; ok {
			nameAlpha := uint8(gs.NameBgOpacity * 255)
			if d.Name != "" {
				style := styleRegular
				playersMu.RLock()
				if p, ok := players[d.Name]; ok {
					if p.Sharing && p.Sharee {
						style = styleBoldItalic
					} else if p.Sharing {
						style = styleBold
					} else if p.Sharee {
						style = styleItalic
					}
				}
				playersMu.RUnlock()
				if m.nameTag != nil && m.nameTagKey.FontGen == fontGen && m.nameTagKey.Opacity == nameAlpha && m.nameTagKey.Text == d.Name && m.nameTagKey.Colors == m.Colors && m.nameTagKey.Style == style {
					scale := 1.0
					if !gs.nameTagsNative {
						scale = gs.GameScale
					}
					top := y + int(20*gs.GameScale)
					left := x - int(float64(m.nameTagW)/2*scale)
					op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest, DisableMipmaps: true}
					op.GeoM.Scale(scale, scale)
					op.GeoM.Translate(float64(left), float64(top))
					screen.DrawImage(m.nameTag, op)
				} else {
					textClr, bgClr, frameClr := mobileNameColors(m.Colors)
					bgClr.A = nameAlpha
					frameClr.A = nameAlpha
					face := mainFont
					switch style {
					case styleBold:
						face = mainFontBold
					case styleItalic:
						face = mainFontItalic
					case styleBoldItalic:
						face = mainFontBoldItalic
					}
					w, h := text.Measure(d.Name, face, 0)
					iw := int(math.Ceil(w))
					ih := int(math.Ceil(h))
					scale := 1.0
					if !gs.nameTagsNative {
						scale = gs.GameScale
					}
					top := y + int(20*gs.GameScale)
					left := x - int(float64(iw)/2*scale)
					op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest, DisableMipmaps: true}
					op.GeoM.Scale(float64(iw+5)*scale, float64(ih)*scale)
					op.GeoM.Translate(float64(left), float64(top))
					op.ColorScale.ScaleWithColor(bgClr)
					screen.DrawImage(whiteImage, op)
					vector.StrokeRect(screen, float32(left), float32(top), float32(iw+5)*float32(scale), float32(ih)*float32(scale), 1, frameClr, false)
					opTxt := &text.DrawOptions{}
					opTxt.GeoM.Scale(scale, scale)
					opTxt.GeoM.Translate(float64(left)+2*scale, float64(top)+2*scale)
					opTxt.ColorScale.ScaleWithColor(textClr)
					text.Draw(screen, d.Name, face, opTxt)
				}
			} else {
				back := int((m.Colors >> 4) & 0x0f)
				if back != kColorCodeBackWhite && back != kColorCodeBackBlue && !(back == kColorCodeBackBlack && d.Type == kDescMonster) {
					if back >= len(nameBackColors) {
						back = 0
					}
					barClr := nameBackColors[back]
					barClr.A = nameAlpha
					size := mobileSize(d.PictID)
					top := y + int(float64(size)*gs.GameScale/2+2*gs.GameScale)
					left := x - int(6*gs.GameScale)
					op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest, DisableMipmaps: true}
					op.GeoM.Scale(12*gs.GameScale, 2*gs.GameScale)
					op.GeoM.Translate(float64(left), float64(top))
					op.ColorScale.ScaleWithColor(barClr)
					screen.DrawImage(whiteImage, op)
				}
			}
		}
	}
}

// drawSpeechBubbles renders speech bubbles at native resolution.
func drawSpeechBubbles(screen *ebiten.Image, snap drawSnapshot, alpha float64) {
	if !gs.SpeechBubbles {
		return
	}
	descMap := snap.descriptors
	for _, b := range snap.bubbles {
		bubbleType := b.Type & kBubbleTypeMask
		typeOK := true
		switch bubbleType {
		case kBubbleNormal:
			typeOK = gs.BubbleNormal
		case kBubbleWhisper:
			typeOK = gs.BubbleWhisper
		case kBubbleYell:
			typeOK = gs.BubbleYell
		case kBubbleThought:
			typeOK = gs.BubbleThought
		case kBubbleRealAction:
			typeOK = gs.BubbleRealAction
		case kBubbleMonster:
			typeOK = gs.BubbleMonster
		case kBubblePlayerAction:
			typeOK = gs.BubblePlayerAction
		case kBubblePonder:
			typeOK = gs.BubblePonder
		case kBubbleNarrate:
			typeOK = gs.BubbleNarrate
		}
		originOK := true
		switch {
		case b.Index == playerIndex:
			originOK = gs.BubbleSelf
		case bubbleType == kBubbleMonster:
			originOK = gs.BubbleMonsters
		case bubbleType == kBubbleNarrate:
			originOK = gs.BubbleNarration
		default:
			originOK = gs.BubbleOtherPlayers
		}
		if !(typeOK && originOK) {
			continue
		}
		hpos := float64(b.H)
		vpos := float64(b.V)
		if !b.Far {
			var m *frameMobile
			for i := range snap.mobiles {
				if snap.mobiles[i].Index == b.Index {
					m = &snap.mobiles[i]
					break
				}
			}
			if m != nil {
				hpos = float64(m.H)
				vpos = float64(m.V)
				if gs.MotionSmoothing {
					if pm, ok := snap.prevMobiles[b.Index]; ok {
						dh := int(m.H) - int(pm.H) - snap.picShiftX
						dv := int(m.V) - int(pm.V) - snap.picShiftY
						if dh*dh+dv*dv <= maxMobileInterpPixels*maxMobileInterpPixels {
							hpos = float64(pm.H)*(1-alpha) + float64(m.H)*alpha
							vpos = float64(pm.V)*(1-alpha) + float64(m.V)*alpha
						}
					}
				}
			}
		}
		x := roundToInt((hpos + float64(fieldCenterX)) * gs.GameScale)
		y := roundToInt((vpos + float64(fieldCenterY)) * gs.GameScale)
		if !b.Far {
			if d, ok := descMap[b.Index]; ok {
				if size := mobileSize(d.PictID); size > 0 {
					tailHeight := int(10 * gs.GameScale)
					y += tailHeight - int(math.Round(float64(size)*gs.GameScale))
				}
			}
		}
		borderCol, bgCol, textCol := bubbleColors(b.Type)
		drawBubble(screen, b.Text, x, y, b.Type, b.Far, b.NoArrow, borderCol, bgCol, textCol)
	}
}

// lerpBar interpolates status bar values, skipping interpolation when the
// current value is lower than the previous.
func lerpBar(prev, cur int, alpha float64) int {
	if cur < prev {
		return cur
	}
	return int(math.Round(float64(prev) + alpha*float64(cur-prev)))
}

// drawStatusBars renders health, balance and spirit bars.
func drawStatusBars(screen *ebiten.Image, ox, oy int, snap drawSnapshot, alpha float64) {
	drawRect := func(x, y, w, h int, clr color.RGBA) {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterNearest, DisableMipmaps: true}
		op.GeoM.Scale(float64(w), float64(h))
		op.GeoM.Translate(float64(ox+x), float64(oy+y))
		op.ColorScale.ScaleWithColor(clr)
		op.ColorScale.ScaleAlpha(float32(gs.BarOpacity))
		screen.DrawImage(whiteImage, op)
	}
	barWidth := int(110 * gs.GameScale)
	barHeight := int(8 * gs.GameScale)

	fieldWidth := int(float64(gameAreaSizeX) * gs.GameScale)
	fieldHeight := int(float64(gameAreaSizeY) * gs.GameScale)

	var x, y, dx, dy int
	switch gs.BarPlacement {
	case BarPlacementLowerLeft:
		x = int(20 * gs.GameScale)
		spacing := int(4 * gs.GameScale)
		y = fieldHeight - int(20*gs.GameScale) - 3*barHeight - 2*spacing
		dx = 0
		dy = barHeight + spacing
	case BarPlacementLowerRight:
		x = fieldWidth - int(20*gs.GameScale) - barWidth
		spacing := int(4 * gs.GameScale)
		y = fieldHeight - int(20*gs.GameScale) - 3*barHeight - 2*spacing
		dx = 0
		dy = barHeight + spacing
	case BarPlacementUpperRight:
		x = fieldWidth - int(20*gs.GameScale) - barWidth
		spacing := int(4 * gs.GameScale)
		y = int(20 * gs.GameScale)
		dx = 0
		dy = barHeight + spacing
	default: // BarPlacementBottom
		slot := (fieldWidth - 3*barWidth) / 6
		x = slot
		y = fieldHeight - int(20*gs.GameScale) - barHeight
		dx = barWidth + 2*slot
		dy = 0
	}

	screenW := screen.Bounds().Dx()
	screenH := screen.Bounds().Dy()
	minX := -ox
	minY := -oy
	maxX := screenW - ox - barWidth - 2*dx
	maxY := screenH - oy - barHeight - 2*dy
	if x < minX {
		x = minX
	} else if x > maxX {
		x = maxX
	}
	if y < minY {
		y = minY
	} else if y > maxY {
		y = maxY
	}

	drawBar := func(x, y int, cur, max int, clr color.RGBA) {
		alpha := uint8(255)
		frameClr := color.RGBA{0xff, 0xff, 0xff, alpha}
		pad := int(gs.GameScale)
		drawRect(x-pad, y-pad, barWidth+2*pad, pad, frameClr)
		drawRect(x-pad, y+barHeight, barWidth+2*pad, pad, frameClr)
		drawRect(x-pad, y, pad, barHeight, frameClr)
		drawRect(x+barWidth, y, pad, barHeight, frameClr)
		if max > 0 && cur > 0 {
			w := barWidth * cur / max
			base := clr
			if gs.BarColorByValue {
				ratio := float64(cur) / float64(max)
				switch {
				case ratio <= 0.33:
					base = color.RGBA{0xff, 0x00, 0x00, 0xff}
				case ratio <= 0.66:
					base = color.RGBA{0xff, 0xff, 0x00, 0xff}
				default:
					base = color.RGBA{0x00, 0xff, 0x00, 0xff}
				}
			}
			fillClr := color.RGBA{base.R, base.G, base.B, alpha}
			drawRect(x, y, w, barHeight, fillClr)
		}
	}

	hp := lerpBar(snap.prevHP, snap.hp, alpha)
	hpMax := lerpBar(snap.prevHPMax, snap.hpMax, alpha)
	drawBar(x, y, hp, hpMax, color.RGBA{0x00, 0xff, 0, 0xff})
	x += dx
	y += dy
	bal := lerpBar(snap.prevBalance, snap.balance, alpha)
	balMax := lerpBar(snap.prevBalanceMax, snap.balanceMax, alpha)
	drawBar(x, y, bal, balMax, color.RGBA{0x00, 0x00, 0xff, 0xff})
	x += dx
	y += dy
	sp := lerpBar(snap.prevSP, snap.sp, alpha)
	spMax := lerpBar(snap.prevSPMax, snap.spMax, alpha)
	drawBar(x, y, sp, spMax, color.RGBA{0xff, 0x00, 0x00, 0xff})
}

var fpsImage *ebiten.Image
var lastFPS time.Time
var fpsWidth, fpsHeight float64

func drawServerFPS(screen *ebiten.Image, ox, oy int, fps float64) {
	if fps <= 0 {
		return
	}
	if time.Since(lastFPS) >= time.Second {
		lastFPS = time.Now()

		lat := netLatency
		msg := fmt.Sprintf("FPS: %0.2f Server: %0.2f Ping: %-3v ms", ebiten.ActualFPS(), fps, lat.Milliseconds())
		w, h := text.Measure(msg, mainFont, 0)

		if fpsImage == nil || fpsHeight != h {
			logDebug("Allocated FPS image.")
			fpsImage = newImage(int(w*1.3), int(h))
		}

		fpsImage.Clear()
		text.Draw(fpsImage, msg, mainFont, &text.DrawOptions{})
		fpsWidth, fpsHeight = w, h
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(ox)-fpsWidth, float64(oy))
	screen.DrawImage(fpsImage, op)

}

// equippedItemPicts returns pict IDs for items equipped in right and left hands.
func equippedItemPicts() (uint16, uint16) {
	items := getInventory()
	var rightID, leftID uint16
	var bothIDRight, bothIDLeft uint16
	if clImages != nil {
		for _, it := range items {
			if !it.Equipped {
				continue
			}
			slot := clImages.ItemSlot(uint32(it.ID))
			switch slot {
			case kItemSlotRightHand:
				if id := clImages.ItemRightHandPict(uint32(it.ID)); id != 0 {
					rightID = uint16(id)
				} else if id := clImages.ItemWornPict(uint32(it.ID)); id != 0 {
					rightID = uint16(id)
				}
			case kItemSlotLeftHand:
				if id := clImages.ItemLeftHandPict(uint32(it.ID)); id != 0 {
					leftID = uint16(id)
				} else if id := clImages.ItemWornPict(uint32(it.ID)); id != 0 {
					leftID = uint16(id)
				}
			case kItemSlotBothHands:
				if id := clImages.ItemRightHandPict(uint32(it.ID)); id != 0 {
					bothIDRight = uint16(id)
				} else if id := clImages.ItemWornPict(uint32(it.ID)); id != 0 {
					bothIDRight = uint16(id)
				}
				if id := clImages.ItemLeftHandPict(uint32(it.ID)); id != 0 {
					bothIDLeft = uint16(id)
				} else if id := clImages.ItemWornPict(uint32(it.ID)); id != 0 {
					bothIDLeft = uint16(id)
				}
			}
		}
	}
	if rightID == 0 && leftID == 0 {
		if bothIDRight != 0 || bothIDLeft != 0 {
			if rightID == 0 {
				rightID = bothIDRight
				if rightID == 0 {
					rightID = bothIDLeft
				}
			}
			if leftID == 0 {
				leftID = bothIDLeft
				if leftID == 0 {
					leftID = bothIDRight
				}
			}
		}
	}
	return rightID, leftID
}

// drawInputOverlay renders the text entry box when chatting.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	eui.Layout(outsideWidth, outsideHeight)

	if outsideWidth > 512 && outsideHeight > 384 {
		if gs.WindowWidth != outsideWidth || gs.WindowHeight != outsideHeight {
			gs.WindowWidth = outsideWidth
			gs.WindowHeight = outsideHeight
			settingsDirty = true
		}
	}

	return outsideWidth, outsideHeight
}

func runGame(ctx context.Context) {
	gameCtx = ctx

	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	w, h := ebiten.Monitor().Size()
	if w == 0 || h == 0 {
		w, h = initialWindowW, initialWindowH
	}
	if gameWin != nil {
		gameWin.SetSize(eui.Point{X: float32(w), Y: float32(h)})
	}
	if gs.Fullscreen {
		ebiten.SetFullscreen(true)
	}
	ebiten.SetWindowFloating(gs.Fullscreen || gs.AlwaysOnTop)

	op := &ebiten.RunGameOptions{ScreenTransparent: false}
	if err := ebiten.RunGameWithOptions(&Game{}, op); err != nil {
		log.Printf("ebiten: %v", err)
	}
	saveSettings()
}

func initGame() {
	ebiten.SetWindowTitle("goThoom Client")
	ebiten.SetVsyncEnabled(gs.vsync)
	ebiten.SetTPS(ebiten.SyncWithFPS)
	ebiten.SetCursorShape(ebiten.CursorShapeDefault)

	resetInventory()

	loadSettings()
	theme := gs.Theme
	if theme == "" {
		darkMode, err := dark.IsDarkMode()
		if err == nil {
			if darkMode {
				theme = "AccentDark"
			} else {
				theme = "AccentLight"
			}
		} else {
			theme = "AccentDark"
		}
	}
	eui.LoadTheme(theme)
	eui.LoadStyle("RoundHybrid")
	initUI()
	updateDimmedScreenBG()
	updateCharacterButtons()

	close(gameStarted)
}

func makeGameWindow() {
	if gameWin != nil {
		return
	}
	gameWin = eui.NewWindow()
	gameWin.Title = "Clan Lord"
	gameWin.Closable = false
	gameWin.Resizable = true
	gameWin.NoBGColor = true
	gameWin.Movable = true
	gameWin.NoScroll = true
	gameWin.NoCache = true
	gameWin.NoScale = true
	gameWin.AlwaysDrawFirst = true
	if !settingsLoaded {
		gameWin.SetZone(eui.HZoneCenter, eui.VZoneTop)
	}
	gameWin.Size = eui.Point{X: 8000, Y: 8000}
	gameWin.MarkOpen()
	gameWin.OnResize = func() { onGameWindowResize() }
	// Titlebar maximize button controlled by settings (now default on)
	gameWin.Maximizable = true
	// Keep same horizontal center on maximize
	gameWin.OnMaximize = func() {
		if gameWin == nil {
			return
		}
		// Record current center X before size change
		pos := gameWin.GetPos()
		sz := gameWin.GetSize()
		centerX := float64(pos.X) + float64(sz.X)/2
		// Maximize to screen bounds first
		w, h := eui.ScreenSize()
		gameWin.ClearZone()
		_ = gameWin.SetPos(eui.Point{X: 0, Y: 0})
		_ = gameWin.SetSize(eui.Point{X: float32(w), Y: float32(h)})
		// Aspect ratio handler will adjust size via OnResize; recalc size
		sz2 := gameWin.GetSize()
		newW := float64(sz2.X)
		// Recenter horizontally to keep same center
		newX := centerX - newW/2
		if newX < 0 {
			newX = 0
		}
		maxX := float64(w) - newW
		if newX > maxX {
			newX = maxX
		}
		_ = gameWin.SetPos(eui.Point{X: float32(newX), Y: 0})
		updateGameImageSize()
		layoutNotifications()
	}
	updateGameWindowSize()
	updateGameImageSize()
	layoutNotifications()
}

// onGameWindowResize enforces the game's aspect ratio on the window's
// content area (excluding titlebar and padding) and updates the image size.
func onGameWindowResize() {
	if gameWin == nil {
		return
	}
	if inAspectResize {
		updateGameImageSize()
		return
	}

	size := gameWin.GetSize()
	if size.X <= 0 || size.Y <= 0 {
		return
	}

	// Available inner content area (exclude titlebar and padding)
	pad := float64(2 * gameWin.Padding)
	title := float64(gameWin.GetTitleSize())
	availW := float64(int(size.X)&^1) - pad
	availH := float64(int(size.Y)&^1) - pad - title
	if availW <= 0 || availH <= 0 {
		updateGameImageSize()
		return
	}

	// Fit the content to the largest rectangle with the game's aspect ratio.
	targetW := float64(gameAreaSizeX)
	targetH := float64(gameAreaSizeY)
	scale := math.Min(availW/targetW, availH/targetH)
	if scale < 0.25 {
		scale = 0.25
	}
	fitW := targetW * scale
	fitH := targetH * scale
	newW := float32(math.Round(fitW + pad))
	newH := float32(math.Round(fitH + pad + title))

	if math.Abs(float64(size.X)-float64(newW)) > 0.5 || math.Abs(float64(size.Y)-float64(newH)) > 0.5 {
		inAspectResize = true
		_ = gameWin.SetSize(eui.Point{X: newW, Y: newH})
		inAspectResize = false
	}
	updateGameImageSize()
	layoutNotifications()
}

func noteFrame() {
	if playingMovie {
		return
	}
	now := time.Now()
	frameMu.Lock()
	if !lastFrameTime.IsZero() {
		dt := now.Sub(lastFrameTime)
		ms := int(dt.Round(10*time.Millisecond) / time.Millisecond)
		if ms > 0 {
			intervalHist[ms]++
			var modeMS, modeCount int
			for v, c := range intervalHist {
				if c > modeCount {
					modeMS, modeCount = v, c
				}
			}
			if modeMS > 0 {
				fps := (1000.0 / float64(modeMS))
				if fps < 1 {
					fps = 1
				}
				serverFPS = fps
				frameInterval = time.Second / time.Duration(fps)
			}
		}
	}
	lastFrameTime = now
	frameMu.Unlock()
	select {
	case frameCh <- struct{}{}:
	default:
	}
}

func sendInputLoop(ctx context.Context, conn net.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-frameCh:
		}
		frameMu.Lock()
		interval := frameInterval
		last := lastFrameTime
		frameMu.Unlock()
		if time.Since(last) > 2*time.Second || conn == nil {
			continue
		}
		delay := interval
		if delay <= 0 {
			delay = 200 * time.Millisecond
		}
		if gs.lateInputUpdates {
			latencyMu.Lock()
			lat := netLatency
			latencyMu.Unlock()
			// Send the input early enough for the server to receive it
			// before the next update, adding a safety margin to the
			// measured latency.
			adjusted := (lat * lateRatio) / 100
			delay = interval - adjusted
			if delay < 0 {
				delay = 0
			}
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		frameMu.Lock()
		last = lastFrameTime
		frameMu.Unlock()
		if time.Since(last) > 2*time.Second || conn == nil {
			continue
		}
		inputMu.Lock()
		s := latestInput
		inputMu.Unlock()
		if err := sendPlayerInput(conn, s.mouseX, s.mouseY, s.mouseDown); err != nil {
			logError("send player input: %v", err)
		}
	}
}

func udpReadLoop(ctx context.Context, conn net.Conn) {
	for {
		if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			logError("udp deadline: %v", err)
			return
		}
		m, err := readUDPMessage(conn)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			logError("udp read error: %v", err)
			handleDisconnect()
			return
		}
		tag := binary.BigEndian.Uint16(m[:2])
		flags := frameFlags(m)
		if recorder != nil {
			if !wroteLoginBlocks {
				if tag == 2 { // first draw state
					if len(loginGameState) > 0 {
						recorder.AddBlock(gameStateBlock(loginGameState), flagGameState)
					}
					if len(loginMobileData) > 0 {
						recorder.AddBlock(loginMobileData, flagMobileData)
					}
					if len(loginPictureTable) > 0 {
						recorder.AddBlock(loginPictureTable, flagPictureTable)
					}
					wroteLoginBlocks = true
					if err := recorder.WriteFrame(m, flags); err != nil {
						logError("record frame: %v", err)
					}
				} else {
					if flags&flagGameState != 0 {
						payload := append([]byte(nil), m[2:]...)
						parseGameState(payload, uint16(clientVersion), uint16(movieRevision))
						loginGameState = payload
					}
					if flags&flagMobileData != 0 {
						payload := append([]byte(nil), m[2:]...)
						parseMobileTable(payload, 0, uint16(clientVersion), uint16(movieRevision))
						loginMobileData = payload
					}
					if flags&flagPictureTable != 0 {
						payload := append([]byte(nil), m[2:]...)
						loginPictureTable = payload
					}
				}
			} else {
				if err := recorder.WriteFrame(m, flags); err != nil {
					logError("record frame: %v", err)
				}
			}
		}
		latencyMu.Lock()
		if !lastInputSent.IsZero() {
			rtt := time.Since(lastInputSent)
			if netLatency == 0 {
				netLatency = rtt
			} else {
				netLatency = (netLatency*7 + rtt) / 8
			}
			lastInputSent = time.Time{}
		}
		latencyMu.Unlock()
		processServerMessage(m)
	}
}

func tcpReadLoop(ctx context.Context, conn net.Conn) {
loop:
	for {
		if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			logError("set read deadline: %v", err)
			break
		}
		m, err := readTCPMessage(conn)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				select {
				case <-ctx.Done():
					break loop
				default:
					continue
				}
			}
			logError("read error: %v", err)
			handleDisconnect()
			break
		}
		tag := binary.BigEndian.Uint16(m[:2])
		flags := frameFlags(m)
		if recorder != nil {
			if !wroteLoginBlocks {
				if tag == 2 { // first draw state
					if len(loginGameState) > 0 {
						recorder.AddBlock(gameStateBlock(loginGameState), flagGameState)
					}
					if len(loginMobileData) > 0 {
						recorder.AddBlock(loginMobileData, flagMobileData)
					}
					if len(loginPictureTable) > 0 {
						recorder.AddBlock(loginPictureTable, flagPictureTable)
					}
					wroteLoginBlocks = true
					if err := recorder.WriteFrame(m, flags); err != nil {
						logError("record frame: %v", err)
					}
				} else {
					if flags&flagGameState != 0 {
						payload := append([]byte(nil), m[2:]...)
						parseGameState(payload, uint16(clientVersion), uint16(movieRevision))
						loginGameState = payload
					}
					if flags&flagMobileData != 0 {
						payload := append([]byte(nil), m[2:]...)
						parseMobileTable(payload, 0, uint16(clientVersion), uint16(movieRevision))
						loginMobileData = payload
					}
					if flags&flagPictureTable != 0 {
						payload := append([]byte(nil), m[2:]...)
						loginPictureTable = payload
					}
				}
			} else {
				if err := recorder.WriteFrame(m, flags); err != nil {
					logError("record frame: %v", err)
				}
			}
		}
		processServerMessage(m)
		// Allow maintenance queues to issue commands even when the
		// player isn't moving; this keeps /be-info and /be-who flowing
		// during idle periods on live connections.
		if pendingCommand == "" {
			if !maybeEnqueueInfo() {
				_ = maybeEnqueueWho()
			}
		}
		select {
		case <-ctx.Done():
			break loop
		default:
		}
	}
}

func frameFlags(m []byte) uint16 {
	flags := uint16(0)
	if gPlayersListIsStale {
		flags |= flagStale
	}
	switch {
	case looksLikeGameState(m):
		flags |= flagGameState
	case looksLikeMobileData(m):
		flags |= flagMobileData
	case looksLikePictureTable(m):
		flags |= flagPictureTable
	}
	return flags
}

func looksLikeGameState(m []byte) bool {
	if i := bytes.IndexByte(m, 0); i >= 0 {
		rest := m[i+1:]
		return looksLikePictureTable(rest) || looksLikeMobileData(rest)
	}
	return false
}

func looksLikeMobileData(m []byte) bool {
	return bytes.Contains(m, []byte{0xff, 0xff, 0xff, 0xff})
}

func looksLikePictureTable(m []byte) bool {
	if len(m) < 2 {
		return false
	}
	count := int(binary.BigEndian.Uint16(m[:2]))
	size := 2 + 6*count + 4
	return count > 0 && size == len(m)
}

// roundToInt returns the nearest integer to f. It avoids calling math.Round
// and handles negative values correctly.
func roundToInt(f float64) int {
	if f >= 0 {
		return int(f + 0.5)
	}
	return int(f - 0.5)
}
