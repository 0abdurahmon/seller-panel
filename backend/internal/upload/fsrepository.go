package upload

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
)

// FSRepository keeps files on disk as <dir>/<shop_id>/<name>. Directories and files are world readable
// on purpose: nginx serves them to the public from the same volume.
type FSRepository struct {
	root *os.Root
}

// NewFSRepository creates dir if needed and confines all access to it.
func NewFSRepository(dir string) (*FSRepository, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("open upload dir: %w", err)
	}
	return &FSRepository{root: root}, nil
}

// Close releases the directory handle.
func (r *FSRepository) Close() error { return r.root.Close() }

func (r *FSRepository) Save(_ context.Context, shopID int64, name string, data []byte) error {
	dir := strconv.FormatInt(shopID, 10)
	if err := r.root.Mkdir(dir, 0o755); err != nil && !errors.Is(err, fs.ErrExist) {
		return fmt.Errorf("create shop dir: %w", err)
	}
	var tag [8]byte
	if _, err := rand.Read(tag[:]); err != nil {
		return fmt.Errorf("temp name: %w", err)
	}
	tmp := dir + "/.tmp-" + hex.EncodeToString(tag[:])
	f, err := r.root.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = r.root.Remove(tmp)
		return fmt.Errorf("write file: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = r.root.Remove(tmp)
		return fmt.Errorf("close file: %w", err)
	}
	if err := r.root.Rename(tmp, dir+"/"+name); err != nil {
		_ = r.root.Remove(tmp)
		return fmt.Errorf("publish file: %w", err)
	}
	return nil
}

func (r *FSRepository) Delete(_ context.Context, shopID int64, name string) error {
	if err := r.root.Remove(strconv.FormatInt(shopID, 10) + "/" + name); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrNotFound
		}
		return fmt.Errorf("delete file: %w", err)
	}
	return nil
}
