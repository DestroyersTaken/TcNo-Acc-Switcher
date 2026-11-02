package security

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTree(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "login.dat"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readLogin(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "login.dat"))
	if err != nil {
		t.Fatalf("read %s: %v", dir, err)
	}
	return string(b)
}

func TestPlainSaveCommitReplacesOldSave(t *testing.T) {
	normal := filepath.Join(t.TempDir(), "Accounts", "Alice")
	writeTree(t, normal, "old")

	save, err := beginPlainAccountSave("Epic", "uid", "Alice", normal)
	if err != nil {
		t.Fatal(err)
	}
	if save.DestRoot == normal {
		t.Fatal("save must be staged outside the live cache folder")
	}
	writeTree(t, save.DestRoot, "new")
	if got := readLogin(t, normal); got != "old" {
		t.Fatalf("old save changed before commit: %q", got)
	}

	if err := commitPlainAccountSave(save, normal); err != nil {
		t.Fatal(err)
	}
	CleanupAccountSave(save)

	if got := readLogin(t, normal); got != "new" {
		t.Fatalf("after commit = %q, want new", got)
	}
	for _, leftover := range []string{normal + plainSaveStagingSuffix, normal + plainSaveOldSuffix} {
		if _, err := os.Stat(leftover); !os.IsNotExist(err) {
			t.Fatalf("leftover %s: %v", leftover, err)
		}
	}
}

func TestPlainSaveAbortKeepsOldSave(t *testing.T) {
	normal := filepath.Join(t.TempDir(), "Accounts", "Bob")
	writeTree(t, normal, "good")

	save, err := beginPlainAccountSave("Riot", "uid", "Bob", normal)
	if err != nil {
		t.Fatal(err)
	}
	writeTree(t, save.DestRoot, "half-copied")
	// Copy failed: caller returns without committing, deferred cleanup runs.
	CleanupAccountSave(save)

	if got := readLogin(t, normal); got != "good" {
		t.Fatalf("old save lost after failed copy: %q", got)
	}
	if _, err := os.Stat(save.DestRoot); !os.IsNotExist(err) {
		t.Fatalf("staging not cleaned up: %v", err)
	}
}

func TestPlainSaveFirstSaveHasNoOldFolder(t *testing.T) {
	normal := filepath.Join(t.TempDir(), "Accounts", "New")
	save, err := beginPlainAccountSave("EA", "uid", "New", normal)
	if err != nil {
		t.Fatal(err)
	}
	writeTree(t, save.DestRoot, "first")
	if err := commitPlainAccountSave(save, normal); err != nil {
		t.Fatal(err)
	}
	if got := readLogin(t, normal); got != "first" {
		t.Fatalf("got %q", got)
	}
}

func TestRecoverParkedPlainSave(t *testing.T) {
	normal := filepath.Join(t.TempDir(), "Accounts", "Crash")
	writeTree(t, normal+plainSaveOldSuffix, "parked")
	recoverParkedPlainSave(normal)
	if got := readLogin(t, normal); got != "parked" {
		t.Fatalf("got %q", got)
	}
}
