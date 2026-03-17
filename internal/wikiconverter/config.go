package wikiconverter

import "github.com/nodasoft/Mediawiki-to-MDX-docusaurus/internal/wikireader"

// Config holds the converter configuration
type Config struct {
	wikireader.DBConfig
	OutputDir      string // Output directory for generated Docusaurus mdx-files
	Namespace      string
	Verbose        bool
	DownloadAssets bool   // Whether to download assets locally
	AssetBaseURL   string // Base URL for assets (e.g., "https://wiki.example.com/images") for download them from a mediawiki source website
	ImageAssetsDir string // Subdirectory path for locally stored images (e.g., "./static/img")
	FileAssetsDir  string // Subdirectory path for locally stored files (e.g., "./static/files")
	ImageAssetsURL string // Base URL for locally stored images (e.g., "https://example.com/static/img")
	FileAssetsURL  string // Base URL for locally stored files (e.g., "https://example.com/static/files")
}
