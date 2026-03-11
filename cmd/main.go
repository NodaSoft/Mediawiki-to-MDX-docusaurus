package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/nodasoft/Mediawiki-to-MDX-docusaurus/internal/wikiconverter"
)

func envOrDefault(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}

func envBoolOrDefault(envKey string, fallback bool) bool {
	v := os.Getenv(envKey)
	if v == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}

	return parsed
}

func main() {
	// Command line flags with environment variable defaults.
	dbHost := flag.String("db-host", envOrDefault("WIKI_DB_HOST", "localhost"), "MediaWiki database host (env: WIKI_DB_HOST)")
	dbPort := flag.String("db-port", envOrDefault("WIKI_DB_PORT", "3306"), "MediaWiki database port (env: WIKI_DB_PORT)")
	dbUser := flag.String("db-user", envOrDefault("WIKI_DB_USER", "root"), "MediaWiki database user (env: WIKI_DB_USER)")
	dbPass := flag.String("db-pass", envOrDefault("WIKI_DB_PASS", ""), "MediaWiki database password (env: WIKI_DB_PASS)")
	dbName := flag.String("db-name", envOrDefault("WIKI_DB_NAME", "mediawiki"), "MediaWiki database name (env: WIKI_DB_NAME)")
	tablePrefix := flag.String("table-prefix", envOrDefault("WIKI_TABLE_PREFIX", ""), "MediaWiki table prefix (e.g., mw_) (env: WIKI_TABLE_PREFIX)")
	outputDir := flag.String("output", envOrDefault("OUTPUT_DIR", "./docs"), "Output directory for Docusaurus docs (env: OUTPUT_DIR)")
	imagesDir := flag.String("images-dir", envOrDefault("IMAGES_DIR", "./static/img/wiki"), "Directory for downloaded images (env: IMAGES_DIR)")
	filesDir := flag.String("files-dir", envOrDefault("FILES_DIR", "./static/files/wiki"), "Directory for downloaded files (env: FILES_DIR)")
	namespace := flag.String("namespace", envOrDefault("NAMESPACE", ""), "Filter by namespace (0=main, 1=talk, etc). Empty for all (env: NAMESPACE)")
	assetBaseURL := flag.String("asset-url", envOrDefault("ASSET_BASE_URL", ""), "Base URL for assets (e.g., https://wiki.example.com/images). If empty, uses /img/ (env: ASSET_BASE_URL)")
	downloadAssets := flag.Bool("download-assets", envBoolOrDefault("DOWNLOAD_ASSETS", false), "Download wiki assets (images and File: attachments). Assets are stored in images-dir and files-dir (requires asset-url) (env: DOWNLOAD_ASSETS=true/false)")
	imageAssetsURL := flag.String("image-url", envOrDefault("IMAGE_BASE_URL", ""), "Base URL for images (e.g., https://wiki.example.com/images). If empty, uses /img/ (env: IMAGE_BASE_URL)")
	fileAssetsURL := flag.String("file-url", envOrDefault("FILE_BASE_URL", ""), "Base URL for files (e.g., https://wiki.example.com/files). If empty, uses /files/ (env: FILE_BASE_URL)")
	verbose := flag.Bool("verbose", envBoolOrDefault("VERBOSE", false), "Verbose output (env: VERBOSE=true/false)")
	flag.Parse()

	if *dbPass == "" {
		log.Fatal("Database password is required (use -db-pass or WIKI_DB_PASS)")
	}

	// Create converter configuration
	config := wikiconverter.Config{
		DBHost:         *dbHost,
		DBPort:         *dbPort,
		DBUser:         *dbUser,
		DBPass:         *dbPass,
		DBName:         *dbName,
		TablePrefix:    *tablePrefix,
		OutputDir:      *outputDir,
		ImageAssetsDir: *imagesDir,
		FileAssetsDir:  *filesDir,
		Namespace:      *namespace,
		AssetBaseURL:   *assetBaseURL,
		DownloadAssets: *downloadAssets,
		Verbose:        *verbose,
		ImageAssetsURL: *imageAssetsURL,
		FileAssetsURL:  *fileAssetsURL,
	}

	// Create converter
	converter, err := wikiconverter.NewConverter(config)
	if err != nil {
		log.Fatalf("Failed to create converter: %v", err)
	}
	defer func() {
		if err := converter.Close(); err != nil {
			log.Printf("Failed to close converter: %v", err)
		}
	}()

	// Run conversion
	fmt.Println("Starting MediaWiki to Docusaurus conversion...")
	stats, err := converter.Convert()
	if err != nil {
		log.Fatalf("Conversion failed: %v", err)
	}

	// Print statistics
	fmt.Println("\n=== Conversion Complete ===")
	fmt.Printf("Total articles processed: %d\n", stats.TotalArticles)
	fmt.Printf("Successfully converted: %d\n", stats.Converted)
	fmt.Printf("Skipped: %d\n", stats.Skipped)
	fmt.Printf("Failed: %d\n", stats.Failed)
	if *downloadAssets {
		fmt.Printf("Assets downloaded: %d\n", stats.ImagesDownloaded)
		fmt.Printf("Asset download failed: %d\n", stats.ImagesDownloadFailed)
		fmt.Printf("Images directory: %s\n", *imagesDir)
		fmt.Printf("Files directory: %s\n", *filesDir)
	}
	fmt.Printf("Output directory: %s\n", *outputDir)

	if *assetBaseURL != "" {
		fmt.Printf("Image base URL: %s\n", *assetBaseURL)
	} else {
		fmt.Println("Images will use relative path: /img/")
	}

	if stats.Failed > 0 {
		os.Exit(1)
	}
}
