package database

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewSyncer(t *testing.T) {
	mirrorURL := "https://mirror.example.com"
	dbPath := "/tmp/test-db"
	arch := "x86_64"
	timeout := 30 * time.Second

	syncer := NewSyncer(mirrorURL, dbPath, arch, timeout)

	if syncer == nil {
		t.Fatal("NewSyncer returned nil")
	}
}

func TestDownloadDatabase(t *testing.T) {
	// Create a test HTTP server
	testData := []byte("fake database content")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/core/os/x86_64/core.db" {
			w.WriteHeader(http.StatusOK)
			w.Write(testData)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "pistacho-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	syncer := NewSyncer(server.URL, tmpDir, "x86_64", 30*time.Second)

	// Test successful download
	err = syncer.Sync([]string{"core"})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify file was created
	dbPath := filepath.Join(tmpDir, "core.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}

	// Verify content
	content, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("failed to read database file: %v", err)
	}
	if string(content) != string(testData) {
		t.Errorf("content mismatch: got %q, want %q", content, testData)
	}
}

func TestDownloadDatabaseNotFound(t *testing.T) {
	// Create a test HTTP server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "pistacho-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	syncer := NewSyncer(server.URL, tmpDir, "x86_64", 30*time.Second)

	err = syncer.Sync([]string{"nonexistent"})
	if err == nil {
		t.Error("expected error for 404 response, got nil")
	}
}

func TestDownloadDatabaseAtomicWrite(t *testing.T) {
	// Test that partial downloads don't corrupt the database
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// First call: write partial data then hang (will timeout)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("partial"))
			time.Sleep(5 * time.Second) // Force timeout
		} else {
			// Second call: write complete data
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("complete database"))
		}
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "pistacho-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Pre-create a "good" database
	dbPath := filepath.Join(tmpDir, "core.db")
	goodData := []byte("good database")
	if err := os.WriteFile(dbPath, goodData, 0644); err != nil {
		t.Fatalf("failed to create initial db: %v", err)
	}

	syncer := NewSyncer(server.URL, tmpDir, "x86_64", 1*time.Second)

	// This should fail due to timeout
	err = syncer.Sync([]string{"core"})
	if err == nil {
		t.Error("expected timeout error, got nil")
	}

	// Verify original database is still intact (atomic write should prevent corruption)
	content, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("failed to read database file: %v", err)
	}
	if string(content) != string(goodData) {
		t.Error("original database was corrupted by failed download")
	}
}

func TestSync(t *testing.T) {
	// Create test server with multiple repositories
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/core/os/x86_64/core.db":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("core database"))
		case "/extra/os/x86_64/extra.db":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("extra database"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "pistacho-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	syncer := NewSyncer(server.URL, tmpDir, "x86_64", 30*time.Second)

	err = syncer.Sync([]string{"core", "extra"})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify both databases were downloaded
	for _, repo := range []string{"core", "extra"} {
		dbPath := filepath.Join(tmpDir, repo+".db")
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			t.Errorf("database %s was not created", repo)
		}
	}
}

func TestSyncPartialFailure(t *testing.T) {
	// Test that Sync stops on first error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/core/os/x86_64/core.db" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("core database"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "pistacho-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	syncer := NewSyncer(server.URL, tmpDir, "x86_64", 30*time.Second)

	err = syncer.Sync([]string{"core", "nonexistent"})
	if err == nil {
		t.Error("expected error when one repo fails, got nil")
	}

	// Verify successful repo was downloaded
	corePath := filepath.Join(tmpDir, "core.db")
	if _, err := os.Stat(corePath); os.IsNotExist(err) {
		t.Error("core database should have been downloaded despite later failures")
	}
}

func TestDatabaseExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pistacho-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	syncer := NewSyncer("http://example.com", tmpDir, "x86_64", 30*time.Second)

	// Test non-existent database
	if syncer.DatabaseExists("core") {
		t.Error("DatabaseExists returned true for non-existent database")
	}

	// Create database file
	dbPath := filepath.Join(tmpDir, "core.db")
	if err := os.WriteFile(dbPath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	// Test existing database
	if !syncer.DatabaseExists("core") {
		t.Error("DatabaseExists returned false for existing database")
	}
}

func TestLastSyncTime(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pistacho-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	syncer := NewSyncer("http://example.com", tmpDir, "x86_64", 30*time.Second)

	// Test non-existent database
	syncTime, err := syncer.LastSyncTime("core")
	if err != nil {
		t.Errorf("expected no error for non-existent database, got: %v", err)
	}
	if !syncTime.IsZero() {
		t.Error("expected zero time for non-existent database")
	}

	// Create database file
	dbPath := filepath.Join(tmpDir, "core.db")
	beforeWrite := time.Now()
	if err := os.WriteFile(dbPath, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	afterWrite := time.Now()

	// Test existing database
	syncTime, err = syncer.LastSyncTime("core")
	if err != nil {
		t.Fatalf("LastSyncTime failed: %v", err)
	}

	// Verify time is reasonable (within our write window)
	if syncTime.Before(beforeWrite.Add(-time.Second)) || syncTime.After(afterWrite.Add(time.Second)) {
		t.Errorf("sync time %v not within expected range [%v, %v]", syncTime, beforeWrite, afterWrite)
	}
}

func TestSyncWithInvalidURL(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pistacho-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	syncer := NewSyncer("http://invalid-mirror-that-does-not-exist.local", tmpDir, "x86_64", 2*time.Second)

	err = syncer.Sync([]string{"core"})
	if err == nil {
		t.Error("expected error for invalid mirror URL, got nil")
	}
}

func TestSyncCreatesDBPath(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "pistacho-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use a subdirectory that doesn't exist yet
	dbPath := filepath.Join(tmpDir, "db")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test database"))
	}))
	defer server.Close()

	syncer := NewSyncer(server.URL, dbPath, "x86_64", 30*time.Second)

	err = syncer.Sync([]string{"core"})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("Sync did not create DBPath directory")
	}

	// Verify database file exists
	corePath := filepath.Join(dbPath, "core.db")
	if _, err := os.Stat(corePath); os.IsNotExist(err) {
		t.Error("database file was not created")
	}
}

func ExampleSyncer_Sync() {
	syncer := NewSyncer(
		"https://mirror.rackspace.com/archlinux",
		"/kod/db",
		"x86_64",
		300*time.Second,
	)

	if err := syncer.Sync([]string{"core", "extra"}); err != nil {
		fmt.Printf("sync failed: %v\n", err)
		return
	}

	fmt.Println("All databases synced successfully")
}

func ExampleSyncer_LastSyncTime() {
	syncer := NewSyncer(
		"https://mirror.rackspace.com/archlinux",
		"/kod/db",
		"x86_64",
		300*time.Second,
	)

	syncTime, err := syncer.LastSyncTime("core")
	if err != nil {
		fmt.Printf("error checking sync time: %v\n", err)
		return
	}

	if syncTime.IsZero() {
		fmt.Println("Database has never been synced")
	} else {
		fmt.Printf("Last synced: %v\n", syncTime)
	}
}
