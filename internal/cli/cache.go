// Package cli implements command-line interface commands for pith.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kodos-prj/pistacho/pkg/config"
)

// CacheCommand implements the 'pith cache-clean' command.
// It manages and cleans the package download cache.
type CacheCommand struct {
	config *config.Config
}

// NewCacheCommand creates a new cache command instance.
func NewCacheCommand(cfg *config.Config) *CacheCommand {
	return &CacheCommand{
		config: cfg,
	}
}

// CacheOptions holds command-line options for cache operations.
type CacheOptions struct {
	DryRun  bool   // Preview mode: don't make changes
	Verbose bool   // Show detailed output
	Force   bool   // Skip confirmation prompt
	Action  string // "clean" (remove all), "list" (show contents), "prune" (remove old)
}

// CacheFile represents a cached package file
type CacheFile struct {
	Name  string
	Path  string
	Size  int64
	IsOld bool // True if older than retention period
}

// CacheSummary reports cache operation results
type CacheSummary struct {
	TotalFiles     int
	TotalSize      int64
	FilesRemoved   int
	SpaceFreed     int64
	FilesWithError int
	RetentionDays  int
	RemovedFiles   []string
	SkippedFiles   []string
	CachedFiles    []CacheFile
}

// Execute runs the cache command.
func (c *CacheCommand) Execute(opts *CacheOptions) (*CacheSummary, error) {
	if opts == nil {
		opts = &CacheOptions{Action: "clean"}
	}

	summary := &CacheSummary{
		RemovedFiles: []string{},
		SkippedFiles: []string{},
		CachedFiles:  []CacheFile{},
	}

	// Validate cache path exists
	if c.config.CachePath == "" {
		c.config.CachePath = filepath.Join(c.config.BaseDir, "cache")
	}

	// Check if cache directory exists
	stat, err := os.Stat(c.config.CachePath)
	if err != nil {
		if os.IsNotExist(err) {
			if opts.Verbose {
				fmt.Println("Cache directory does not exist. Nothing to clean.")
			}
			return summary, nil
		}
		return nil, fmt.Errorf("failed to access cache directory: %w", err)
	}

	if !stat.IsDir() {
		return nil, fmt.Errorf("cache path is not a directory: %s", c.config.CachePath)
	}

	// Load cache files
	cacheFiles, err := c.listCacheFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to list cache files: %w", err)
	}

	summary.TotalFiles = len(cacheFiles)
	for _, cf := range cacheFiles {
		summary.TotalSize += cf.Size
		summary.CachedFiles = append(summary.CachedFiles, cf)
	}

	// Handle different actions
	switch opts.Action {
	case "list":
		return c.handleList(summary, opts)
	case "clean":
		return c.handleClean(cacheFiles, summary, opts)
	case "prune":
		return c.handlePrune(cacheFiles, summary, opts)
	default:
		return nil, fmt.Errorf("unknown cache action: %s", opts.Action)
	}
}

// listCacheFiles lists all files in the cache directory
func (c *CacheCommand) listCacheFiles() ([]CacheFile, error) {
	var files []CacheFile

	entries, err := os.ReadDir(c.config.CachePath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip directories
		}

		info, err := entry.Info()
		if err != nil {
			continue // Skip on error
		}

		// Only include .pkg.tar.zst files
		if !strings.HasSuffix(entry.Name(), ".pkg.tar.zst") {
			continue
		}

		files = append(files, CacheFile{
			Name: entry.Name(),
			Path: filepath.Join(c.config.CachePath, entry.Name()),
			Size: info.Size(),
		})
	}

	return files, nil
}

// handleList shows cache contents without modifying anything
func (c *CacheCommand) handleList(summary *CacheSummary, opts *CacheOptions) (*CacheSummary, error) {
	if len(summary.CachedFiles) == 0 {
		fmt.Println("Cache is empty.")
		return summary, nil
	}

	fmt.Printf("Cache Contents (%d files, %.2f GB):\n\n", summary.TotalFiles, float64(summary.TotalSize)/(1024*1024*1024))

	if opts.Verbose {
		fmt.Println("┌──────────────────────────────────┬────────────┐")
		fmt.Println("│ Filename                         │ Size       │")
		fmt.Println("├──────────────────────────────────┼────────────┤")

		for _, cf := range summary.CachedFiles {
			sizeStr := c.formatSize(cf.Size)
			fmt.Printf("│ %-32s │ %10s │\n", truncateString(cf.Name, 32), sizeStr)
		}

		fmt.Println("└──────────────────────────────────┴────────────┘")
	} else {
		for _, cf := range summary.CachedFiles {
			sizeStr := c.formatSize(cf.Size)
			fmt.Printf("  %s (%s)\n", cf.Name, sizeStr)
		}
	}

	return summary, nil
}

// handleClean removes all cache files
func (c *CacheCommand) handleClean(cacheFiles []CacheFile, summary *CacheSummary, opts *CacheOptions) (*CacheSummary, error) {
	if len(cacheFiles) == 0 {
		if opts.Verbose {
			fmt.Println("Cache is empty. Nothing to clean.")
		}
		return summary, nil
	}

	// Show plan
	fmt.Printf("✓ Found %d cached files (%.2f GB)\n\n", len(cacheFiles), float64(summary.TotalSize)/(1024*1024*1024))

	if opts.Verbose {
		c.showCachePlan(cacheFiles)
	}

	// Ask for confirmation if not --force
	if !opts.Force && !opts.DryRun {
		if !c.askForConfirmation("cache clean", len(cacheFiles)) {
			fmt.Println("Cache clean cancelled.")
			return summary, nil
		}
	}

	// If dry-run, stop here
	if opts.DryRun {
		return summary, nil
	}

	// Execute cleanup
	for _, cf := range cacheFiles {
		err := os.Remove(cf.Path)
		if err != nil {
			if opts.Verbose {
				fmt.Printf("  ✗ Failed to remove %s: %v\n", cf.Name, err)
			}
			summary.SkippedFiles = append(summary.SkippedFiles, cf.Name)
			summary.FilesWithError++
		} else {
			summary.RemovedFiles = append(summary.RemovedFiles, cf.Name)
			summary.FilesRemoved++
			summary.SpaceFreed += cf.Size
			if opts.Verbose {
				fmt.Printf("  ✓ Removed %s (%.2f MB)\n", cf.Name, float64(cf.Size)/(1024*1024))
			}
		}
	}

	// Show results
	c.showCacheResults(summary, opts.Verbose, len(cacheFiles) > 0)

	return summary, nil
}

// handlePrune removes only old cached files (kept for compatibility)
func (c *CacheCommand) handlePrune(cacheFiles []CacheFile, summary *CacheSummary, opts *CacheOptions) (*CacheSummary, error) {
	// For now, prune acts like clean (removes all unused cache)
	// In future could implement retention period logic
	if len(cacheFiles) == 0 {
		if opts.Verbose {
			fmt.Println("Cache is empty. Nothing to prune.")
		}
		return summary, nil
	}

	return c.handleClean(cacheFiles, summary, opts)
}

// showCachePlan displays what will be cleaned
func (c *CacheCommand) showCachePlan(cacheFiles []CacheFile) {
	fmt.Println("Files to remove:")
	fmt.Println("┌──────────────────────────────────┬────────────┐")
	fmt.Println("│ Filename                         │ Size       │")
	fmt.Println("├──────────────────────────────────┼────────────┤")

	for _, cf := range cacheFiles {
		sizeStr := c.formatSize(cf.Size)
		fmt.Printf("│ %-32s │ %10s │\n", truncateString(cf.Name, 32), sizeStr)
	}

	fmt.Println("└──────────────────────────────────┴────────────┘")
	fmt.Println()
}

// showCacheResults displays summary after cache operation
func (c *CacheCommand) showCacheResults(summary *CacheSummary, verbose bool, executed bool) {
	if verbose || executed {
		fmt.Println()
		if summary.FilesRemoved > 0 {
			fmt.Printf("✓ Cache Clean Summary:\n")
			fmt.Printf("  Files removed:  %d\n", summary.FilesRemoved)
			fmt.Printf("  Space freed:    %.2f GB\n", float64(summary.SpaceFreed)/(1024*1024*1024))
			if summary.FilesWithError > 0 {
				fmt.Printf("  Files with error: %d\n", summary.FilesWithError)
			}
		} else {
			fmt.Println("✓ No cache files needed cleaning")
		}
	} else {
		if summary.FilesRemoved > 0 {
			fmt.Printf("✓ Cache clean complete: %d files removed, %.2f GB freed\n",
				summary.FilesRemoved,
				float64(summary.SpaceFreed)/(1024*1024*1024))
		} else {
			fmt.Println("✓ Cache is clean")
		}
	}
}

// formatSize converts bytes to human-readable format
func (c *CacheCommand) formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(bytes)/(1024))
	}
	if bytes < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	}
	return fmt.Sprintf("%.1f GB", float64(bytes)/(1024*1024*1024))
}

// askForConfirmation prompts user before cache operation
func (c *CacheCommand) askForConfirmation(operation string, count int) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Proceed with %s of %d file(s)? (y/n) ", operation, count)

	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// truncateString truncates a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
