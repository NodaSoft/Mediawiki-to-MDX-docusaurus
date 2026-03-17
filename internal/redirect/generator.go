package redirect

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/nodasoft/Mediawiki-to-MDX-docusaurus/internal/wikiconverter"
	"github.com/nodasoft/Mediawiki-to-MDX-docusaurus/internal/wikireader"
	"gopkg.in/yaml.v3"
)

// Map represents the redirect mapping structure
type Map struct {
	// Map from old MediaWiki URL path to new Docusaurus path
	Redirects map[string]string `yaml:"redirects"`
	// Map for MediaWiki internal redirects (page -> page)
	WikiRedirects map[string]string `yaml:"wiki_redirects"`
}

type Config struct {
	wikireader.DBConfig
	Namespace      string
	Verbose        bool
	OutputFile     string // Output YAML file for redirect map
	PageBaseURL    string // Base URL for pages (e.g., "https://docs.example.com/docs/")
	ImageAssetsURL string // Base URL for images (e.g., "https://docs.example.com/static/images/")
	FileAssetsURL  string // Base URL for files (e.g., "https://docs.example.com/static/files/")
}

type Generator struct {
	config Config
}

func NewGenerator(cfg Config) *Generator {
	return &Generator{
		config: cfg,
	}
}

func (g *Generator) Run() error {
	fmt.Println("Generating redirect map from MediaWiki database...")

	// Create wiki reader
	reader, err := wikireader.NewWikiDBReader(g.config.DBConfig, g.config.Namespace)
	if err != nil {
		return fmt.Errorf("failed to create wiki reader: %w", err)
	}
	defer reader.Close()

	// Fetch all pages
	pages, err := reader.FetchPages()
	if err != nil {
		return fmt.Errorf("failed to fetch pages: %w", err)
	}

	if g.config.Verbose {
		log.Printf("Found %d pages in database", len(pages))
	}

	// Build redirect map
	redirectMap := Map{
		Redirects:     make(map[string]string),
		WikiRedirects: make(map[string]string),
	}

	// Process each page
	for _, page := range pages {
		if !strings.Contains(page.Title, "Самостоятельное") {
			//continue
		}
		// Convert page title to old MediaWiki URL format
		oldURL := convertToMediaWikiURL(page.Title, page.Namespace)
		oldRUURL := convertToMediaWikiRUURL(page.Title, page.Namespace)
		// Handle MediaWiki redirects
		if page.IsRedirect {
			redirectTarget := wikiconverter.ExtractRedirectTarget(page.Content)
			if redirectTarget != "" {
				redirectTarget = strings.ReplaceAll(redirectTarget, " ", "_")
				redirectMap.WikiRedirects[oldURL] = redirectTarget
				redirectMap.WikiRedirects[oldRUURL] = redirectTarget
				if g.config.Verbose {
					log.Printf("  Wiki redirect: %s -> %s", oldURL, redirectTarget)
				}
			}
		} else {
			// Convert page title to new Docusaurus URL format
			newURL := g.convertToDocusaurusURL(page.Title, page.Namespace)

			// Add to redirect map
			redirectMap.Redirects[oldURL] = newURL
			redirectMap.Redirects[oldRUURL] = newURL

			if g.config.Verbose {
				log.Printf("  Redirect: %s -> %s", oldURL, newURL)
				log.Printf("  Redirect: %s -> %s", oldRUURL, newURL)
			}
		}
	}

	// Save to YAML file
	if err := saveRedirectMap(g.config.OutputFile, redirectMap); err != nil {
		return fmt.Errorf("failed to save redirect map: %w", err)
	}

	fmt.Printf("\n=== Generation Complete ===\n")
	fmt.Printf("Total redirects: %d\n", len(redirectMap.Redirects))
	fmt.Printf("Wiki redirects: %d\n", len(redirectMap.WikiRedirects))
	fmt.Printf("Output file: %s\n", g.config.OutputFile)

	return nil
}

// convertToDocusaurusURL converts a page title to Docusaurus URL format
func (g *Generator) convertToDocusaurusURL(title string, namespace int) string {
	// Convert title to ID format using the converter's ConvertInternalLink function
	return wikiconverter.ConvertInternalLink(title, g.config.PageBaseURL, g.config.ImageAssetsURL, g.config.FileAssetsURL, namespace)
}

// convertToMediaWikiURL converts a page title to MediaWiki URL format
func convertToMediaWikiURL(title string, namespace int) string {
	// Replace spaces with underscores
	urlTitle := strings.ReplaceAll(title, " ", "_")
	urlTitle = strings.ReplaceAll(urlTitle, "\u200e", "")
	urlTitle = strings.TrimSpace(urlTitle)

	// Add namespace prefix if needed
	var prefix string
	switch namespace {
	case 0:
		prefix = ""
	case 1:
		prefix = "Talk:"
	case 2:
		prefix = "User:"
	case 4:
		prefix = "Project:"
	case 6:
		prefix = "File:"
	case 8:
		prefix = "MediaWiki:"
	case 10:
		prefix = "Template:"
	case 12:
		prefix = "Help:"
	case 14:
		prefix = "Category:"
	default:
		prefix = fmt.Sprintf("NS%d:", namespace)
	}

	return prefix + urlTitle
}

// convertToMediaWikiRUURL converts a page title to MediaWiki URL format for Russian wiki
func convertToMediaWikiRUURL(title string, namespace int) string {
	// Replace spaces with underscores
	urlTitle := strings.ReplaceAll(title, " ", "_")
	urlTitle = strings.ReplaceAll(urlTitle, "\u200e", "")
	urlTitle = strings.TrimSpace(urlTitle)

	// Add namespace prefix if needed
	var prefix string
	switch namespace {
	case 0:
		prefix = ""
	case 1:
		prefix = "Разговор:"
	case 2:
		prefix = "Пользователь:"
	case 4:
		prefix = "Проект:"
	case 6:
		prefix = "Файл:"
	case 8:
		prefix = "MediaWiki:"
	case 10:
		prefix = "Шаблон:"
	case 12:
		prefix = "Помощь:"
	case 14:
		prefix = "Категория:"
	default:
		prefix = fmt.Sprintf("NS%d:", namespace)
	}

	return prefix + urlTitle
}

// saveRedirectMap saves the redirect map to a YAML file
func saveRedirectMap(filename string, redirectMap Map) error {
	// Create directory if needed
	dir := filepath.Dir(filename)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Marshal to YAML
	data, err := yaml.Marshal(redirectMap)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
