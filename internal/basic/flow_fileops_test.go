package basic

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func withFastCopyRetries(t *testing.T) {
	t.Helper()
	prev := copyRetryDelays
	copyRetryDelays = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	t.Cleanup(func() { copyRetryDelays = prev })
}

func TestCopyFileCopiesContent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "a.txt")
	dst := filepath.Join(dir, "nested", "b.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != "hello" {
		t.Fatalf("dst = %q, %v", got, err)
	}
}

func TestCopyFileRetriesWhileLocked(t *testing.T) {
	withFastCopyRetries(t)
	calls := 0
	prev := copyFileOnceFn
	copyFileOnceFn = func(src, dst string) error {
		calls++
		if calls < 3 {
			return errors.New("open x: The process cannot access the file because it is being used by another process.")
		}
		return nil
	}
	t.Cleanup(func() { copyFileOnceFn = prev })

	if err := copyFile("src", "dst"); err != nil {
		t.Fatalf("expected success after lock cleared, got %v", err)
	}
	if calls != 3 {
		t.Fatalf("calls = %d, want 3", calls)
	}
}

func TestCopyFileGivesUpAfterRetries(t *testing.T) {
	withFastCopyRetries(t)
	calls := 0
	prev := copyFileOnceFn
	copyFileOnceFn = func(src, dst string) error {
		calls++
		return errors.New("sharing violation")
	}
	t.Cleanup(func() { copyFileOnceFn = prev })

	if err := copyFile("src", "dst"); err == nil {
		t.Fatal("expected error when file stays locked")
	}
	if want := len(copyRetryDelays) + 1; calls != want {
		t.Fatalf("calls = %d, want %d", calls, want)
	}
}

func TestCopyFileDoesNotRetryOtherErrors(t *testing.T) {
	withFastCopyRetries(t)
	calls := 0
	prev := copyFileOnceFn
	copyFileOnceFn = func(src, dst string) error {
		calls++
		return os.ErrNotExist
	}
	t.Cleanup(func() { copyFileOnceFn = prev })

	if err := copyFile("src", "dst"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err = %v", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestAutoSavedSessionNameIsUnique(t *testing.T) {
	now := time.Date(2026, 11, 2, 18, 5, 0, 0, time.UTC)
	first := autoSavedSessionName(map[string]string{}, now)
	if first != "Unsaved login 2026-11-02 18.05" {
		t.Fatalf("first = %q", first)
	}
	second := autoSavedSessionName(map[string]string{"a": first}, now)
	if second != "Unsaved login 2026-11-02 18.05 (2)" {
		t.Fatalf("second = %q", second)
	}
}
