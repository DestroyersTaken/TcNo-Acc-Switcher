package basic

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"TcNo-Acc-Switcher/internal/actionlog"
)

func isSharingViolationErr(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "being used by another process") ||
		strings.Contains(s, "the process cannot access the file") ||
		strings.Contains(s, "resource busy") ||
		strings.Contains(s, "sharing violation") ||
		strings.Contains(s, "lock violation")
}

// copyRetryDelays is how long copyFile waits between attempts when a file is
// locked by another process. A launcher that was just closed can keep handles
// on its login files (LevelDB, cookies) for a second or two after the process
// itself is gone; this rides that out instead of failing the whole switch.
// Total wait is about 4.5 s. Package variable so tests can shorten it.
var copyRetryDelays = []time.Duration{
	150 * time.Millisecond,
	300 * time.Millisecond,
	600 * time.Millisecond,
	1200 * time.Millisecond,
	2200 * time.Millisecond,
}

// copyFile copies src to dst, retrying while either side is locked by another
// process. Non-lock errors fail straight away.
func copyFile(src, dst string) error {
	err := copyFileOnceFn(src, dst)
	for attempt := 0; err != nil && isSharingViolationErr(err) && attempt < len(copyRetryDelays); attempt++ {
		wait := copyRetryDelays[attempt]
		logFlow().Info("copyFile: file locked, retrying", "src", src, "dst", dst, "attempt", attempt+1, "wait", wait)
		time.Sleep(wait)
		err = copyFileOnceFn(src, dst)
	}
	return err
}

// copyFileOnceFn is the single-attempt copy; a variable only so tests can
// simulate a locked file, which Linux cannot produce on demand.
var copyFileOnceFn = copyFileOnce

func copyFileOnce(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		if isSharingViolationErr(err) {
			logFlow().Warn("copyFile: source locked (sharing violation)", "src", src, "dst", dst, "err", err)
		} else {
			logFlow().Debug("copyFile: open source failed", "src", src, "dst", dst, "err", err)
		}
		actionlog.Record("file:copy", src, dst, err)
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(dst), err)
	}
	out, err := os.Create(dst)
	if err != nil {
		if isSharingViolationErr(err) {
			logFlow().Warn("copyFile: destination locked (sharing violation)", "src", src, "dst", dst, "err", err)
		} else {
			logFlow().Debug("copyFile: create destination failed", "src", src, "dst", dst, "err", err)
		}
		return fmt.Errorf("create %s: %w", dst, err)
	}
	defer out.Close()
	if _, err = io.Copy(out, in); err != nil {
		if isSharingViolationErr(err) {
			logFlow().Warn("copyFile: io.Copy hit sharing violation", "src", src, "dst", dst, "err", err)
		} else {
			logFlow().Debug("copyFile: io.Copy failed", "src", src, "dst", dst, "err", err)
		}
		actionlog.Record("file:copy", src, dst, err)
		return fmt.Errorf("copy %s -> %s: %w", src, dst, err)
	}
	// Flush before the platform is relaunched: on network drives, removable
	// media or behind some antivirus filters, Close alone can return before the
	// data is on disk and the launcher then reads a short file.
	if err := out.Sync(); err != nil {
		logFlow().Debug("copyFile: sync failed", "dst", dst, "err", err)
	}
	logFlow().Debug("copied file", "src", src, "dst", dst)
	actionlog.Record("file:copy", src, dst, nil)
	return nil
}

func copyFileToDir(src, dir string) error {
	return copyFile(src, filepath.Join(dir, filepath.Base(src)))
}

// copyOpErr wraps an error from copyFile/copyDir with a platform-key prefix
// and the live source path so user-facing toasts identify the failing operation.
func copyOpErr(platformKey, op, src string, err error) error {
	if err == nil {
		return nil
	}
	key := strings.TrimSpace(platformKey)
	if key == "" {
		key = "platform"
	}
	return fmt.Errorf("%s: %s %s: %w", key, op, src, err)
}

func copyDir(src, dst string) error {
	logFlow().Debug("copy directory tree", "src", src, "dst", dst)
	return filepath.WalkDir(src, func(path string, de os.DirEntry, err error) error {
		if err != nil {
			if isSharingViolationErr(err) {
				logFlow().Warn("copyDir: walk hit sharing violation", "src", src, "dst", dst, "path", path, "err", err)
			} else {
				logFlow().Debug("copyDir: walk failed", "src", src, "dst", dst, "path", path, "err", err)
			}
			return fmt.Errorf("walk %s: %w", path, err)
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return fmt.Errorf("rel %s: %w", path, err)
		}
		t := filepath.Join(dst, rel)
		if de.IsDir() {
			if err := os.MkdirAll(t, 0o755); err != nil {
				return fmt.Errorf("mkdir %s: %w", t, err)
			}
			return nil
		}
		return copyFile(path, t)
	})
}
