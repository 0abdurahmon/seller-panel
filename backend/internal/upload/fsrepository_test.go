package upload

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFSRepositorySaveAndOpen(t *testing.T) {
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
	f, err := repo.Open(ctx, 5, name)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(f)
	f.Close()
	if string(got) != "data" {
		t.Fatalf("read %q", got)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "5"))
	if len(entries) != 1 {
		t.Fatalf("shop dir has %d entries, want 1 (no temp files left)", len(entries))
	}
	info, _ := os.Stat(filepath.Join(dir, "5", name))
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("mode = %v, want 0644, nginx must be able to read it", info.Mode().Perm())
	}
	dirInfo, _ := os.Stat(filepath.Join(dir, "5"))
	if dirInfo.Mode().Perm() != 0o755 {
		t.Fatalf("dir mode = %v, want 0755", dirInfo.Mode().Perm())
	}
}

func TestFSRepositoryIsolatesShopsAndRejectsEscapes(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewFSRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	ctx := context.Background()
	name := strings.Repeat("b", 32) + ".webp"
	if err := repo.Save(ctx, 1, name, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Open(ctx, 2, name); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other shop: err = %v", err)
	}
	if _, err := repo.Open(ctx, 1, "../../../etc/passwd"); err == nil {
		t.Fatal("path escape must fail")
	}
	outside := filepath.Join(t.TempDir(), "secret")
	_ = os.WriteFile(outside, []byte("secret"), 0o600)
	if err := os.Symlink(outside, filepath.Join(dir, "1", "link.webp")); err != nil {
		t.Fatal(err)
	}
	if f, err := repo.Open(ctx, 1, "link.webp"); err == nil {
		f.Close()
		t.Fatal("symlink out of the upload dir must not open")
	}
}

func TestFSRepositoryOpenDirectoryIsNotFound(t *testing.T) {
	repo, err := NewFSRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	_ = repo.Save(context.Background(), 1, strings.Repeat("c", 32)+".webp", []byte("x"))
	if _, err := repo.Open(context.Background(), 1, "."); err == nil {
		t.Fatal("opening a directory must fail")
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
	if _, err := repo.Open(ctx, 1, name); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after delete: err = %v", err)
	}
	if err := repo.Delete(ctx, 1, name); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete: err = %v", err)
	}
	if err := repo.Delete(ctx, 1, "../../../etc/passwd"); err == nil {
		t.Fatal("path escape must fail")
	}
}
