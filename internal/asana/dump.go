package asana

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/ArnautVasile/asana-mini/internal/client"
	"github.com/ArnautVasile/asana-mini/internal/config"
	"github.com/ArnautVasile/asana-mini/internal/write"
)

func Run(ctx context.Context, interval time.Duration, cfg *config.Config) error {
	cl := client.New(cfg.AsanaPAT, cfg.HTTPTimeout)

	// immediate run
	if err := oneRun(ctx, cl, cfg.OutDir); err != nil {
		// log and continue; allow graceful stop on next select
		fmt.Printf("[run] error: %v\n", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			start := time.Now()
			if err := oneRun(ctx, cl, cfg.OutDir); err != nil {
				fmt.Printf("[run] error: %v\n", err)
			}
			fmt.Printf("[run] cycle finished in %v (interval %v)\n", time.Since(start), interval)
		}
	}
}

func oneRun(ctx context.Context, cl *client.Client, outDir string) error {
	workspaces, err := cl.Workspaces(ctx)
	if err != nil {
		return fmt.Errorf("workspaces: %w", err)
	}

	for _, ws := range workspaces {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wsDir := filepath.Join(outDir, write.SafeDirName(ws.Name, ws.Gid))

		users, err := cl.Users(ctx, ws.Gid)
		if err != nil {
			return fmt.Errorf("users %s: %w", ws.Gid, err)
		}
		if err := write.JSON(wsDir, "users.json", users); err != nil {
			return fmt.Errorf("write users %s: %w", ws.Gid, err)
		}

		projects, err := cl.Projects(ctx, ws.Gid)
		if err != nil {
			return fmt.Errorf("projects %s: %w", ws.Gid, err)
		}
		if err := write.JSON(wsDir, "projects.json", projects); err != nil {
			return fmt.Errorf("write projects %s: %w", ws.Gid, err)
		}

		fmt.Printf("[ws %s] wrote %d users, %d projects -> %s\n",
			ws.Name, len(users), len(projects), wsDir)
	}
	return nil
}
