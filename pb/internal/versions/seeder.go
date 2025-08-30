package versions

import (
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// AppVersionConfig represents an app version configuration for seeding
type AppVersionConfig struct {
	Version           string
	Platform          string
	Architecture      string
	DownloadURL       string
	FileSize          int64
	ChecksumSHA256    string
	ReleaseNotes      string
	IsLatest          bool
	IsReleased        bool
	IsPrerelease      bool
	MinimumOSVersion  string
	DownloadCount     int
	CreatedOffset     time.Duration // Offset from now for created timestamp
}

// generateChecksum creates a realistic-looking SHA256 checksum for testing
func generateChecksum(platform, version, arch string) string {
	input := fmt.Sprintf("%s-%s-%s-ramble", platform, version, arch)
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash)
}

// SeedAppVersions creates sample app versions for development testing
// This function should ONLY be called in development environments
func SeedAppVersions(app core.App) error {
	// Safety check - only run in development
	if os.Getenv("DEVELOPMENT") != "true" {
		log.Println("Skipping app versions seeding - not in development mode")
		return nil
	}

	log.Println("🌱 Seeding app versions...")

	// Check if versions already exist
	existingVersions, err := app.FindRecordsByFilter("app_versions", "", "", 1, 0)
	if err == nil && len(existingVersions) > 0 {
		log.Println("App versions already exist, skipping seeding")
		return nil
	}

	// Define version history with multiple versions per platform
	versions := []AppVersionConfig{
		// Windows versions
		{
			Version:          "1.0.0",
			Platform:         "windows",
			Architecture:     "amd64",
			DownloadURL:      "https://ramble-releases.s3.amazonaws.com/v1.0.0/ramble-setup-1.0.0-windows-amd64.exe",
			FileSize:         26214400, // 25MB
			ChecksumSHA256:   generateChecksum("windows", "1.0.0", "amd64"),
			ReleaseNotes:     "# Ramble v1.0.0\n\n## Features\n- Initial release\n- Basic script optimization\n- Windows support\n\n## System Requirements\n- Windows 10 or later",
			IsLatest:         false,
			IsReleased:       true,
			IsPrerelease:     false,
			MinimumOSVersion: "Windows 10",
			DownloadCount:    450,
			CreatedOffset:    -90 * 24 * time.Hour, // 90 days ago
		},
		{
			Version:          "1.1.0",
			Platform:         "windows",
			Architecture:     "amd64",
			DownloadURL:      "https://ramble-releases.s3.amazonaws.com/v1.1.0/ramble-setup-1.1.0-windows-amd64.exe",
			FileSize:         28311552, // 27MB
			ChecksumSHA256:   generateChecksum("windows", "1.1.0", "amd64"),
			ReleaseNotes:     "# Ramble v1.1.0\n\n## New Features\n- Improved AI processing speed\n- Better video quality optimization\n- Enhanced Windows compatibility\n\n## Bug Fixes\n- Fixed audio sync issues\n- Improved memory usage",
			IsLatest:         false,
			IsReleased:       true,
			IsPrerelease:     false,
			MinimumOSVersion: "Windows 10",
			DownloadCount:    320,
			CreatedOffset:    -30 * 24 * time.Hour, // 30 days ago
		},
		{
			Version:          "1.2.0",
			Platform:         "windows",
			Architecture:     "amd64",
			DownloadURL:      "https://ramble-releases.s3.amazonaws.com/v1.2.0/ramble-setup-1.2.0-windows-amd64.exe",
			FileSize:         31457280, // 30MB
			ChecksumSHA256:   generateChecksum("windows", "1.2.0", "amd64"),
			ReleaseNotes:     "# Ramble v1.2.0\n\n## Major Features\n- New script editor with live preview\n- Batch processing capabilities\n- Windows 11 optimizations\n\n## Improvements\n- 40% faster processing\n- Better GPU acceleration\n- Enhanced UI/UX",
			IsLatest:         true,
			IsReleased:       true,
			IsPrerelease:     false,
			MinimumOSVersion: "Windows 10",
			DownloadCount:    125,
			CreatedOffset:    -3 * 24 * time.Hour, // 3 days ago
		},

		// macOS versions
		{
			Version:          "1.0.0",
			Platform:         "macos",
			Architecture:     "universal",
			DownloadURL:      "https://ramble-releases.s3.amazonaws.com/v1.0.0/ramble-1.0.0-macos-universal.dmg",
			FileSize:         29360128, // 28MB
			ChecksumSHA256:   generateChecksum("macos", "1.0.0", "universal"),
			ReleaseNotes:     "# Ramble v1.0.0\n\n## Features\n- Initial macOS release\n- Universal Binary (Intel + Apple Silicon)\n- Native macOS integration\n\n## System Requirements\n- macOS 11 Big Sur or later",
			IsLatest:         false,
			IsReleased:       true,
			IsPrerelease:     false,
			MinimumOSVersion: "macOS 11.0",
			DownloadCount:    380,
			CreatedOffset:    -90 * 24 * time.Hour, // 90 days ago
		},
		{
			Version:          "1.1.0",
			Platform:         "macos",
			Architecture:     "universal",
			DownloadURL:      "https://ramble-releases.s3.amazonaws.com/v1.1.0/ramble-1.1.0-macos-universal.dmg",
			FileSize:         30408704, // 29MB
			ChecksumSHA256:   generateChecksum("macos", "1.1.0", "universal"),
			ReleaseNotes:     "# Ramble v1.1.0\n\n## New Features\n- Apple Silicon optimizations\n- Improved macOS Monterey support\n- Better file system integration\n\n## Performance\n- 2x faster on M1/M2 chips\n- Reduced memory footprint",
			IsLatest:         false,
			IsReleased:       true,
			IsPrerelease:     false,
			MinimumOSVersion: "macOS 11.0",
			DownloadCount:    290,
			CreatedOffset:    -30 * 24 * time.Hour, // 30 days ago
		},
		{
			Version:          "1.2.0",
			Platform:         "macos",
			Architecture:     "universal",
			DownloadURL:      "https://ramble-releases.s3.amazonaws.com/v1.2.0/ramble-1.2.0-macos-universal.dmg",
			FileSize:         33554432, // 32MB
			ChecksumSHA256:   generateChecksum("macos", "1.2.0", "universal"),
			ReleaseNotes:     "# Ramble v1.2.0\n\n## Major Features\n- macOS Sonoma full support\n- New batch processing engine\n- Enhanced Metal GPU acceleration\n\n## macOS Specific\n- App Store distribution ready\n- Notarized for security\n- Native macOS UI improvements",
			IsLatest:         true,
			IsReleased:       true,
			IsPrerelease:     false,
			MinimumOSVersion: "macOS 11.0",
			DownloadCount:    95,
			CreatedOffset:    -3 * 24 * time.Hour, // 3 days ago
		},

		// Linux versions (multiple architectures for latest)
		{
			Version:          "1.0.0",
			Platform:         "linux",
			Architecture:     "amd64",
			DownloadURL:      "https://ramble-releases.s3.amazonaws.com/v1.0.0/ramble-1.0.0-linux-amd64.AppImage",
			FileSize:         27262976, // 26MB
			ChecksumSHA256:   generateChecksum("linux", "1.0.0", "amd64"),
			ReleaseNotes:     "# Ramble v1.0.0\n\n## Features\n- Initial Linux release\n- AppImage distribution\n- X11 and Wayland support\n\n## System Requirements\n- Ubuntu 20.04+ or equivalent\n- 4GB RAM minimum",
			IsLatest:         false,
			IsReleased:       true,
			IsPrerelease:     false,
			MinimumOSVersion: "Ubuntu 20.04",
			DownloadCount:    215,
			CreatedOffset:    -90 * 24 * time.Hour, // 90 days ago
		},
		{
			Version:          "1.1.0",
			Platform:         "linux",
			Architecture:     "amd64",
			DownloadURL:      "https://ramble-releases.s3.amazonaws.com/v1.1.0/ramble-1.1.0-linux-amd64.AppImage",
			FileSize:         28311552, // 27MB
			ChecksumSHA256:   generateChecksum("linux", "1.1.0", "amd64"),
			ReleaseNotes:     "# Ramble v1.1.0\n\n## Linux Improvements\n- Better Wayland compatibility\n- Improved AppImage integration\n- Hardware acceleration support\n\n## Performance\n- Optimized for AMD and Intel CPUs\n- Better GPU detection",
			IsLatest:         false,
			IsReleased:       true,
			IsPrerelease:     false,
			MinimumOSVersion: "Ubuntu 20.04",
			DownloadCount:    180,
			CreatedOffset:    -30 * 24 * time.Hour, // 30 days ago
		},
		{
			Version:          "1.1.0",
			Platform:         "linux",
			Architecture:     "arm64",
			DownloadURL:      "https://ramble-releases.s3.amazonaws.com/v1.1.0/ramble-1.1.0-linux-arm64.AppImage",
			FileSize:         26214400, // 25MB
			ChecksumSHA256:   generateChecksum("linux", "1.1.0", "arm64"),
			ReleaseNotes:     "# Ramble v1.1.0\n\n## ARM64 Release\n- Native ARM64 support\n- Raspberry Pi compatible\n- Optimized for ARM processors\n\n## Features\n- Same functionality as x64 version\n- Lower power consumption",
			IsLatest:         false,
			IsReleased:       true,
			IsPrerelease:     false,
			MinimumOSVersion: "Ubuntu 20.04",
			DownloadCount:    45,
			CreatedOffset:    -30 * 24 * time.Hour, // 30 days ago
		},
		{
			Version:          "1.2.0",
			Platform:         "linux",
			Architecture:     "amd64",
			DownloadURL:      "https://ramble-releases.s3.amazonaws.com/v1.2.0/ramble-1.2.0-linux-amd64.AppImage",
			FileSize:         32505856, // 31MB
			ChecksumSHA256:   generateChecksum("linux", "1.2.0", "amd64"),
			ReleaseNotes:     "# Ramble v1.2.0\n\n## Major Linux Updates\n- Native Flatpak support\n- Enhanced Wayland integration\n- Better desktop environment compatibility\n\n## Performance\n- CUDA and OpenCL acceleration\n- Improved multi-threading\n- 50% faster on Linux",
			IsLatest:         true,
			IsReleased:       true,
			IsPrerelease:     false,
			MinimumOSVersion: "Ubuntu 20.04",
			DownloadCount:    72,
			CreatedOffset:    -3 * 24 * time.Hour, // 3 days ago
		},
		{
			Version:          "1.2.0",
			Platform:         "linux",
			Architecture:     "arm64",
			DownloadURL:      "https://ramble-releases.s3.amazonaws.com/v1.2.0/ramble-1.2.0-linux-arm64.AppImage",
			FileSize:         30408704, // 29MB
			ChecksumSHA256:   generateChecksum("linux", "1.2.0", "arm64"),
			ReleaseNotes:     "# Ramble v1.2.0\n\n## ARM64 Enhancements\n- Optimized for Apple Silicon Linux VMs\n- Raspberry Pi 4/5 support\n- Better ARM GPU acceleration\n\n## Features\n- Full feature parity with x64\n- Native ARM optimizations\n- Lower memory usage",
			IsLatest:         true,
			IsReleased:       true,
			IsPrerelease:     false,
			MinimumOSVersion: "Ubuntu 20.04",
			DownloadCount:    28,
			CreatedOffset:    -3 * 24 * time.Hour, // 3 days ago
		},
	}

	// Get the app_versions collection
	collection, err := app.FindCollectionByNameOrId("app_versions")
	if err != nil {
		return fmt.Errorf("failed to find app_versions collection: %w", err)
	}

	// Create each version record
	now := time.Now()
	for _, version := range versions {
		record := core.NewRecord(collection)
		
		// Set the data
		record.Set("version", version.Version)
		record.Set("platform", version.Platform)
		record.Set("architecture", version.Architecture)
		record.Set("download_url", version.DownloadURL)
		record.Set("file_size", version.FileSize)
		record.Set("checksum_sha256", version.ChecksumSHA256)
		record.Set("release_notes", version.ReleaseNotes)
		
		// Explicitly cast booleans to ensure they're properly set
		record.Set("is_latest", bool(version.IsLatest))
		record.Set("is_released", bool(version.IsReleased))
		record.Set("is_prerelease", bool(version.IsPrerelease))
		record.Set("minimum_os_version", version.MinimumOSVersion)
		record.Set("download_count", version.DownloadCount)
		
		// Set created timestamp with offset to simulate version history
		createdTime := now.Add(version.CreatedOffset)
		record.Set("created", createdTime)
		record.Set("updated", createdTime)

		// Save the record
		if err := app.Save(record); err != nil {
			log.Printf("Failed to create version %s for %s: %v", version.Version, version.Platform, err)
			continue
		}

		log.Printf("✓ Created version %s for %s (%s) - Latest: %v", 
			version.Version, version.Platform, version.Architecture, version.IsLatest)
	}

	log.Printf("🎉 Successfully seeded %d app versions", len(versions))
	log.Println("📱 Latest versions that should appear in UI:")
	log.Println("   - Windows 1.2.0 (amd64)")
	log.Println("   - macOS 1.2.0 (universal)")
	log.Println("   - Linux 1.2.0 (amd64, arm64)")
	
	return nil
}