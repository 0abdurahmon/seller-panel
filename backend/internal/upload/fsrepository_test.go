package upload

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFSRepositorySave(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewFSRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	ctx := context.Background()
	name := strings.Repeat("a", 32) + ".webp"

	if err := repo.Save(ctx, 5, name, []byte("data")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "5", name))
	if err != nil || string(got) != "data" {
		t.Fatalf("read %q, %v", got, err)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "5"))
	if len(entries) != 1 {
		t.Fatalf("shop dir has %d entries, want 1 (no temp files left)", len(entries))
	}
	if runtime.GOOS != "windows" { // Windows has no Unix permission bits; the volume is served from Linux.
		info, _ := os.Stat(filepath.Join(dir, "5", name))
		if info.Mode().Perm() != 0o644 {
			t.Fatalf("mode = %v, want 0644, nginx must be able to read it", info.Mode().Perm())
		}
		dirInfo, _ := os.Stat(filepath.Join(dir, "5"))
		if dirInfo.Mode().Perm() != 0o755 {
			t.Fatalf("dir mode = %v, want 0755", dirInfo.Mode().Perm())
		}
	}
}

func TestFSRepositoryDoesNotWriteOutsideTheUploadDir(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewFSRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	ctx := context.Background()
	name := strings.Repeat("b", 32) + ".webp"

	if err := repo.Save(ctx, 1, "../../x.webp", []byte("x")); err == nil {
		t.Fatal("a name with .. must not be saved")
	}

	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(dir, "9")); err != nil {
		t.Skipf("cannot create a symlink here (Windows needs a privilege): %v", err)
	}
	if err := repo.Save(ctx, 9, name, []byte("x")); err == nil {
		t.Fatal("a shop directory that is a symlink out of the upload dir must not be written")
	}
	if entries, _ := os.ReadDir(outside); len(entries) != 0 {
		t.Fatalf("a file was written outside the upload dir: %v", entries)
	}
}

func TestFSRepositoryDelete(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewFSRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	ctx := context.Background()
	name := strings.Repeat("d", 32) + ".webp"
	if err := repo.Save(ctx, 1, name, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, 2, name); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other shop: err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "1", name)); err != nil {
		t.Fatal("other shop's delete removed the file")
	}
	if err := repo.Delete(ctx, 1, name); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "1", name)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("after delete: err = %v, want the file gone", err)
	}
	if err := repo.Delete(ctx, 1, name); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete: err = %v", err)
	}
	if err := repo.Delete(ctx, 1, "../../../etc/passwd"); err == nil {
		t.Fatal("path escape must fail")
	}
}
