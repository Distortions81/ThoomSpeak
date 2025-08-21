package main

import (
	"compress/gzip"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"

	"gothoom/climg"
	"gothoom/clsnd"
)

const defaultUpdateBase = "https://m45sci.xyz/downloads/clanlord"
const soundFontURL = "https://m45sci.xyz/u/dist/goThoom/soundfont.sf2"
const soundFontFile = "soundfont.sf2"

var updateBase = defaultUpdateBase

// downloadStatus, when set by the UI, receives human-readable status updates
// during downloads (e.g., connecting, bytes downloaded, completion).
var downloadStatus func(string)

// downloadProgress, when set by the UI, receives byte progress updates.
// total will be <= 0 if unknown.
var downloadProgress func(name string, read, total int64)

// downloadCtx and downloadCancel allow in-flight downloads to be aborted.
var downloadCtx = context.Background()
var downloadCancel context.CancelFunc = func() {}

var downloadGZ = func(url, dest string) error {
	consoleMessage(fmt.Sprintf("Downloading: %v...", url))
	if downloadStatus != nil {
		downloadStatus(fmt.Sprintf("Connecting to %s...", url))
	}

	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, url, nil)
	if err != nil {
		if downloadStatus != nil {
			downloadStatus(fmt.Sprintf("Error creating request: %v", err))
		}
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logError("GET %v: %v", url, err)
		if downloadStatus != nil {
			downloadStatus(fmt.Sprintf("Error connecting: %v", err))
		}
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("GET %v: %v", url, resp.Status)
		logError("download %v: %v", url, err)
		if downloadStatus != nil {
			downloadStatus(fmt.Sprintf("HTTP error: %v", resp.Status))
		}
		return err
	}
	// Inform UI that we are connected and initialize progress.
	if downloadStatus != nil {
		// Show a succinct state transition so "Connecting" doesn't linger.
		host := resp.Request.URL.Host
		humanTotal := "unknown"
		if resp.ContentLength > 0 {
			humanTotal = humanize.Bytes(uint64(resp.ContentLength))
		}
		downloadStatus(fmt.Sprintf("Connected to %s — starting download (%s)", host, humanTotal))
	}

	// Set up compressed byte counter for progress percentage and speed/ETA.
	pc := &progCounter{name: filepath.Base(dest), size: resp.ContentLength}
	// Kick the UI once so it can switch the bar from idle to active.
	if downloadProgress != nil {
		downloadProgress(pc.name, 0, pc.size)
	}
	body := io.TeeReader(resp.Body, pc)
	gz, err := gzip.NewReader(body)
	if err != nil {
		logError("gzip reader %v: %v", url, err)
		if downloadStatus != nil {
			downloadStatus(fmt.Sprintf("Error: %v", err))
		}
		return err
	}
	defer gz.Close()
	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		logError("create %v: %v", tmp, err)
		if downloadStatus != nil {
			downloadStatus(fmt.Sprintf("Error: %v", err))
		}
		return err
	}
	removeTmp := true
	defer func() {
		if removeTmp {
			os.Remove(tmp)
		}
	}()
	// Copy the payload to disk while the progCounter (on the compressed stream)
	// drives progress updates.
	if _, err := io.Copy(f, gz); err != nil {
		f.Close()
		logError("copy %v: %v", tmp, err)
		if downloadStatus != nil {
			downloadStatus(fmt.Sprintf("Error: %v", err))
		}
		return err
	}
	if err := f.Close(); err != nil {
		logError("close %v: %v", tmp, err)
		if downloadStatus != nil {
			downloadStatus(fmt.Sprintf("Error: %v", err))
		}
		return err
	}
	// Ensure a final 100% progress update when size is known.
	if downloadProgress != nil && pc.size > 0 {
		downloadProgress(pc.name, pc.size, pc.size)
	}
	consoleMessage("Download complete.")
	if downloadStatus != nil {
		downloadStatus(fmt.Sprintf("Download complete: %s", filepath.Base(dest)))
	}
	if err := os.Rename(tmp, dest); err != nil {
		logError("rename %v to %v: %v", tmp, dest, err)
		return err
	}
	removeTmp = false
	return nil
}

var downloadFile = func(url, dest string) error {
	consoleMessage(fmt.Sprintf("Downloading: %v...", url))
	if downloadStatus != nil {
		downloadStatus(fmt.Sprintf("Connecting to %s...", url))
	}

	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, url, nil)
	if err != nil {
		if downloadStatus != nil {
			downloadStatus(fmt.Sprintf("Error creating request: %v", err))
		}
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logError("GET %v: %v", url, err)
		if downloadStatus != nil {
			downloadStatus(fmt.Sprintf("Error connecting: %v", err))
		}
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("GET %v: %v", url, resp.Status)
		logError("download %v: %v", url, err)
		if downloadStatus != nil {
			downloadStatus(fmt.Sprintf("HTTP error: %v", resp.Status))
		}
		return err
	}
	if downloadStatus != nil {
		host := resp.Request.URL.Host
		humanTotal := "unknown"
		if resp.ContentLength > 0 {
			humanTotal = humanize.Bytes(uint64(resp.ContentLength))
		}
		downloadStatus(fmt.Sprintf("Connected to %s — starting download (%s)", host, humanTotal))
	}
	pc := &progCounter{name: filepath.Base(dest), size: resp.ContentLength}
	if downloadProgress != nil {
		downloadProgress(pc.name, 0, pc.size)
	}
	body := io.TeeReader(resp.Body, pc)
	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		logError("create %v: %v", tmp, err)
		if downloadStatus != nil {
			downloadStatus(fmt.Sprintf("Error: %v", err))
		}
		return err
	}
	removeTmp := true
	defer func() {
		if removeTmp {
			os.Remove(tmp)
		}
	}()
	if _, err := io.Copy(f, body); err != nil {
		f.Close()
		logError("copy %v: %v", tmp, err)
		if downloadStatus != nil {
			downloadStatus(fmt.Sprintf("Error: %v", err))
		}
		return err
	}
	if err := f.Close(); err != nil {
		logError("close %v: %v", tmp, err)
		if downloadStatus != nil {
			downloadStatus(fmt.Sprintf("Error: %v", err))
		}
		return err
	}
	if downloadProgress != nil && pc.size > 0 {
		downloadProgress(pc.name, pc.size, pc.size)
	}
	consoleMessage("Download complete.")
	if downloadStatus != nil {
		downloadStatus(fmt.Sprintf("Download complete: %s", filepath.Base(dest)))
	}
	if err := os.Rename(tmp, dest); err != nil {
		logError("rename %v to %v: %v", tmp, dest, err)
		return err
	}
	removeTmp = false
	return nil
}

func headSize(url string) int64 {
	resp, err := http.Head(url)
	if err != nil {
		return -1
	}
	resp.Body.Close()
	return resp.ContentLength
}

// progCounter tracks compressed bytes for progress percentage.
type progCounter struct {
	last  time.Time
	total int64
	size  int64
	name  string
}

func (pc *progCounter) Write(p []byte) (int, error) {
	n := len(p)
	pc.total += int64(n)
	if time.Since(pc.last) >= 200*time.Millisecond {
		if downloadProgress != nil {
			downloadProgress(pc.name, pc.total, pc.size)
		}
		pc.last = time.Now()
	}
	return n, nil
}

func autoUpdate(resp []byte, dataDir string) (int, error) {
	if len(resp) < 16 {
		return 0, fmt.Errorf("short response for update")
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logError("create %v: %v", dataDir, err)
		return 0, err
	}
	base := string(resp[16:])
	if i := strings.IndexByte(base, 0); i >= 0 {
		base = base[:i]
	}
	base = strings.TrimRight(base, "/")
	clientVer := binary.BigEndian.Uint32(resp[4:8])
	logDebug("Client version: %v", clientVer)
	imgVer := int(binary.BigEndian.Uint32(resp[8:12]) >> 8)
	sndVer := int(binary.BigEndian.Uint32(resp[12:16]) >> 8)
	imgPath := filepath.Join(dataDir, CL_ImagesFile)
	if old, err := readKeyFileVersion(imgPath); err == nil {
		if experimental {
			patchURL := fmt.Sprintf("%v/data/CL_Images.%dto%d.gz", base, int(old>>8), imgVer)
			if err := downloadPatch(patchURL, imgPath, climg.ApplyPatch); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					imgURL := fmt.Sprintf("%v/data/CL_Images.%d.gz", base, imgVer)
					if err := downloadGZ(imgURL, imgPath); err != nil {
						logError("download %v: %v", imgURL, err)
						return 0, err
					}
				} else {
					logError("patch %v: %v", patchURL, err)
					return 0, err
				}
			}
		} else {
			imgURL := fmt.Sprintf("%v/data/CL_Images.%d.gz", base, imgVer)
			if err := downloadGZ(imgURL, imgPath); err != nil {
				logError("download %v: %v", imgURL, err)
				return 0, err
			}
		}
	} else {
		imgURL := fmt.Sprintf("%v/data/CL_Images.%d.gz", base, imgVer)
		if err := downloadGZ(imgURL, imgPath); err != nil {
			logError("download %v: %v", imgURL, err)
			return 0, err
		}
	}
	sndPath := filepath.Join(dataDir, CL_SoundsFile)
	if old, err := readKeyFileVersion(sndPath); err == nil {
		if experimental {
			patchURL := fmt.Sprintf("%v/data/CL_Sounds.%dto%d.gz", base, int(old>>8), sndVer)
			if err := downloadPatch(patchURL, sndPath, clsnd.ApplyPatch); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					sndURL := fmt.Sprintf("%v/data/CL_Sounds.%d.gz", base, sndVer)
					if err := downloadGZ(sndURL, sndPath); err != nil {
						logError("download %v: %v", sndURL, err)
						return 0, err
					}
				} else {
					logError("patch %v: %v", patchURL, err)
					return 0, err
				}
			}
		} else {
			sndURL := fmt.Sprintf("%v/data/CL_Sounds.%d.gz", base, sndVer)
			if err := downloadGZ(sndURL, sndPath); err != nil {
				logError("download %v: %v", sndURL, err)
				return 0, err
			}
		}
	} else {
		sndURL := fmt.Sprintf("%v/data/CL_Sounds.%d.gz", base, sndVer)
		if err := downloadGZ(sndURL, sndPath); err != nil {
			logError("download %v: %v", sndURL, err)
			return 0, err
		}
	}
	return int(clientVer >> 8), nil
}

type fileInfo struct {
	Name string
	Size int64
}

type dataFilesStatus struct {
	NeedImages    bool
	NeedSounds    bool
	NeedSoundfont bool
	Files         []fileInfo
	SoundfontSize int64
	Version       int
	ImageVersion  int
	SoundVersion  int
}

func checkDataFiles(clientVer int) (dataFilesStatus, error) {
	var status dataFilesStatus

	imgPath := filepath.Join(dataDirPath, CL_ImagesFile)
	if v, err := readKeyFileVersion(imgPath); err != nil {
		if !os.IsNotExist(err) {
			logError("read %v: %v", imgPath, err)
		}
		status.NeedImages = true
	} else {
		ver := int(v >> 8)
		status.Version = ver
		status.ImageVersion = ver
		if ver < clientVer {
			status.NeedImages = true
		}
	}

	sndPath := filepath.Join(dataDirPath, CL_SoundsFile)
	if v, err := readKeyFileVersion(sndPath); err != nil {
		if !os.IsNotExist(err) {
			logError("read %v: %v", sndPath, err)
		}
		status.NeedSounds = true
	} else {
		ver := int(v >> 8)
		status.SoundVersion = ver
		if status.Version == 0 || ver > status.Version {
			status.Version = ver
		}
		if ver < clientVer {
			status.NeedSounds = true
		}
	}

	if status.NeedImages {
		name := fmt.Sprintf("CL_Images.%d.gz", clientVer)
		size := headSize(fmt.Sprintf("%v/data/%s", updateBase, name))
		status.Files = append(status.Files, fileInfo{Name: name, Size: size})
	}
	if status.NeedSounds {
		name := fmt.Sprintf("CL_Sounds.%d.gz", clientVer)
		size := headSize(fmt.Sprintf("%v/data/%s", updateBase, name))
		status.Files = append(status.Files, fileInfo{Name: name, Size: size})
	}

	sfPath := filepath.Join(dataDirPath, soundFontFile)
	if _, err := os.Stat(sfPath); errors.Is(err, os.ErrNotExist) {
		status.NeedSoundfont = true
		status.SoundfontSize = headSize(soundFontURL)
	}

	return status, nil
}

func downloadDataFiles(clientVer int, status dataFilesStatus, getSoundfont bool) error {
	if err := os.MkdirAll(dataDirPath, 0755); err != nil {
		logError("create %v: %v", dataDirPath, err)
		return err
	}
	if status.NeedImages {
		imgPath := filepath.Join(dataDirPath, CL_ImagesFile)
		if status.ImageVersion > 0 && experimental {
			patchURL := fmt.Sprintf("%v/data/CL_Images.%dto%d.gz", updateBase, status.ImageVersion, clientVer)
			if err := downloadPatch(patchURL, imgPath, climg.ApplyPatch); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					imgURL := fmt.Sprintf("%v/data/CL_Images.%d.gz", updateBase, clientVer)
					if err := downloadGZ(imgURL, imgPath); err != nil {
						logError("download %v: %v", imgURL, err)
						return fmt.Errorf("download CL_Images: %w", err)
					}
				} else {
					logError("patch %v: %v", patchURL, err)
					return fmt.Errorf("patch CL_Images: %w", err)
				}
			}
		} else {
			imgURL := fmt.Sprintf("%v/data/CL_Images.%d.gz", updateBase, clientVer)
			if err := downloadGZ(imgURL, imgPath); err != nil {
				logError("download %v: %v", imgURL, err)
				return fmt.Errorf("download CL_Images: %w", err)
			}
		}
	}
	if status.NeedSounds {
		sndPath := filepath.Join(dataDirPath, CL_SoundsFile)
		if status.SoundVersion > 0 && experimental {
			patchURL := fmt.Sprintf("%v/data/CL_Sounds.%dto%d.gz", updateBase, status.SoundVersion, clientVer)
			if err := downloadPatch(patchURL, sndPath, clsnd.ApplyPatch); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					sndURL := fmt.Sprintf("%v/data/CL_Sounds.%d.gz", updateBase, clientVer)
					if err := downloadGZ(sndURL, sndPath); err != nil {
						logError("download %v: %v", sndURL, err)
						return fmt.Errorf("download CL_Sounds: %w", err)
					}
				} else {
					logError("patch %v: %v", patchURL, err)
					return fmt.Errorf("patch CL_Sounds: %w", err)
				}
			}
		} else {
			sndURL := fmt.Sprintf("%v/data/CL_Sounds.%d.gz", updateBase, clientVer)
			if err := downloadGZ(sndURL, sndPath); err != nil {
				logError("download %v: %v", sndURL, err)
				return fmt.Errorf("download CL_Sounds: %w", err)
			}
		}
	}
	if getSoundfont {
		sfPath := filepath.Join(dataDirPath, soundFontFile)
		if err := downloadFile(soundFontURL, sfPath); err != nil {
			logError("download %v: %v", soundFontURL, err)
			return fmt.Errorf("download soundfont: %w", err)
		}
	}
	return nil
}

func downloadPatch(url, dest string, apply func(string, []byte) error) error {
	consoleMessage(fmt.Sprintf("Downloading: %v...", url))
	if downloadStatus != nil {
		downloadStatus(fmt.Sprintf("Connecting to %s...", url))
	}
	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return os.ErrNotExist
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %v: %v", url, resp.Status)
	}
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	defer gz.Close()
	data, err := io.ReadAll(gz)
	if err != nil {
		return err
	}
	if err := apply(dest, data); err != nil {
		return err
	}
	if downloadStatus != nil {
		downloadStatus(fmt.Sprintf("Patch applied: %s", filepath.Base(dest)))
	}
	return nil
}
