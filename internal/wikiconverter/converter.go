package wikiconverter

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

const (
	// File permissions
	dirPermissions  = 0755
	filePermissions = 0644

	// HTTP client timeout
	httpClientTimeout = 30 * time.Second

	// Description length limit
	maxDescriptionLength = 160
)

// Config holds the converter configuration
type Config struct {
	DBHost         string
	DBPort         string
	DBUser         string
	DBPass         string
	DBName         string
	TablePrefix    string
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

// Converter handles the conversion process
type Converter struct {
	reader     *WikiReader
	config     Config
	parser     *WikiParser
	formatter  *DocusaurusFormatter
	downloader *Downloader
}

// Stats holds conversion statistics
type Stats struct {
	TotalArticles        int
	Converted            int
	Skipped              int
	Failed               int
	ImagesDownloaded     int
	ImagesDownloadFailed int
}

// NewConverter creates a new converter instance
func NewConverter(config Config) (*Converter, error) {
	reader, err := NewWikiReader(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create WikiReader: %w", err)
	}

	return &Converter{
		reader:     reader,
		config:     config,
		parser:     NewWikiParserWithImageURL(config.ImageAssetsURL, config.FileAssetsURL),
		formatter:  NewDocusaurusFormatter(),
		downloader: NewDownloader(config),
	}, nil
}

// Close closes the converter and releases resources
func (c *Converter) Close() error {
	if c.reader != nil {
		return c.reader.Close()
	}
	return nil
}

// Convert performs the conversion
func (c *Converter) Convert() (*Stats, error) {
	stats := &Stats{}

	// Create output directory
	if err := os.MkdirAll(c.config.OutputDir, dirPermissions); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Fetch pages from MediaWiki
	pages, err := c.reader.FetchPages()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pages: %w", err)
	}

	stats.TotalArticles = len(pages)
	if c.config.Verbose {
		log.Printf("Found %d pages to convert", stats.TotalArticles)
	}

	// Convert each page
	for i, page := range pages {
		if c.config.Verbose {
			log.Printf("[%d/%d] Processing: %s", i+1, stats.TotalArticles, page.Title)
		}

		// Skip redirects
		if page.IsRedirect {
			if c.config.Verbose {
				log.Printf("  Skipping redirect: %s", page.Title)
			}
			stats.Skipped++
			continue
		}

		// Convert page
		mdxContent := c.convertPage(page)

		// Save page
		if err = c.savePage(page, mdxContent); err != nil {
			log.Printf("  ERROR saving %s: %v", page.Title, err)
			stats.Failed++
			continue
		}

		stats.Converted++
	}

	if c.config.DownloadAssets {
		downloaded, failed, err := c.downloader.downloadAssets(c.parser.assetsUnique())
		if err != nil {
			log.Printf("  ERROR downloading assets: %v", err)
		}
		stats.ImagesDownloaded = downloaded
		stats.ImagesDownloadFailed = failed
	}

	return stats, nil
}

// convertPage converts a single page to Docusaurus format
func (c *Converter) convertPage(page WikiPage) string {
	// Parse wiki markup to markdown
	markdown := c.parser.Parse(page.Content)

	// Create Docusaurus document
	doc := DocusaurusDoc{
		Title:       page.Title,
		ID:          c.generateID(page.Title),
		Description: c.extractDescription(markdown),
		Content:     markdown,
		Sidebar:     c.generateSidebarPosition(page.ID),
	}

	// Format as Docusaurus MDX
	mdxContent := c.formatter.Format(doc)

	return mdxContent
}

func (c *Converter) savePage(page WikiPage, mdxContent string) error {
	// Generate filename
	filename := c.generateFilename(page.Title, page.Namespace)
	fullPath := filepath.Join(c.config.OutputDir, filename)

	// Create subdirectories if needed
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(mdxContent), filePermissions); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	if c.config.Verbose {
		log.Printf("  ✓ Saved to: %s", filename)
	}

	return nil
}

// cyrillicToLatin maps Cyrillic characters to Latin equivalents
var cyrillicToLatin = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'д': "d", 'е': "e", 'ё': "yo", 'ж': "zh",
	'з': "z", 'и': "i", 'й': "y", 'к': "k", 'л': "l", 'м': "m", 'н': "n", 'о': "o",
	'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u", 'ф': "f", 'х': "h", 'ц': "ts",
	'ч': "ch", 'ш': "sh", 'щ': "sch", 'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu", 'я': "ya",
	'А': "a", 'Б': "b", 'В': "v", 'Г': "g", 'Д': "d", 'Е': "e", 'Ё': "yo", 'Ж': "zh",
	'З': "z", 'И': "i", 'Й': "y", 'К': "k", 'Л': "l", 'М': "m", 'Н': "n", 'О': "o",
	'П': "p", 'Р': "r", 'С': "s", 'Т': "t", 'У': "u", 'Ф': "f", 'Х': "h", 'Ц': "ts",
	'Ч': "ch", 'Ш': "sh", 'Щ': "sch", 'Ъ': "", 'Ы': "y", 'Ь': "", 'Э': "e", 'Ю': "yu", 'Я': "ya",
}

// transliterateCyrillic converts Cyrillic characters to Latin
func transliterateCyrillic(s string) string {
	var result strings.Builder
	for _, r := range s {
		if latin, ok := cyrillicToLatin[r]; ok {
			result.WriteString(latin)
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// generateID creates a valid Docusaurus ID from a title by converting to lowercase,
// transliterating Cyrillic characters, and removing special characters
func (c *Converter) generateID(title string) string {
	id := strings.ToLower(title)
	// Transliterate Cyrillic to Latin
	id = transliterateCyrillic(id)
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, "/", "-")
	// Remove special characters
	id = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, id)

	return id
}

// generateFilename creates a filename from a title and namespace,
// organizing files into subdirectories based on namespace
func (c *Converter) generateFilename(title string, namespace int) string {
	// Create subdirectory based on namespace
	var subdir string
	switch namespace {
	case 0:
		subdir = "" // Main namespace
	case 1:
		subdir = "talk"
	case 2:
		subdir = "user"
	case 4:
		subdir = "project"
	case 6:
		subdir = "file"
	case 8:
		subdir = "mediawiki"
	case 10:
		subdir = "template"
	case 12:
		subdir = "help"
	case 14:
		subdir = "category"
	default:
		subdir = fmt.Sprintf("ns-%d", namespace)
	}

	filename := c.generateID(title) + ".md"

	if subdir != "" {
		return filepath.Join(subdir, filename)
	}
	return filename
}

// generateSidebarPosition generates sidebar position based on page ID
func (c *Converter) generateSidebarPosition(pageID int) string {
	return fmt.Sprintf("%d", pageID)
}

// extractDescription extracts the first non-empty paragraph as a description,
// limited to maxDescriptionLength characters
func (c *Converter) extractDescription(markdown string) string {
	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		line = stripHTMLTags(line)
		line = strings.ReplaceAll(line, "<=", "≤")
		line = strings.ReplaceAll(line, ">=", "≥")
		line = strings.ReplaceAll(line, "{", "")
		line = strings.ReplaceAll(line, "}", "")
		line = strings.ReplaceAll(line, "|", "")
		line = strings.ReplaceAll(line, "*", "")
		line = strings.ReplaceAll(line, "`", "")
		line = strings.TrimSpace(line)
		line = strings.Trim(line, "-.,!' \"")
		line = strings.Trim(line, " ")
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "---") {
			// Limit to maxDescriptionLength characters
			if len(line) > maxDescriptionLength {
				return line[:maxDescriptionLength-3] + "..."
			}
			return line
		}
	}
	return ""
}

// stripHTMLTags removes HTML tags from a string
func stripHTMLTags(s string) string {
	return regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, "")
}
