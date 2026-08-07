// Package download handles downloading Arch Linux packages from mirrors.
package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Downloader manages package downloads from Arch mirrors.
type Downloader struct {
	mirrorURL       string
	cachePath       string
	maxConcurrent   int
	downloadTimeout time.Duration
	httpClient      *http.Client
}

// NewDownloader creates a new package downloader.
func NewDownloader(mirrorURL, cachePath string, maxConcurrent int, timeout time.Duration) *Downloader {
	return &Downloader{
		mirrorURL:       mirrorURL,
		cachePath:       cachePath,
		maxConcurrent:   maxConcurrent,
		downloadTimeout: timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// PackageInfo contains information needed to download a package.
type PackageInfo struct {
	Name         string // e.g., "bash"
	Version      string // e.g., "5.3.9-1"
	Repo         string // e.g., "core", "extra"
	Architecture string // e.g., "x86_64", "aarch64", "any" — from desc %ARCH%
}

// DownloadPackage downloads a single package from the Arch mirror.
// Returns the local path to the downloaded package.
func (d *Downloader) DownloadPackage(pkg PackageInfo) (string, error) {
	// Ensure cache directory exists
	if err := os.MkdirAll(d.cachePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Construct package filename and URL
	// Format: bash-5.3.9-1-x86_64.pkg.tar.zst
	arch := pkg.Architecture
	if arch == "" {
		arch = "x86_64"
	}
	filename := fmt.Sprintf("%s-%s-%s.pkg.tar.zst", pkg.Name, pkg.Version, arch)
	// URL format: https://mirror.rackspace.com/archlinux/core/os/x86_64/bash-5.3.9-1-x86_64.pkg.tar.zst
	pkgURL := fmt.Sprintf("%s/%s/os/%s/%s", d.mirrorURL, pkg.Repo, arch, filename)

	fmt.Printf("Downloading %s/%s from %s...\n", pkg.Repo, filename, pkgURL)

	// Download the package
	resp, err := d.httpClient.Get(pkgURL)
	if err != nil {
		return "", fmt.Errorf("failed to download package: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("package download failed with status: %s", resp.Status)
	}

	// Create temporary file in cache
	tmpPath := filepath.Join(d.cachePath, filename+".tmp")
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	// Copy downloaded data to file with progress reporting
	written, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to write package: %w", err)
	}

	// Close file before rename
	tmpFile.Close()

	// Atomic rename to final location
	finalPath := filepath.Join(d.cachePath, filename)
	if err := os.Rename(tmpPath, finalPath); err != nil {
		os.Remove(tmpPath)
		return "", fmt.Errorf("failed to move package to final location: %w", err)
	}

	fmt.Printf("✓ Downloaded %s (%d bytes)\n", filename, written)
	return finalPath, nil
}

// DownloadPackages downloads multiple packages concurrently.
// Returns a map of package names to local paths and any error.
func (d *Downloader) DownloadPackages(packages []PackageInfo) (map[string]string, error) {
	results := make(map[string]string)
	resultsMutex := &sync.Mutex{}
	errorsMutex := &sync.Mutex{}
	var downloadErrors []error

	// Create a semaphore to limit concurrent downloads
	semaphore := make(chan struct{}, d.maxConcurrent)
	var wg sync.WaitGroup

	for _, pkg := range packages {
		wg.Add(1)
		go func(p PackageInfo) {
			defer wg.Done()

			// Acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Download package
			path, err := d.DownloadPackage(p)
			if err != nil {
				errorsMutex.Lock()
				downloadErrors = append(downloadErrors, fmt.Errorf("failed to download %s: %w", p.Name, err))
				errorsMutex.Unlock()
				return
			}

			// Store result
			resultsMutex.Lock()
			results[p.Name] = path
			resultsMutex.Unlock()
		}(pkg)
	}

	// Wait for all downloads to complete
	wg.Wait()

	// Return errors if any occurred
	if len(downloadErrors) > 0 {
		// Collect all errors
		var errMsg string
		for i, err := range downloadErrors {
			errMsg += err.Error()
			if i < len(downloadErrors)-1 {
				errMsg += "; "
			}
		}
		return results, fmt.Errorf("download errors: %s", errMsg)
	}

	return results, nil
}

// PackageExists checks if a package has already been downloaded.
func (d *Downloader) PackageExists(pkg PackageInfo) bool {
	arch := pkg.Architecture
	if arch == "" {
		arch = "x86_64"
	}
	filename := fmt.Sprintf("%s-%s-%s.pkg.tar.zst", pkg.Name, pkg.Version, arch)
	pkgPath := filepath.Join(d.cachePath, filename)
	_, err := os.Stat(pkgPath)
	return err == nil
}

// GetLocalPath returns the local path where a package would be cached.
func (d *Downloader) GetLocalPath(pkg PackageInfo) string {
	arch := pkg.Architecture
	if arch == "" {
		arch = "x86_64"
	}
	filename := fmt.Sprintf("%s-%s-%s.pkg.tar.zst", pkg.Name, pkg.Version, arch)
	return filepath.Join(d.cachePath, filename)
}

// CleanCache removes old or unused packages from the cache.
// This is planned for future implementation when package store and registry
// integration is complete and a keep-versions policy is established.
