package wikiconverter

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nodasoft/Mediawiki-to-MDX-docusaurus/internal/wikireader"
)

const (
	// File permissions
	dirPermissions  = 0755
	filePermissions = 0644

	// Description length limit
	maxDescriptionLength = 160
)

// Converter handles the conversion process
type Converter struct {
	reader     wikireader.WikiReader
	config     Config
	parser     *WikiParser
	formatter  *DocusaurusFormatter
	downloader *Downloader
}

// Redirect represents a page redirect
type Redirect struct {
	From string
	To   string
}

// Stats holds conversion statistics
type Stats struct {
	TotalArticles        int
	Converted            int
	Skipped              int
	Failed               int
	ImagesDownloaded     int
	ImagesDownloadFailed int
	Redirects            []Redirect
}

// NewConverter creates a new converter instance
func NewConverter(cfg Config) (*Converter, error) {
	dbConfig := cfg.DBConfig
	reader, err := wikireader.NewWikiDBReader(dbConfig, cfg.Namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to create WikiReader: %w", err)
	}

	return &Converter{
		reader:     reader,
		config:     cfg,
		parser:     NewWikiParserWithImageURL(cfg.ImageAssetsURL, cfg.FileAssetsURL),
		formatter:  NewDocusaurusFormatter(),
		downloader: NewDownloader(cfg),
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

	stats.Redirects = c.collectRedirects(pages)
	c.parser.SetRedirects(stats.Redirects)

	// Convert each page
	for i, page := range pages {
		//if strings.HasSuffix(page.Title, "EN") || strings.HasSuffix(page.Title, "en") {
		//	continue
		//}
		if c.config.Verbose {
			log.Printf("[%d/%d] Processing: %s", i+1, stats.TotalArticles, page.Title)
		}

		// Handle redirects
		if page.IsRedirect {
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

// collectRedirects collects page redirects from a list of WikiPages
func (c *Converter) collectRedirects(pages []wikireader.WikiPage) []Redirect {
	var redirects []Redirect
	for _, page := range pages {
		if page.IsRedirect {
			redirectTarget := ExtractRedirectTarget(page.Content)
			if redirectTarget != "" {
				redirects = append(redirects, Redirect{
					From: page.Title,
					To:   redirectTarget,
				})
				if c.config.Verbose {
					log.Printf("  Redirect: %s -> %s", page.Title, redirectTarget)
				}
			} else {
				if c.config.Verbose {
					log.Printf("  Skipping redirect (no target found): %s", page.Title)
				}
			}
		}
	}

	return redirects
}

// convertPage converts a single page to Docusaurus format
func (c *Converter) convertPage(page wikireader.WikiPage) string {
	// Parse wiki markup to markdown
	markdown := c.parser.Parse(page.Content)

	// Create Docusaurus document
	doc := DocusaurusDoc{
		Title:       page.Title,
		ID:          generateIDByTitle(page.Title),
		Description: c.extractDescription(markdown),
		Content:     markdown,
		Sidebar:     c.generateSidebarPosition(page.ID),
	}

	// Format as Docusaurus MDX
	mdxContent := c.formatter.Format(doc)

	return mdxContent
}

func (c *Converter) savePage(page wikireader.WikiPage, mdxContent string) error {
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

// generateFilename creates a filename from a title and namespace,
// organizing files into subdirectories based on namespace
func (c *Converter) generateFilename(title string, namespace int) string {
	filename, subdir := GeneratePageFilepath(title, namespace)

	if subdir != "" {
		// Create subdirectory if it doesn't exist
		subdirPath := filepath.Join(c.config.OutputDir, subdir)
		if err := os.MkdirAll(subdirPath, dirPermissions); err != nil {
			log.Printf("Warning: failed to create subdirectory %s: %v", subdirPath, err)
		}
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
	inAdmonition := false
	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Skip admonition blocks
		if strings.HasPrefix(trimmedLine, ":::") {
			if inAdmonition {
				inAdmonition = false
				continue
			} else {
				inAdmonition = true
				continue
			}
		}
		if inAdmonition {
			continue
		}

		// Remove markdown images: ![alt text](url)
		line = regexp.MustCompile(`!\[([^\]]*)\]\((?:[^()]|\([^)]*\))*\)`).ReplaceAllString(line, "")
		// Remove markdown links: [text](url)
		line = regexp.MustCompile(`\[([^\]]*)\]\([^\)]+\)`).ReplaceAllString(line, "$1")
		// Preserve comparison operators before removing HTML-like tags
		line = strings.ReplaceAll(line, "<=", "≤")
		line = strings.ReplaceAll(line, ">=", "≥")
		// Remove HTML tags and HTML-like tags
		line = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(line, "")
		line = strings.ReplaceAll(line, "{", " ")
		line = strings.ReplaceAll(line, "}", " ")
		line = strings.ReplaceAll(line, "|", " ")
		line = strings.ReplaceAll(line, "***", "")
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "*", " ")
		line = strings.ReplaceAll(line, "`", " ")
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
