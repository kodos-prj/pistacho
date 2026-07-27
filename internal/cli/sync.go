// Package cli implements command-line interface commands for pith.
package cli

import (
	"fmt"
	"time"

	"github.com/kodos-prj/pistacho/pkg/config"
	"github.com/kodos-prj/pistacho/pkg/database"
)

// SyncCommand implements the 'pith sync' command.
// It downloads package databases from configured Arch mirrors.
type SyncCommand struct {
	config *config.Config
}

// NewSyncCommand creates a new sync command instance.
func NewSyncCommand(cfg *config.Config) *SyncCommand {
	return &SyncCommand{
		config: cfg,
	}
}

// Execute runs the sync command.
// It downloads all configured repository databases from the mirror.
func (s *SyncCommand) Execute() error {
	fmt.Println("Syncing package databases...")
	fmt.Printf("Mirror: %s\n", s.config.MirrorURL)
	fmt.Printf("Architecture: %s\n", s.config.Architecture)
	fmt.Printf("Repositories: %v\n", s.config.Repositories)
	fmt.Println()

	// Create syncer
	syncer := database.NewSyncer(
		s.config.MirrorURL,
		s.config.DBPath,
		s.config.Architecture,
		time.Duration(s.config.DownloadTimeout)*time.Second,
	)

	// Perform sync
	if err := syncer.Sync(s.config.Repositories); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Println("\n✓ All databases synced successfully")
	return nil
}

// ExecuteWithForce runs the sync command with force flag.
// Force flag downloads databases even if they already exist.
func (s *SyncCommand) ExecuteWithForce() error {
	// For now, this is the same as Execute since we always re-download
	// In the future, we could check timestamps and skip if recent
	return s.Execute()
}

// ShowStatus displays the current sync status.
// It shows when each repository was last synced.
func (s *SyncCommand) ShowStatus() error {
	fmt.Println("Database sync status:")
	fmt.Printf("Database path: %s\n\n", s.config.DBPath)

	syncer := database.NewSyncer(
		s.config.MirrorURL,
		s.config.DBPath,
		s.config.Architecture,
		time.Duration(s.config.DownloadTimeout)*time.Second,
	)

	for _, repo := range s.config.Repositories {
		if syncer.DatabaseExists(repo) {
			syncTime, err := syncer.LastSyncTime(repo)
			if err != nil {
				fmt.Printf("  %s: Error checking sync time: %v\n", repo, err)
				continue
			}

			if syncTime.IsZero() {
				fmt.Printf("  %s: Never synced\n", repo)
			} else {
				fmt.Printf("  %s: Last synced %v ago\n", repo, time.Since(syncTime).Round(time.Second))
			}
		} else {
			fmt.Printf("  %s: Not synced\n", repo)
		}
	}

	return nil
}
