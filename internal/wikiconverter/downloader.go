package wikiconverter

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nodasoft/Mediawiki-to-MDX-docusaurus/internal/config"
)

const (
	// HTTP client timeout
	httpClientTimeout     = 30 * time.Second
	defaultImageAssetsDir = "./static/img"
	defaultFileAssetsDir  = "./static/files"
)

// Downloader handles downloading assets from MediaWiki
type Downloader struct {
	assetBaseURL   string // Base URL for assets (e.g., https://wiki.example.com/images)
	imageAssetsDir string // Directory for downloaded images
	fileAssetsDir  string // Directory for downloaded files
	verbose        bool   // Verbose output
}

// NewDownloader creates a new asset downloader
func NewDownloader(config config.Config) *Downloader {
	downloader := &Downloader{
		assetBaseURL:   config.AssetBaseURL,
		imageAssetsDir: config.ImageAssetsDir,
		fileAssetsDir:  config.FileAssetsDir,
		verbose:        config.Verbose,
	}

	if config.DownloadAssets && downloader.imageAssetsDir == "" {
		downloader.imageAssetsDir = defaultImageAssetsDir
	}

	if config.DownloadAssets && downloader.fileAssetsDir == "" {
		downloader.fileAssetsDir = defaultFileAssetsDir
	}

	return downloader
}

// downloadAssets downloads all assets (images and files) from the MediaWiki server
// Returns the number of successfully downloaded assets, failed downloads, and any error
func (d *Downloader) downloadAssets(assets map[string]struct{}) (int, int, error) {
	if d.assetBaseURL == "" {
		return 0, 0, fmt.Errorf("asset-url is required when download-assets is enabled")
	}
	if len(assets) == 0 {
		return 0, 0, nil
	}

	if err := os.MkdirAll(d.imageAssetsDir, dirPermissions); err != nil {
		return 0, 0, fmt.Errorf("failed to create image assets directory: %w", err)
	}

	if err := os.MkdirAll(d.fileAssetsDir, dirPermissions); err != nil {
		return 0, 0, fmt.Errorf("failed to create file assets directory: %w", err)
	}

	client := &http.Client{Timeout: httpClientTimeout}

	downloaded := 0
	failed := 0
	for assetName := range assets {
		saved, err := d.downloadAsset(client, assetName)
		if err != nil {
			failed++
			if d.verbose {
				log.Printf("  ✗ Failed asset %s: %v", assetName, err)
			}
			continue
		}
		if saved {
			downloaded++
		}
	}

	return downloaded, failed, nil
}

// downloadAsset downloads a single asset from MediaWiki
// Returns true if the asset was downloaded, false if it already exists or wasn't found
func (d *Downloader) downloadAsset(client *http.Client, assetName string) (bool, error) {
	targetPath := d.targetAssetPath(assetName)
	if _, err := os.Stat(targetPath); err == nil {
		if d.verbose {
			log.Printf("  - Asset exists, skip: %s", assetName)
		}
		return false, nil
	}

	candidates := buildMediaWikiImageCandidates(d.assetBaseURL, assetName)
	if len(candidates) == 0 {
		return false, fmt.Errorf("no download URL candidates for %s", assetName)
	}

	for _, candidateURL := range candidates {
		resp, err := client.Get(candidateURL)
		if err != nil {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			continue
		}

		if err := writeImageFile(targetPath, resp.Body); err != nil {
			_ = resp.Body.Close()
			return false, err
		}
		_ = resp.Body.Close()

		if d.verbose {
			log.Printf("  ✓ Downloaded asset: %s", assetName)
		}

		return true, nil
	}

	return false, fmt.Errorf("not found at expected MediaWiki paths")
}

// targetAssetPath returns the local filesystem path for an asset
func (d *Downloader) targetAssetPath(filename string) string {
	if isImageAsset(filename) {
		return filepath.Join(d.imageAssetsDir, filename)
	}
	return filepath.Join(d.fileAssetsDir, filename)
}

// buildMediaWikiImageCandidates generates possible URLs for a MediaWiki image
// MediaWiki stores images in subdirectories based on MD5 hash
func buildMediaWikiImageCandidates(baseURL, imageName string) []string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	imageName = normalizeAssetName(imageName)
	if baseURL == "" || imageName == "" {
		return nil
	}

	escaped := url.PathEscape(imageName)

	hash := md5.Sum([]byte(imageName))
	hashHex := hex.EncodeToString(hash[:])
	dir1 := hashHex[:1]
	dir2 := hashHex[:2]

	return []string{
		baseURL + "/" + dir1 + "/" + dir2 + "/" + escaped,
		baseURL + "/" + escaped,
	}
}

// writeImageFile writes an image file to disk from a reader
func writeImageFile(targetPath string, src io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), dirPermissions); err != nil {
		return fmt.Errorf("failed to create image directory: %w", err)
	}

	f, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create image file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	if _, err := io.Copy(f, src); err != nil {
		_ = os.Remove(targetPath)
		return fmt.Errorf("failed to write image content: %w", err)
	}

	return nil
}
