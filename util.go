package main

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/twofish"
	"golang.org/x/text/encoding/charmap"
)

func simpleEncrypt(data []byte) {
	key := []byte{0x3c, 0x5a, 0x69, 0x93, 0xa5, 0xc6}
	j := 0
	for i := range data {
		data[i] ^= key[j]
		j++
		if j == len(key) {
			j = 0
		}
	}
}

func encodeMacRoman(s string) []byte {
	b, err := charmap.Macintosh.NewEncoder().Bytes([]byte(s))
	if err != nil {
		return []byte(s)
	}
	return b
}

// utfFold preserves the original string without stripping accents.
// Accented characters should be kept everywhere except for filesystem paths.
func utfFold(s string) string { return s }

func encodeFullVersion(v int) uint32 { return uint32(v) << 8 }

const (
	baseVersion  = 1353
	kDescPlayer  = 1
	kDescMonster = 2
	kDescNPC     = 3
)

const (
	kColorCodeBackWhite = iota
	kColorCodeBackGreen
	kColorCodeBackYellow
	kColorCodeBackRed
	kColorCodeBackBlack
	kColorCodeBackBlue
	kColorCodeBackGrey
	kColorCodeBackCyan
	kColorCodeBackOrange
)

func hexDump(prefix string, data []byte) {
	if !doDebug {
		return
	}
	log.Printf("%v %d bytes\n%v", prefix, len(data), hex.Dump(data))
}

const (
	kTypeVersion = 0x56657273 // 'Vers'
)

var errorNames = map[int16]string{
	-30972: "kDownloadNewVersionLive",
	-30973: "kDownloadNewVersionTest",
	-30999: "kBadCharName",
	-30998: "kBadCharPass",
	-30996: "kIncompatibleVersions",
	-30992: "kShuttingDown",
	-30991: "kGameNotOpen",
	-30988: "kBadAcctName",
	-30987: "kBadAcctPass",
	-30985: "kNoFreeSlot",
	-30984: "kBadAcctChar",
	-30981: "kCharOnline",
}

// errorFriendly maps known kError codes to concise, plain-English descriptions
// that are shown to the user. The technical kError name is still surfaced
// alongside the code for those who want details.
var errorFriendly = map[int16]string{
	-30999: "Unknown character name",
	-30998: "Incorrect character password",
	-30996: "Incompatible client version",
	-30992: "Server is shutting down",
	-30991: "Game is not open",
	-30988: "Unknown account name",
	-30987: "Incorrect account password",
	-30985: "Server is full (no free slot)",
	-30984: "Character does not belong to this account",
	-30981: "Character is already logged in",
	-30973: "A newer client/data version is required (test)",
	-30972: "A newer client/data version is required",
}

// describeKError returns a friendly description and the kError name for a
// known error code. ok is false when the code is unknown.
func describeKError(code int16) (desc, name string, ok bool) {
	name, ok = errorNames[code]
	if !ok {
		return "", "", false
	}
	if d, ok2 := errorFriendly[code]; ok2 {
		return d, name, true
	}
	// Fallback to the technical name split into words
	return name, name, true
}

var doDebug bool
var silent bool
var ackFrame int32
var resendFrame int32
var lastAckFrame int32
var numFrames int
var lostFrames int
var frameBuckets [5]int
var lostBuckets [5]int
var bucketTimes [5]int64
var commandNum uint32 = 1
var pendingCommand string
var commandQueue []string
var playerName string
var playerIndex uint8 = 0xff

func enqueueCommand(cmd string) {
	if cmd != "" {
		commandQueue = append(commandQueue, cmd)
	}
}

func nextCommand() {
	if pendingCommand == "" && len(commandQueue) > 0 {
		pendingCommand = commandQueue[0]
		commandQueue = commandQueue[1:]
	}
}

// updateFrameCounters tracks frame statistics and detects dropped frames.
// It returns the number of frames missing between the previous and
// current acknowledgement numbers.
func updateFrameCounters(newFrame int32) int {
	now := time.Now().Unix()
	idx := int(now % 5)
	if bucketTimes[idx] != now {
		frameBuckets[idx] = 0
		lostBuckets[idx] = 0
		bucketTimes[idx] = now
	}
	frameBuckets[idx]++
	numFrames++
	dropped := 0
	if lastAckFrame != 0 {
		lost := int(newFrame - lastAckFrame - 1)
		if lost > 0 {
			lostFrames += lost
			dropped = lost
			frameBuckets[idx] += lost
			lostBuckets[idx] += lost
		}
	}
	lastAckFrame = newFrame
	return dropped
}

func droppedPercent() float64 {
	now := time.Now().Unix()
	total := 0
	lost := 0
	for i := 0; i < 5; i++ {
		if now-bucketTimes[i] < 5 {
			total += frameBuckets[i]
			lost += lostBuckets[i]
		}
	}
	if total == 0 {
		return 0
	}
	return float64(lost) * 100 / float64(total)
}

const (
	kBubbleNormal = iota
	kBubbleWhisper
	kBubbleYell
	kBubbleThought
	kBubbleRealAction
	kBubbleMonster
	kBubblePlayerAction
	kBubblePonder
	kBubbleNarrate
)

const (
	kBubbleTypeMask  = 0x3F
	kBubbleNotCommon = 0x40
	kBubbleFar       = 0x80
)

func isChatBubble(t int) bool {
	switch t {
	case kBubbleNormal, kBubbleWhisper, kBubbleYell, kBubbleThought, kBubblePonder, kBubbleRealAction, kBubblePlayerAction:
		return true
	default:
		return false
	}
}

// bubble languages and codes from Public_cl.h
const (
	kBubbleHalfling = iota
	kBubbleSylvan
	kBubblePeople
	kBubbleThoom
	kBubbleDwarf
	kBubbleGhorakZo
	kBubbleAncient
	kBubbleMagic
	kBubbleCommon
	kBubbleThieves
	kBubbleMystic
	kBubbleLangMonster
	kBubbleLangUnknown
	kBubbleOrga
	kBubbleSirrush
	kBubbleAzcatl
	kBubbleLepori
	kBubbleNumLanguages
)

const (
	kBubbleLanguageMask  = 0x3F
	kBubbleCodeMask      = 0xC0
	kBubbleCodeKnown     = 0x00
	kBubbleUnknownShort  = 0x40
	kBubbleUnknownMedium = 0x80
	kBubbleUnknownLong   = 0xC0
)

const kPIMDownField = 0x0001 // mouse down; player wants to move

// illumination flags from Public_cl.h
const (
	kLightAdjust25Pct  = 1 << 0
	kLightAdjust50Pct  = 1 << 1
	kLightAreaIsDarker = 1 << 2
	kLightNoNightMods  = 1 << 3
	kLightNoShadows    = 1 << 4
	kLightForce100Pct  = 1 << 5
)

// inventory command values from Public_cl.h
const (
	kInvCmdNone = iota
	kInvCmdFull
	kInvCmdAdd
	kInvCmdAddEquip
	kInvCmdDelete
	kInvCmdEquip
	kInvCmdUnequip
	kInvCmdMultiple
	kInvCmdName

	kInvCmdIndex         = 0x80
	kInvCmdLegacyPadding = 0xB1 // trailing byte in legacy inventory streams
)

// item slots from Public_cl.h
const (
	kItemSlotNotInventory = iota
	kItemSlotNotWearable
	kItemSlotForehead
	kItemSlotNeck
	kItemSlotShoulder
	kItemSlotArms
	kItemSlotGloves
	kItemSlotFinger
	kItemSlotCoat
	kItemSlotCloak
	kItemSlotTorso
	kItemSlotWaist
	kItemSlotLegs
	kItemSlotFeet
	kItemSlotRightHand
	kItemSlotLeftHand
	kItemSlotBothHands
	kItemSlotHead

	kItemSlotFirstReal = kItemSlotForehead
	kItemSlotLastReal  = kItemSlotHead
)

func readKeyFileVersion(path string) (uint32, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var header [12]byte
	if _, err := io.ReadFull(f, header[:]); err != nil {
		return 0, err
	}
	count := int(binary.BigEndian.Uint32(header[2:6]))

	entry := make([]byte, 16)
	for i := 0; i < count; i++ {
		if _, err := io.ReadFull(f, entry); err != nil {
			return 0, err
		}
		pos := binary.BigEndian.Uint32(entry[0:4])
		size := binary.BigEndian.Uint32(entry[4:8])
		typ := binary.BigEndian.Uint32(entry[8:12])
		id := binary.BigEndian.Uint32(entry[12:16])
		if typ == kTypeVersion && id == 0 {
			if _, err := f.Seek(int64(pos), io.SeekStart); err != nil {
				return 0, err
			}
			buf := make([]byte, size)
			if _, err := io.ReadFull(f, buf); err != nil {
				return 0, err
			}
			v := binary.BigEndian.Uint32(buf)
			if v <= 0xFF {
				v <<= 8
			}
			return v, nil
		}
	}
	return 0, fmt.Errorf("version record not found in %v", path)
}

func answerChallenge(password string, challenge []byte) ([]byte, error) {
	digest := md5.Sum([]byte(password))
	key := make([]byte, len(digest))
	copy(key, digest[:])
	swapped := make([]byte, len(key))
	for i := 0; i < len(key); i += 4 {
		v := binary.BigEndian.Uint32(key[i : i+4])
		binary.LittleEndian.PutUint32(swapped[i:i+4], v)
	}
	block, err := twofish.NewCipher(swapped)
	if err != nil {
		return nil, err
	}
	if len(challenge)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("invalid challenge length")
	}
	plain := make([]byte, len(challenge))
	for i := 0; i < len(challenge); i += block.BlockSize() {
		block.Decrypt(plain[i:i+block.BlockSize()], challenge[i:i+block.BlockSize()])
	}
	h := md5.Sum(plain)
	encoded := make([]byte, len(h))
	for i := 0; i < len(h); i += block.BlockSize() {
		block.Encrypt(encoded[i:i+block.BlockSize()], h[i:i+block.BlockSize()])
	}
	return encoded, nil
}

// answerChallengeHash is like answerChallenge but takes a precomputed
// MD5 hash of the password encoded as a hex string.
func answerChallengeHash(passHash string, challenge []byte) ([]byte, error) {
	digest, err := hex.DecodeString(passHash)
	if err != nil {
		return nil, err
	}
	if len(digest) != md5.Size {
		return nil, fmt.Errorf("invalid password hash length")
	}
	key := make([]byte, len(digest))
	copy(key, digest)
	swapped := make([]byte, len(key))
	for i := 0; i < len(key); i += 4 {
		v := binary.BigEndian.Uint32(key[i : i+4])
		binary.LittleEndian.PutUint32(swapped[i:i+4], v)
	}
	block, err := twofish.NewCipher(swapped)
	if err != nil {
		return nil, err
	}
	if len(challenge)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("invalid challenge length")
	}
	plain := make([]byte, len(challenge))
	for i := 0; i < len(challenge); i += block.BlockSize() {
		block.Decrypt(plain[i:i+block.BlockSize()], challenge[i:i+block.BlockSize()])
	}
	h := md5.Sum(plain)
	encoded := make([]byte, len(h))
	for i := 0; i < len(h); i += block.BlockSize() {
		block.Encrypt(encoded[i:i+block.BlockSize()], h[i:i+block.BlockSize()])
	}
	return encoded, nil
}
