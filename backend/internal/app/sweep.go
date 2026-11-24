package app

import (
	"context"
	"fmt"
	"time"

	"github.com/0abdurahmon/seller-panel/backend/internal/upload"
)

const (
	// sweepEvery is how often unused image files are looked for.
	sweepEvery = time.Hour
	// sweepFirst is the wait before the first run after the start.
	sweepFirst = time.Minute
	// sweepGrace protects a file that was just written and whose row is not saved yet.
	sweepGrace = time.Hour
)

// SweepImages deletes image files that no product photo and no shop logo uses and that are older than olderThan.
func (a *App) SweepImages(ctx context.Context, olderThan time.Duration) (upload.SweepResult, error) {
	return a.images.Sweep(ctx, a.imageInUse, olderThan, time.Now())
}

// imageInUse looks at the stored URLs by their last part (/<shop_id>/<name>), so a changed UPLOAD_PUBLIC_URL
// never makes a live image look unused.
func (a *App) imageInUse(ctx context.Context, shopID int64, name string) (bool, error) {
	suffix := fmt.Sprintf("/%d/%s", shopID, name)
	var used bool
	err := a.pool.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM product_photos pp JOIN products p ON p.id = pp.product_id
		                 WHERE p.shop_id = $1 AND right(pp.url, length($2::text)) = $2)
		     OR EXISTS (SELECT 1 FROM shops WHERE id = $1 AND right(logo_url, length($2::text)) = $2)`,
		shopID, suffix).Scan(&used)
	return used, err
}

func (a *App) sweepLoop(ctx context.Context) {
	timer := time.NewTimer(sweepFirst)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		}
		res, err := a.SweepImages(ctx, sweepGrace)
		switch {
		case err != nil && ctx.Err() == nil:
			a.logger.Error("sweep images", "error", err, "deleted", res.Deleted)
		case res.Deleted > 0:
			a.logger.Info("swept unused images", "deleted", res.Deleted, "checked", res.Checked)
		}
		timer.Reset(sweepEvery)
	}
}
