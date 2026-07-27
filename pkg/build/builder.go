// Package build provides build system integration for Chisel.
// builder.go implements the build manager for compiling AUR packages.
package build

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/kodos-prj/pistacho/pkg/aur"
)

// BuildManager manages the build process for AUR packages
type BuildManager struct {
	buildCacheDir  string // /kod/build-cache/ - persistent cache for build directories
	logsDir        string // /kod/build-logs/ - build logs
	gitHandler     *aur.GitHandler
	pkgbuildParser *aur.PKGBUILDParser
}

// NewBuildManager creates a new build manager
func NewBuildManager(buildCacheDir, logsDir string) (*BuildManager, error) {
	// Create directories if they don't exist
	if err := os.MkdirAll(buildCacheDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create build cache directory: %w", err)
	}

	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	return &BuildManager{
		buildCacheDir:  buildCacheDir,
		logsDir:        logsDir,
		gitHandler:     aur.NewGitHandler(buildCacheDir),
		pkgbuildParser: aur.NewPKGBUILDParser(),
	}, nil
}

// BuildResult contains information about a build result
type BuildResult struct {
	PackageName    string
	PackageVersion string
	BuildStatus    string // "success", "failed"
	ArtifactPath   string // Path to built .pkg.tar.zst
	LogPath        string // Path to build log
	StartTime      time.Time
	EndTime        time.Time
	BuildLog       string // Full build output (for error reporting)
}

// BuildAURPackage builds an AUR package and returns the path to the built artifact
// pkgName: package name
// version: package version
// pkgbuildPath: path to cloned PKGBUILD directory
// Returns: path to built .pkg.tar.zst file, error
func (bm *BuildManager) BuildAURPackage(pkgName, version, pkgbuildPath string) (string, error) {
	if pkgName == "" {
		return "", fmt.Errorf("package name cannot be empty")
	}

	if pkgbuildPath == "" {
		return "", fmt.Errorf("PKGBUILD path cannot be empty")
	}

	// Verify PKGBUILD exists
	if err := bm.gitHandler.VerifyPKGBUILD(pkgbuildPath); err != nil {
		return "", fmt.Errorf("invalid PKGBUILD: %w", err)
	}

	// Create unique build directory with timestamp
	timestamp := time.Now().Unix()
	uniqueBuildDir := filepath.Join(bm.buildCacheDir, fmt.Sprintf("%s-%s-%d", pkgName, version, timestamp))

	// Create build directory
	if err := os.MkdirAll(uniqueBuildDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create build directory: %w", err)
	}

	// Copy PKGBUILD and all files to build directory
	if err := bm.copyBuildFiles(pkgbuildPath, uniqueBuildDir); err != nil {
		return "", fmt.Errorf("failed to copy build files: %w", err)
	}

	// Execute build
	logPath := filepath.Join(bm.logsDir, fmt.Sprintf("%s-%s.log", pkgName, version))
	buildOutput, err := bm.executeBuild(uniqueBuildDir, logPath)
	if err != nil {
		// Save error log for debugging
		_ = os.WriteFile(logPath, []byte(buildOutput), 0644)
		return "", fmt.Errorf("build failed for %s/%s: %w\nBuild output:\n%s", pkgName, version, err, buildOutput)
	}

	// Find built artifact in build directory
	artifactPath, err := bm.findBuiltArtifact(uniqueBuildDir, pkgName)
	if err != nil {
		return "", fmt.Errorf("failed to find built artifact: %w", err)
	}

	// Verify artifact exists and is valid
	if _, err := os.Stat(artifactPath); err != nil {
		return "", fmt.Errorf("artifact not found after build: %s", artifactPath)
	}

	// Save successful build log
	_ = os.WriteFile(logPath, []byte(buildOutput), 0644)

	return artifactPath, nil
}

// copyBuildFiles copies PKGBUILD and associated files to build directory
func (bm *BuildManager) copyBuildFiles(srcDir, destDir string) error {
	// Walk through source directory and copy all files to destination
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from source
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Skip the root directory itself
		if relPath == "." {
			return nil
		}

		destPath := filepath.Join(destDir, relPath)

		if info.IsDir() {
			// Create directory
			return os.MkdirAll(destPath, info.Mode())
		}

		// Copy file content
		srcData, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.WriteFile(destPath, srcData, info.Mode()); err != nil {
			return err
		}

		return nil
	})
}

// executeBuild runs makepkg to build the package
func (bm *BuildManager) executeBuild(buildDir, logPath string) (string, error) {
	// Run makepkg with:
	// -s: install missing dependencies
	// -r: remove build files after successful build
	// -C: skip integrity checks (user assumes responsibility)
	cmd := exec.Command("makepkg", "-s", "-r", "-C")
	cmd.Dir = buildDir

	// Create a buffer to capture output for logging
	var outputBuffer bytes.Buffer

	// Create a MultiWriter that writes to both console and buffer
	multiWriter := io.MultiWriter(os.Stdout, &outputBuffer)

	// Stream output to console and capture for logging
	cmd.Stdout = multiWriter
	cmd.Stderr = multiWriter

	// Run the command
	err := cmd.Run()
	buildOutput := outputBuffer.String()

	if err != nil {
		return buildOutput, err
	}

	return buildOutput, nil
}

// findBuiltArtifact searches for the built .pkg.tar.zst file in the build directory
func (bm *BuildManager) findBuiltArtifact(buildDir, pkgName string) (string, error) {
	// List files in build directory looking for .pkg.tar.zst
	entries, err := os.ReadDir(buildDir)
	if err != nil {
		return "", fmt.Errorf("failed to read build directory: %w", err)
	}

	var artifacts []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".zst" {
			if filepath.Ext(filepath.Base(entry.Name()[:len(entry.Name())-4])) == ".tar" {
				artifacts = append(artifacts, filepath.Join(buildDir, entry.Name()))
			}
		}
	}

	if len(artifacts) == 0 {
		return "", fmt.Errorf("no .pkg.tar.zst artifacts found in build directory")
	}

	// Return the first artifact (for split packages, this would be the base package)
	return artifacts[0], nil
}

// CleanupBuildArtifacts removes old build directories from the cache
// maxAge: maximum age of build directories to keep (older ones are deleted)
func (bm *BuildManager) CleanupBuildArtifacts(maxAge time.Duration) error {
	entries, err := os.ReadDir(bm.buildCacheDir)
	if err != nil {
		return fmt.Errorf("failed to read build cache directory: %w", err)
	}

	now := time.Now()
	var cleanedCount int

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Check if directory is older than maxAge
		if now.Sub(info.ModTime()) > maxAge {
			dirPath := filepath.Join(bm.buildCacheDir, entry.Name())
			if err := os.RemoveAll(dirPath); err != nil {
				// Log error but continue cleaning other directories
				fmt.Printf("Warning: failed to remove old build directory %s: %v\n", dirPath, err)
			} else {
				cleanedCount++
			}
		}
	}

	return nil
}

// CleanupBuildLogs removes old build log files
// maxAge: maximum age of log files to keep
func (bm *BuildManager) CleanupBuildLogs(maxAge time.Duration) error {
	entries, err := os.ReadDir(bm.logsDir)
	if err != nil {
		return fmt.Errorf("failed to read logs directory: %w", err)
	}

	now := time.Now()
	var cleanedCount int

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".log" {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Check if file is older than maxAge
		if now.Sub(info.ModTime()) > maxAge {
			filePath := filepath.Join(bm.logsDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				// Log error but continue cleaning other files
				fmt.Printf("Warning: failed to remove old log file %s: %v\n", filePath, err)
			} else {
				cleanedCount++
			}
		}
	}

	return nil
}

// GetBuildLog reads and returns the build log for a package
func (bm *BuildManager) GetBuildLog(pkgName, version string) (string, error) {
	logPath := filepath.Join(bm.logsDir, fmt.Sprintf("%s-%s.log", pkgName, version))

	content, err := os.ReadFile(logPath)
	if err != nil {
		return "", fmt.Errorf("failed to read build log: %w", err)
	}

	return string(content), nil
}

// GetBuildCacheSize returns the total size of the build cache in bytes
func (bm *BuildManager) GetBuildCacheSize() (int64, error) {
	var totalSize int64

	err := filepath.Walk(bm.buildCacheDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return totalSize, err
}

// VerifyMakepkgAvailable checks if makepkg command is available
func VerifyMakepkgAvailable() error {
	cmd := exec.Command("which", "makepkg")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("makepkg not found: ensure base-devel is installed")
	}
	return nil
}

// VerifyBaseDevelInstalled checks if base-devel package group is installed
// This is a convenience check to ensure build tools are available
func VerifyBaseDevelInstalled() error {
	// Check for essential build tools
	tools := []string{"gcc", "make", "tar", "gzip"}
	for _, tool := range tools {
		cmd := exec.Command("which", tool)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("essential build tool not found: %s (ensure base-devel is installed)", tool)
		}
	}
	return nil
}
