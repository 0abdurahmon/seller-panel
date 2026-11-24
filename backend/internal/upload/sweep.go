package upload

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// tempPrefix starts the name of a file that is being saved; one that stays behind is the leftover of a crash.
const tempPrefix = ".tmp-"

// ReferenceCheck says whether a product or the shop still uses the file.
type ReferenceCheck func(ctx context.Context, shopID int64, name string) (bool, error)

// SweepResult counts what a sweep looked at and removed.
type SweepResult struct {
	Checked int
	Deleted int
}

// Sweep deletes the files nothing refers to: a crash between writing a file and saving the row that names it
// leaves one behind. Files younger than olderThan are never touched, so a file that is being saved right now is
// safe; files whose name this service did not generate are left alone. It stops at the first error and removes
// nothing it could not check.
func (s *Service) Sweep(ctx context.Context, used ReferenceCheck, olderThan time.Duration, now time.Time) (SweepResult, error) {
	files, err := s.repo.List(ctx)
	if err != nil {
		return SweepResult{}, err
	}
	var res SweepResult
	for _, f := range files {
		if now.Sub(f.ModTime) < olderThan {
			continue
		}
		temp := strings.HasPrefix(f.Name, tempPrefix)
		if !temp && !fileName.MatchString(f.Name) {
			continue
		}
		res.Checked++
		if !temp {
			inUse, err := used(ctx, f.ShopID, f.Name)
			if err != nil {
				return res, fmt.Errorf("check %d/%s: %w", f.ShopID, f.Name, err)
			}
			if inUse {
				continue
			}
		}
		if err := s.repo.Delete(ctx, f.ShopID, f.Name); err != nil && !errors.Is(err, ErrNotFound) {
			return res, fmt.Errorf("delete %d/%s: %w", f.ShopID, f.Name, err)
		}
		res.Deleted++
	}
	return res, nil
}
