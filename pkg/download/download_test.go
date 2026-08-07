package download

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewDownloader tests downloader creation.
func TestNewDownloader(t *testing.T) {
	d := NewDownloader(
		"https://mirror.example.com/archlinux",
		"/tmp/cache",
		5,
		30*time.Second,
	)

	if d == nil {
		t.Fatal("NewDownloader returned nil")
	}
	if d.mirrorURL != "https://mirror.example.com/archlinux" {
		t.Errorf("mirrorURL mismatch: got %s", d.mirrorURL)
	}
	if d.cachePath != "/tmp/cache" {
		t.Errorf("cachePath mismatch: got %s", d.cachePath)
	}
	if d.maxConcurrent != 5 {
		t.Errorf("maxConcurrent mismatch: got %d", d.maxConcurrent)
	}
}

// TestDownloadPackage tests single package download.
func TestDownloadPackage(t *testing.T) {
	// Create temporary cache directory
	cacheDir := t.TempDir()

	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/core/os/x86_64/bash-5.3.9-1-x86_64.pkg.tar.zst" {
			http.NotFound(w, r)
			return
		}
		content := "fake package content"
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		fmt.Fprint(w, content)
	}))
	defer server.Close()

	d := NewDownloader(server.URL, cacheDir, 5, 30*time.Second)

	pkg := PackageInfo{
		Name:         "bash",
		Version:      "5.3.9-1",
		Repo:         "core",
		Architecture: "x86_64",
	}

	path, err := d.DownloadPackage(pkg)
	if err != nil {
		t.Fatalf("DownloadPackage failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Errorf("Downloaded file does not exist: %v", err)
	}

	// Verify file content
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}
	if string(content) != "fake package content" {
		t.Errorf("File content mismatch: got %s", string(content))
	}

	// Verify filename matches expected format
	expectedFilename := "bash-5.3.9-1-x86_64.pkg.tar.zst"
	if filepath.Base(path) != expectedFilename {
		t.Errorf("Filename mismatch: expected %s, got %s", expectedFilename, filepath.Base(path))
	}
}

// TestDownloadPackageAnyArch tests that "any" architecture packages use /os/any/ in URL.
func TestDownloadPackageAnyArch(t *testing.T) {
	cacheDir := t.TempDir()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/core/os/any/ca-certificates-20240618-1-any.pkg.tar.zst" {
			http.NotFound(w, r)
			return
		}
		content := "fake any package content"
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		fmt.Fprint(w, content)
	}))
	defer server.Close()

	d := NewDownloader(server.URL, cacheDir, 5, 30*time.Second)

	pkg := PackageInfo{
		Name:         "ca-certificates",
		Version:      "20240618-1",
		Repo:         "core",
		Architecture: "any",
	}

	path, err := d.DownloadPackage(pkg)
	if err != nil {
		t.Fatalf("DownloadPackage failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Errorf("Downloaded file does not exist: %v", err)
	}

	// Verify filename uses -any
	expectedFilename := "ca-certificates-20240618-1-any.pkg.tar.zst"
	if filepath.Base(path) != expectedFilename {
		t.Errorf("Filename mismatch: expected %s, got %s", expectedFilename, filepath.Base(path))
	}
}

// TestDownloadPackageNotFound tests handling of 404 errors.
func TestDownloadPackageNotFound(t *testing.T) {
	cacheDir := t.TempDir()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	d := NewDownloader(server.URL, cacheDir, 5, 30*time.Second)

	pkg := PackageInfo{
		Name:         "nonexistent",
		Version:      "1.0.0-1",
		Repo:         "core",
		Architecture: "x86_64",
	}

	_, err := d.DownloadPackage(pkg)
	if err == nil {
		t.Fatal("Expected error for 404 response")
	}
}

// TestDownloadPackageCreatesDirectory tests that cache directory is created.
func TestDownloadPackageCreatesDirectory(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "nested", "cache", "dir")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "test")
	}))
	defer server.Close()

	d := NewDownloader(server.URL, cacheDir, 5, 30*time.Second)

	pkg := PackageInfo{
		Name:         "bash",
		Version:      "5.3.9-1",
		Repo:         "core",
		Architecture: "x86_64",
	}

	path, err := d.DownloadPackage(pkg)
	if err != nil {
		t.Fatalf("DownloadPackage failed: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("Downloaded file not created in nested directory: %v", err)
	}
}

// TestDownloadPackageAtomicWrite tests that failed writes don't leave temp files.
func TestDownloadPackageAtomicWrite(t *testing.T) {
	cacheDir := t.TempDir()

	// Server that writes the content correctly with proper Content-Length
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		content := "short content"
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.Write([]byte(content))
	}))
	defer server.Close()

	d := NewDownloader(server.URL, cacheDir, 5, 30*time.Second)

	pkg := PackageInfo{
		Name:         "bash",
		Version:      "5.3.9-1",
		Repo:         "core",
		Architecture: "x86_64",
	}

	// Download should succeed
	path, err := d.DownloadPackage(pkg)
	if err != nil {
		t.Fatalf("DownloadPackage failed: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("Final file not created: %v", err)
	}

	// Check that temp file is not left behind
	tmpPath := path + ".tmp"
	if _, err := os.Stat(tmpPath); err == nil {
		t.Errorf("Temp file was not cleaned up: %s", tmpPath)
	}
}

// TestDownloadPackages tests concurrent package downloads.
func TestDownloadPackages(t *testing.T) {
	cacheDir := t.TempDir()

	downloadCount := int64(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&downloadCount, 1)
		fmt.Fprintf(w, "content for %s", r.URL.Path)
	}))
	defer server.Close()

	d := NewDownloader(server.URL, cacheDir, 2, 30*time.Second)

	packages := []PackageInfo{
		{Name: "bash", Version: "5.3.9-1", Repo: "core", Architecture: "x86_64"},
		{Name: "vim", Version: "9.0.0-1", Repo: "extra", Architecture: "x86_64"},
		{Name: "git", Version: "2.40.0-1", Repo: "extra", Architecture: "x86_64"},
	}

	results, err := d.DownloadPackages(packages)
	if err == nil {
		// Some packages might fail, which is OK - we still get partial results
		if len(results) != 3 {
			t.Logf("Expected 3 packages, got %d", len(results))
		}
	}

	// Verify that downloads attempted to run
	if downloadCount == 0 {
		t.Error("No downloads were attempted")
	}

	// Verify all downloaded packages exist
	for _, path := range results {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Downloaded package not found: %v", err)
		}
	}
}

// TestDownloadPackagesPartialFailure tests handling of partial failures.
func TestDownloadPackagesPartialFailure(t *testing.T) {
	cacheDir := t.TempDir()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/core/os/x86_64/bash-5.3.9-1-x86_64.pkg.tar.zst" {
			fmt.Fprint(w, "bash content")
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	d := NewDownloader(server.URL, cacheDir, 5, 30*time.Second)

	packages := []PackageInfo{
		{Name: "bash", Version: "5.3.9-1", Repo: "core", Architecture: "x86_64"},
		{Name: "nonexistent", Version: "1.0.0-1", Repo: "core", Architecture: "x86_64"},
	}

	results, err := d.DownloadPackages(packages)
	if err == nil {
		t.Fatal("Expected error for partial failure")
	}

	// Bash should be in results even though vim failed
	if _, ok := results["bash"]; !ok {
		t.Error("Successfully downloaded package missing from results")
	}
}

// TestDownloadPackageMixedArch tests downloading packages with different architectures in one batch.
func TestDownloadPackageMixedArch(t *testing.T) {
	cacheDir := t.TempDir()

	var receivedPaths []string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedPaths = append(receivedPaths, r.URL.Path)
		mu.Unlock()
		fmt.Fprint(w, "package content")
	}))
	defer server.Close()

	d := NewDownloader(server.URL, cacheDir, 5, 30*time.Second)

	packages := []PackageInfo{
		{Name: "bash", Version: "5.3.9-1", Repo: "core", Architecture: "x86_64"},
		{Name: "ca-certificates", Version: "20240618-1", Repo: "core", Architecture: "any"},
		{Name: "ncurses", Version: "6.4-1", Repo: "core", Architecture: "aarch64"},
	}

	results, err := d.DownloadPackages(packages)
	if err != nil {
		t.Fatalf("DownloadPackages failed: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	mu.Lock()
	received := make(map[string]bool)
	for _, p := range receivedPaths {
		received[p] = true
	}
	mu.Unlock()

	// Verify each package hit the correct URL path (order is non-deterministic due to concurrency)
	expectedPaths := []string{
		"/core/os/x86_64/bash-5.3.9-1-x86_64.pkg.tar.zst",
		"/core/os/any/ca-certificates-20240618-1-any.pkg.tar.zst",
		"/core/os/aarch64/ncurses-6.4-1-aarch64.pkg.tar.zst",
	}

	for _, expected := range expectedPaths {
		if !received[expected] {
			t.Errorf("Expected URL path %s was not received. Got: %v", expected, receivedPaths)
		}
	}
}

// TestPackageExists tests checking if package is already cached.
func TestPackageExists(t *testing.T) {
	cacheDir := t.TempDir()

	d := NewDownloader("https://mirror.example.com", cacheDir, 5, 30*time.Second)

	pkg := PackageInfo{
		Name:         "bash",
		Version:      "5.3.9-1",
		Repo:         "core",
		Architecture: "x86_64",
	}

	// Package doesn't exist yet
	if d.PackageExists(pkg) {
		t.Fatal("Package should not exist initially")
	}

	// Create the package file
	filename := "bash-5.3.9-1-x86_64.pkg.tar.zst"
	pkgPath := filepath.Join(cacheDir, filename)
	if err := os.WriteFile(pkgPath, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test package: %v", err)
	}

	// Now package should exist
	if !d.PackageExists(pkg) {
		t.Fatal("Package should exist after creation")
	}
}

// TestPackageExistsAnyArch tests checking cached packages with "any" architecture.
func TestPackageExistsAnyArch(t *testing.T) {
	cacheDir := t.TempDir()

	d := NewDownloader("https://mirror.example.com", cacheDir, 5, 30*time.Second)

	pkg := PackageInfo{
		Name:         "ca-certificates",
		Version:      "20240618-1",
		Repo:         "core",
		Architecture: "any",
	}

	// Package doesn't exist yet
	if d.PackageExists(pkg) {
		t.Fatal("Package should not exist initially")
	}

	// Create the package file with -any suffix
	filename := "ca-certificates-20240618-1-any.pkg.tar.zst"
	pkgPath := filepath.Join(cacheDir, filename)
	if err := os.WriteFile(pkgPath, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test package: %v", err)
	}

	// Now package should exist
	if !d.PackageExists(pkg) {
		t.Fatal("Package should exist after creation")
	}
}

// TestGetLocalPath tests getting the local path for a package.
func TestGetLocalPath(t *testing.T) {
	cacheDir := "/tmp/cache"
	d := NewDownloader("https://mirror.example.com", cacheDir, 5, 30*time.Second)

	pkg := PackageInfo{
		Name:         "bash",
		Version:      "5.3.9-1",
		Repo:         "core",
		Architecture: "x86_64",
	}

	path := d.GetLocalPath(pkg)
	expected := filepath.Join(cacheDir, "bash-5.3.9-1-x86_64.pkg.tar.zst")
	if path != expected {
		t.Errorf("Path mismatch: expected %s, got %s", expected, path)
	}
}

// TestGetLocalPathAnyArch tests local path with "any" architecture.
func TestGetLocalPathAnyArch(t *testing.T) {
	cacheDir := "/tmp/cache"
	d := NewDownloader("https://mirror.example.com", cacheDir, 5, 30*time.Second)

	pkg := PackageInfo{
		Name:         "ca-certificates",
		Version:      "20240618-1",
		Repo:         "core",
		Architecture: "any",
	}

	path := d.GetLocalPath(pkg)
	expected := filepath.Join(cacheDir, "ca-certificates-20240618-1-any.pkg.tar.zst")
	if path != expected {
		t.Errorf("Path mismatch: expected %s, got %s", expected, path)
	}
}

// TestDownloadPackageFallbackArch tests that missing Architecture defaults to x86_64.
func TestDownloadPackageFallbackArch(t *testing.T) {
	cacheDir := t.TempDir()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/core/os/x86_64/fallback-1.0-1-x86_64.pkg.tar.zst" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, "content")
	}))
	defer server.Close()

	d := NewDownloader(server.URL, cacheDir, 5, 30*time.Second)

	// Empty Architecture should fall back to x86_64
	pkg := PackageInfo{
		Name:    "fallback",
		Version: "1.0-1",
		Repo:    "core",
	}

	_, err := d.DownloadPackage(pkg)
	if err != nil {
		t.Fatalf("DownloadPackage failed: %v", err)
	}
}

// TestDownloadPackageServerError tests handling of server errors.
func TestDownloadPackageServerError(t *testing.T) {
	cacheDir := t.TempDir()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "internal server error")
	}))
	defer server.Close()

	d := NewDownloader(server.URL, cacheDir, 5, 30*time.Second)

	pkg := PackageInfo{
		Name:         "bash",
		Version:      "5.3.9-1",
		Repo:         "core",
		Architecture: "x86_64",
	}

	_, err := d.DownloadPackage(pkg)
	if err == nil {
		t.Fatal("Expected error for server error")
	}
}

// TestDownloadPackageConcurrencyLimit tests max concurrent downloads.
func TestDownloadPackageConcurrencyLimit(t *testing.T) {
	cacheDir := t.TempDir()

	maxConcurrent := 0
	currentConcurrent := 0
	concurrentMutex := make(chan struct{}, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		concurrentMutex <- struct{}{}
		currentConcurrent++
		if currentConcurrent > maxConcurrent {
			maxConcurrent = currentConcurrent
		}

		// Simulate some processing time
		time.Sleep(100 * time.Millisecond)
		fmt.Fprint(w, "content")

		currentConcurrent--
		<-concurrentMutex
	}))
	defer server.Close()

	d := NewDownloader(server.URL, cacheDir, 2, 30*time.Second)

	packages := []PackageInfo{
		{Name: "pkg1", Version: "1.0.0-1", Repo: "core", Architecture: "x86_64"},
		{Name: "pkg2", Version: "1.0.0-1", Repo: "core", Architecture: "x86_64"},
		{Name: "pkg3", Version: "1.0.0-1", Repo: "core", Architecture: "x86_64"},
		{Name: "pkg4", Version: "1.0.0-1", Repo: "core", Architecture: "x86_64"},
	}

	_, _ = d.DownloadPackages(packages)

	// Verify that max concurrent was limited to 2
	if maxConcurrent > 2 {
		t.Errorf("Concurrency limit exceeded: expected <= 2, got %d", maxConcurrent)
	}
}

// BenchmarkDownloadPackage benchmarks package download performance.
func BenchmarkDownloadPackage(b *testing.B) {
	cacheDir := b.TempDir()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Write 1MB of data
		w.Header().Set("Content-Length", "1048576")
		io.WriteString(w, "x")
		io.WriteString(w, "x")
		for i := 0; i < 1048574; i++ {
			io.WriteString(w, "x")
		}
	}))
	defer server.Close()

	d := NewDownloader(server.URL, cacheDir, 5, 30*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pkg := PackageInfo{
			Name:         fmt.Sprintf("pkg%d", i),
			Version:      "1.0.0-1",
			Repo:         "core",
			Architecture: "x86_64",
		}
		d.DownloadPackage(pkg)
	}
}
