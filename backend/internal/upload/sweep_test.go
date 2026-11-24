package upload

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func hexName(c string) string { return strings.Repeat(c, 32) + ".webp" }

func usedBy(refs map[string]bool) ReferenceCheck {
	return func(_ context.Context, shopID int64, name string) (bool, error) {
		return refs[key(shopID, name)], nil
	}
}

func TestSweepRemovesOnlyOldUnreferencedFiles(t *testing.T) {
	ctx := context.Background()
	repo := newMemRepository()
	svc := NewService(repo, "/u")

	repo.put(1, hexName("a"), 3*time.Hour)                  // old, used by a product
	repo.put(1, hexName("b"), 3*time.Hour)                  // old, nothing uses it
	repo.put(1, hexName("c"), 5*time.Minute)                // new, nothing uses it yet: a save may be running
	repo.put(2, hexName("a"), 3*time.Hour)                  // same name in another shop, not used there
	repo.put(1, "notes.txt", 3*time.Hour)                   // not ours
	repo.put(1, strings.ToUpper(hexName("d")), 3*time.Hour) // not a name we generate
	repo.put(1, ".tmp-0123456789abcdef", 3*time.Hour)       // leftover of a crashed save
	repo.put(1, ".tmp-fedcba9876543210", 5*time.Minute)     // a save in progress

	res, err := svc.Sweep(ctx, usedBy(map[string]bool{key(1, hexName("a")): true}), time.Hour, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if res.Deleted != 3 {
		t.Fatalf("result = %+v, want 3 deleted (old unused file, same name in shop 2, old temp file)", res)
	}
	for _, f := range []struct {
		shop int64
		name string
		keep bool
	}{
		{1, hexName("a"), true}, {1, hexName("b"), false}, {1, hexName("c"), true}, {2, hexName("a"), false},
		{1, "notes.txt", true}, {1, strings.ToUpper(hexName("d")), true},
		{1, ".tmp-0123456789abcdef", false}, {1, ".tmp-fedcba9876543210", true},
	} {
		if _, has := repo.files[key(f.shop, f.name)]; has != f.keep {
			t.Errorf("%d/%s: stored = %v, want %v", f.shop, f.name, has, f.keep)
		}
	}
}

func TestSweepDeletesNothingItCannotCheck(t *testing.T) {
	repo := newMemRepository()
	svc := NewService(repo, "/u")
	repo.put(1, hexName("a"), 3*time.Hour)
	repo.put(1, hexName("b"), 3*time.Hour)

	down := func(context.Context, int64, string) (bool, error) { return false, errors.New("database is down") }
	if _, err := svc.Sweep(context.Background(), down, time.Hour, time.Now()); err == nil {
		t.Fatal("want the error of the check")
	}
	if repo.count() != 2 {
		t.Fatalf("%d files left, want both: a file that could not be checked must stay", repo.count())
	}
}

func TestSweepOnAnEmptyStore(t *testing.T) {
	res, err := NewService(newMemRepository(), "/u").Sweep(context.Background(), usedBy(nil), time.Hour, time.Now())
	if err != nil || res != (SweepResult{}) {
		t.Fatalf("%+v, %v", res, err)
	}
}

func TestFSRepositoryList(t *testing.T) {
	dir := t.TempDir()
	repo, err := NewFSRepository(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	ctx := context.Background()

	if got, err := repo.List(ctx); err != nil || len(got) != 0 {
		t.Fatalf("empty dir: %v, %v", got, err)
	}
	if err := repo.Save(ctx, 3, hexName("a"), []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, 3, hexName("b"), []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, 12, hexName("c"), []byte("x")); err != nil {
		t.Fatal(err)
	}
	// Things that are not shop files are not listed.
	if err := os.MkdirAll(filepath.Join(dir, "lost+found"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "3", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(filepath.Join(dir, "3", hexName("a")), old, old); err != nil {
		t.Fatal(err)
	}

	got, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("listed %d files, want 3: %+v", len(got), got)
	}
	for _, f := range got {
		if f.Name == hexName("a") && (f.ShopID != 3 || time.Since(f.ModTime) < 47*time.Hour) {
			t.Fatalf("file a = %+v, want shop 3 and the old modification time", f)
		}
		if f.ModTime.Location() != time.UTC {
			t.Fatalf("modification time of %s is in %v, want UTC", f.Name, f.ModTime.Location())
		}
	}
}
