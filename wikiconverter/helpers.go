package wikiconverter

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

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

// prepareID creates a valid Docusaurus ID from a title by converting to lowercase,
// transliterating Cyrillic characters, and removing special characters
func prepareID(title string) string {
	id := strings.ToLower(title)
	id = strings.TrimSpace(id)
	// Transliterate Cyrillic to Latin
	id = transliterateCyrillic(id)
	// Replace spaces, underscores, slashes, colons, periods, and double hyphens with hyphens
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, "_", "-")
	id = strings.ReplaceAll(id, "/", "-")
	id = strings.ReplaceAll(id, ":", "-")
	id = strings.ReplaceAll(id, ".", "-")
	id = strings.ReplaceAll(id, "--", "-")
	id = strings.TrimSpace(id)
	id = strings.Trim(id, "-")
	// Remove special characters
	id = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return -1
	}, id)

	// Remove multiple consecutive hyphens
	for strings.Contains(id, "--") {
		id = strings.ReplaceAll(id, "--", "-")
	}

	// Trim hyphens from start and end
	id = strings.Trim(id, "-")

	return id
}

// isImageAsset checks if a file is an image based on its extension
func isImageAsset(filename string) bool {
	switch strings.ToLower(filepath.Ext(strings.TrimSpace(filename))) {
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".bmp", ".tif", ".tiff", ".ico", ".avif":
		return true
	default:
		return false
	}
}

// normalizeAssetName normalizes an asset filename to match MediaWiki conventions
func normalizeAssetName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}

	// Replace problematic characters with underscores
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "\u200e", "")
	runes := []rune(name)
	// Capitalize the first rune
	runes[0] = unicode.ToUpper(runes[0])

	return string(runes)
}

func subdirByNamespace(namespace int) string {
	// Use numeric namespace
	switch namespace {
	case 0:
		return ""
	case 1:
		return "talk"
	case 2:
		return "user"
	case 4:
		return "project"
	case 6:
		return "file"
	case 8:
		return "mediawiki"
	case 10:
		return "template"
	case 12:
		return "help"
	case 14:
		return "category"
	default:
		return fmt.Sprintf("ns-%d", namespace)
	}
}

func subdirByNamespacePrefix(namespacePrefix string) string {
	// Map namespace to subdirectory
	switch namespacePrefix {
	case "category":
		return "category"
	case "file", "image", "Файл", "Изображение":
		return "file"
	case "help":
		return "help"
	case "template":
		return "template"
	case "user":
		return "user"
	case "project":
		return "project"
	case "talk":
		return "talk"
	default:
		return namespacePrefix
	}
}

// ExtractRedirectTarget extracts the target page from a MediaWiki redirect
// MediaWiki redirects have the format: #REDIRECT [[Target Page]]
func ExtractRedirectTarget(content string) string {
	// Match various redirect formats
	// #REDIRECT [[Page]]
	// #redirect [[Page]]
	// #перенаправление [[Page]] (Russian)
	redirectPattern := regexp.MustCompile(`(?i)^#(?:redirect|перенаправление)\s*\[\[([^\]]+)\]\]`)

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if matches := redirectPattern.FindStringSubmatch(line); matches != nil {
			target := strings.TrimSpace(matches[1])
			// Remove any anchor/section references
			if idx := strings.Index(target, "#"); idx != -1 {
				target = target[:idx]
			}
			// Remove any pipe-separated display text
			if idx := strings.Index(target, "|"); idx != -1 {
				target = target[:idx]
			}
			return strings.TrimSpace(target)
		}
	}

	return ""
}

// generateIDByTitle creates a valid Docusaurus ID from a title by converting to lowercase,
// transliterating Cyrillic characters, and removing special characters
func generateIDByTitle(title string) string {
	actualTitle, _ := extractSubDirFromInternalLink(title, 0)
	id := prepareID(actualTitle)

	return id
}

// GeneratePageFilepath creates a filepath and subdirectory from a title and namespace,
// organizing files into subdirectories based on namespace
func GeneratePageFilepath(title string, namespace int) (string, string) {
	// Handle namespace prefixes in title (e.g., "Help:Getting Started")
	// This takes precedence over the numeric namespace parameter
	var subdir string
	title, subdir = extractSubDirFromInternalLink(title, namespace)

	var filename string
	filename = prepareID(title) + ".md"

	return filename, subdir
}

// generateAssetURL generates the URL for an asset based on its type
func generateAssetURL(filename, imageBaseURL, fileBaseURL string) string {
	if isImageAsset(filename) {
		return strings.TrimRight(imageBaseURL, "/") + "/" + filename
	}

	return strings.TrimRight(fileBaseURL, "/") + "/" + filename
}

func ConvertInternalLink(target, pageBaseURL, imageBaseURL, fileBaseURL string, namespace int) string {
	target = strings.TrimSpace(target)

	// Handle anchors (sections within pages)
	// Example: [[Article#Section]] -> article#section
	var anchor string
	target, anchor = extractAnchorFromLink(target)

	// Handle namespace prefixes
	// Example: [[Help:Getting Started]] -> /help/getting-started
	var subdir string
	target, subdir = extractSubDirFromInternalLink(target, namespace)

	var filename string
	if subdir == "file" {
		filename = normalizeAssetName(target)
		return generateAssetURL(filename, imageBaseURL, fileBaseURL) + anchor
	}

	// Convert to slug format
	slug := prepareID(target)

	// Build final link
	if subdir != "" {
		return pageBaseURL + subdir + "/" + slug + anchor
	}

	return pageBaseURL + slug + anchor
}

func extractAnchorFromLink(link string) (string, string) {
	// Handle anchors (sections within pages)
	// Example: [[Article#Section]] -> article#section
	var anchor string
	if strings.Contains(link, "#") {
		parts := strings.SplitN(link, "#", 2)
		link = parts[0]
		anchor = "#" + strings.ToLower(strings.ReplaceAll(parts[1], " ", "-"))
	}

	return link, anchor
}

func extractSubDirFromInternalLink(link string, namespace int) (string, string) {
	var subdir string
	if strings.Contains(link, ":") {
		parts := strings.SplitN(link, ":", 2)
		namespacePrefix := strings.ToLower(strings.TrimSpace(parts[0]))
		link = parts[1]
		link = strings.Trim(link, ":")
		// Map namespace prefixes to subdirectories (matching convertInternalLinkTarget)
		subdir = subdirByNamespacePrefix(namespacePrefix)
	} else {
		// Fall back to numeric namespace if no prefix in title
		subdir = subdirByNamespace(namespace)
	}

	return link, subdir
}
