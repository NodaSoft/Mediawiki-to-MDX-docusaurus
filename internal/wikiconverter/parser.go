package wikiconverter

import (
	"fmt"
	"html"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const imagesBaseURL = "/img/"
const filesBaseURL = "/files/"

var htmlTagNames = map[string]struct{}{
	"a": {}, "abbr": {}, "address": {}, "article": {}, "aside": {}, "audio": {}, "b": {}, "base": {}, "bdi": {}, "bdo": {}, "blockquote": {}, "body": {}, "br": {}, "button": {}, "canvas": {}, "caption": {}, "cite": {}, "code": {}, "col": {}, "colgroup": {}, "data": {}, "datalist": {}, "dd": {}, "del": {}, "details": {}, "dfn": {}, "dialog": {}, "div": {}, "dl": {}, "dt": {}, "em": {}, "embed": {}, "fieldset": {}, "figcaption": {}, "figure": {}, "footer": {}, "form": {}, "h1": {}, "h2": {}, "h3": {}, "h4": {}, "h5": {}, "h6": {}, "head": {}, "header": {}, "hgroup": {}, "hr": {}, "html": {}, "i": {}, "iframe": {}, "img": {}, "input": {}, "ins": {}, "kbd": {}, "label": {}, "legend": {}, "li": {}, "link": {}, "main": {}, "map": {}, "mark": {}, "meta": {}, "meter": {}, "nav": {}, "noscript": {}, "object": {}, "ol": {}, "optgroup": {}, "option": {}, "output": {}, "p": {}, "param": {}, "picture": {}, "pre": {}, "progress": {}, "q": {}, "rp": {}, "rt": {}, "ruby": {}, "s": {}, "samp": {}, "script": {}, "section": {}, "select": {}, "small": {}, "source": {}, "span": {}, "strong": {}, "style": {}, "sub": {}, "summary": {}, "sup": {}, "table": {}, "tbody": {}, "td": {}, "template": {}, "textarea": {}, "tfoot": {}, "th": {}, "thead": {}, "time": {}, "title": {}, "tr": {}, "track": {}, "u": {}, "ul": {}, "var": {}, "video": {}, "wbr": {},
}

// WikiParser converts MediaWiki markup to Markdown
type WikiParser struct {
	// Regex patterns for conversion
	patterns map[string]*regexp.Regexp
	// Base URL for images (if provided)
	imageBaseURL string
	fileBaseURL  string
	redirects    []Redirect
	assets       []string
}

func (p *WikiParser) SetRedirects(redirects []Redirect) {
	p.redirects = redirects
}

// NewWikiParser creates a new wiki parser
func NewWikiParser() *WikiParser {
	return NewWikiParserWithImageURL("", "")
}

// NewWikiParserWithImageURL creates a new wiki parser with custom image base URL
func NewWikiParserWithImageURL(imageBaseURL string, fileBaseURL string) *WikiParser {
	if imageBaseURL == "" {
		imageBaseURL = imagesBaseURL
	}

	if fileBaseURL == "" {
		fileBaseURL = filesBaseURL
	}

	return &WikiParser{
		imageBaseURL: imageBaseURL,
		fileBaseURL:  fileBaseURL,
		patterns: map[string]*regexp.Regexp{
			// Headers
			"h6": regexp.MustCompile(`(?m)^======\s*(.+?)\s*======\s*$`),
			"h5": regexp.MustCompile(`(?m)^=====\s*(.+?)\s*=====\s*$`),
			"h4": regexp.MustCompile(`(?m)^====\s*(.+?)\s*====\s*$`),
			"h3": regexp.MustCompile(`(?m)^===\s*(.+?)\s*===\s*$`),
			"h2": regexp.MustCompile(`(?m)^==\s*(.+?)\s*==\s*$`),

			// Bold and italic
			"boldItalic": regexp.MustCompile(`'''''(.+?)'''''`),
			"bold":       regexp.MustCompile(`'''(.+?)'''`),
			"italic":     regexp.MustCompile(`''(.+?)''`),
			"underline":  regexp.MustCompile(`(?s)<u>(.*?)</u>`),
			"boldHTML":   regexp.MustCompile(`(?s)<b>(.*?)</b>`),

			// Links
			"extLink": regexp.MustCompile(`\[(https?://[^\s\]]+)\s+([^\[\]]+?)\]`),
			"intLink": regexp.MustCompile(`\[\[([^\[\]|]+?)(?:\|([^\[\]]+?))?\]\]`),

			// Images and files
			"imageFile": regexp.MustCompile(`\[\[(File|Image|Файл|Изображение):([^\[\]|]+?)(?:\|([^\[\]]*?))?\]\]`),

			// Lists
			"unordered": regexp.MustCompile(`(?m)^\*+\s`),
			"ordered":   regexp.MustCompile(`(?m)^#+\s`),
			"htmlList":  regexp.MustCompile(`(?is)<\s*(/?)\s*(ul|ol|li)\b[^>]*>`),

			// Code
			"code":       regexp.MustCompile(`<code>(.*?)</code>`),
			"pre":        regexp.MustCompile(`(?is)<pre\b[^>]*>(.*?)</pre>`),
			"nowiki":     regexp.MustCompile(`<nowiki>(.*?)</nowiki>`),
			"syntaxhl":   regexp.MustCompile(`(?s)<syntaxhighlight[^>]*lang="([^"]*)"[^>]*>(.*?)</syntaxhighlight>`),
			"source":     regexp.MustCompile(`(?s)<source\b([^>]*)>(.*?)</source>`),
			"blockquote": regexp.MustCompile(`(?is)<blockquote\b[^>]*>(.*?)</blockquote>`),
			"json":       regexp.MustCompile(`({[^\n}]*})`),

			// Templates (basic handling)
			"template": regexp.MustCompile(`\{\{([^}]+)\}\}`),

			// HTML tags to remove
			"htmlComment": regexp.MustCompile(`<!--.*?-->`),
			"br":          regexp.MustCompile(`(?i)<br\b[^>]*>`),
			"hr":          regexp.MustCompile(`(?i)<hr\b[^>]*>`),
			"p":           regexp.MustCompile(`(?i)</?p\b[^>]*>`),
			"small":       regexp.MustCompile(`(?i)</?small\b[^>]*>`),
			"sub":         regexp.MustCompile(`(?i)</?sub\b[^>]*>`),
			"div":         regexp.MustCompile(`(?i)</?div[^>]*>`),
			"span":        regexp.MustCompile(`(?i)</?span[^>]*>`),
			"sidebarmenu": regexp.MustCompile(`(?i)</?sidebarmenu[^>]*>`),
			"font":        regexp.MustCompile(`(?i)<font\s+color=["']?([a-z#0-9]+)["']?>([\S\s]*?)<\/font>`),

			// Non-HTML tags
			"nonHTMLTags": regexp.MustCompile(`(?i)<(/?)([a-zа-я][a-zа-я0-9:_-]+)([^><]*)?/?>`),
			// Greater than followed by a number
			"gtPlusNumber": regexp.MustCompile(`>(\d+)`),
			// Less than followed by a number
			"ltPlusNumber": regexp.MustCompile(`<(\d+)`),
		},
	}
}

// Parse converts MediaWiki markup to Markdown
func (p *WikiParser) Parse(wikitext string) string {
	text := wikitext

	// Convert lists
	text = p.convertLists(text)

	// Convert simple HTML tags
	text = p.convertSimpleHTML(text)

	// Convert images (before links to avoid conflicts)
	text = p.convertAssets(text)

	// Convert external links
	text = p.convertExternalLink(text)

	// Convert internal links
	text = p.convertInternalLink(text)

	// Convert MediaWiki tables
	text = p.convertTables(text)

	// Convert HTML tables
	text = p.convertHTMLTables(text)

	// Convert HTML lists
	text = p.convertHTMLLists(text)

	// Clean up multiple blank lines
	text = regexp.MustCompile(`\n{3,}`).ReplaceAllString(text, "\n\n")
	text = strings.ReplaceAll(text, "<=", "≤")
	text = strings.ReplaceAll(text, ">=", "≥")
	text = strings.ReplaceAll(text, "<>", "≠")
	text = strings.ReplaceAll(text, " clear=\"all\"", "")
	// Replace non-breaking space with regular space
	text = strings.ReplaceAll(text, "\u00a0", " ")

	// Convert json
	text = p.convertJSON(text)

	text = p.convertFontTags(text)
	text = p.convertHTMLStylesToMDX(text)
	text = p.codeifyNonHTMLTags(text)

	text = p.patterns["gtPlusNumber"].ReplaceAllString(text, "&gt;$1")
	text = p.patterns["ltPlusNumber"].ReplaceAllString(text, "&lt;$1")

	return strings.TrimSpace(text)
}

// convertJSON wraps JSON objects in inline code blocks to prevent MDX parsing issues
func (p *WikiParser) convertJSON(text string) string {
	matches := p.patterns["json"].FindAllStringIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	var result strings.Builder
	lastEnd := 0

	for _, match := range matches {
		start := match[0]
		end := match[1]

		// Check if already wrapped in backticks
		alreadyWrapped := false

		// Check before the match
		if start > 0 && text[start-1] == '`' {
			alreadyWrapped = true
		}

		// Check after the match
		if !alreadyWrapped && end < len(text) && text[end] == '`' {
			alreadyWrapped = true
		}

		// Write everything before this match
		result.WriteString(text[lastEnd:start])

		if alreadyWrapped {
			// Keep as-is
			result.WriteString(text[start:end])
		} else {
			// Wrap in inline code
			jsonContent := text[start:end]
			result.WriteString(wrapInlineCode(jsonContent))
		}

		lastEnd = end
	}

	// Write remaining text
	result.WriteString(text[lastEnd:])

	return result.String()
}

// convertSimpleHTML converts basic HTML tags to Markdown equivalents
func (p *WikiParser) convertSimpleHTML(text string) string {
	// Remove HTML comments
	text = p.patterns["htmlComment"].ReplaceAllString(text, "")

	// Convert headers (from largest to smallest to avoid conflicts)
	text = p.patterns["h6"].ReplaceAllString(text, "###### $1")
	text = p.patterns["h5"].ReplaceAllString(text, "##### $1")
	text = p.patterns["h4"].ReplaceAllString(text, "#### $1")
	text = p.patterns["h3"].ReplaceAllString(text, "### $1")
	text = p.patterns["h2"].ReplaceAllString(text, "## $1")

	// Convert bold and italic (order matters!)
	text = p.patterns["underline"].ReplaceAllString(text, "__${1}__")
	text = p.patterns["boldHTML"].ReplaceAllString(text, "**${1}**")
	text = p.patterns["boldItalic"].ReplaceAllString(text, "***$1***")
	text = p.patterns["bold"].ReplaceAllString(text, "**$1**")
	text = p.patterns["italic"].ReplaceAllString(text, "*$1*")

	// Convert syntax highlighting
	text = p.patterns["syntaxhl"].ReplaceAllStringFunc(text, func(match string) string {
		matches := p.patterns["syntaxhl"].FindStringSubmatch(match)
		if len(matches) >= 3 {
			return wrapMultiLineCode(matches[2], matches[1])
		}
		return match
	})

	// Convert pre tags
	text = p.patterns["pre"].ReplaceAllStringFunc(text, func(match string) string {
		matches := p.patterns["pre"].FindStringSubmatch(match)
		if len(matches) >= 2 {
			return wrapMultiLineCode(matches[1], "")
		}
		return match
	})

	// Convert nowiki tags
	text = p.patterns["nowiki"].ReplaceAllStringFunc(text, func(match string) string {
		matches := p.patterns["nowiki"].FindStringSubmatch(match)
		if len(matches) >= 1 {
			return wrapInlineCode(matches[1])
		}
		return match
	})

	// Convert inline code
	text = p.patterns["code"].ReplaceAllStringFunc(text, func(match string) string {
		matches := p.patterns["code"].FindStringSubmatch(match)
		if len(matches) >= 2 {
			return wrapInlineCode(matches[1])
		}
		return match
	})

	// Convert source tags
	text = p.patterns["source"].ReplaceAllStringFunc(text, func(match string) string {
		matches := p.patterns["source"].FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}

		attrs := matches[1]
		code := strings.TrimSpace(matches[2])
		lang := ""

		if attrMatch := regexp.MustCompile(`(?i)(?:lang|language)\s*=\s*["']([^"']+)["']`).FindStringSubmatch(attrs); len(attrMatch) >= 2 {
			lang = strings.TrimSpace(attrMatch[1])
		}

		return wrapMultiLineCode(code, lang)
	})

	// Convert blockquote tags
	text = p.patterns["blockquote"].ReplaceAllStringFunc(text, func(match string) string {
		matches := p.patterns["blockquote"].FindStringSubmatch(match)
		if len(matches) < 2 {
			return match
		}

		content := strings.TrimSpace(matches[1])
		if content == "" {
			return ""
		}

		lines := strings.Split(content, "\n")
		for i, line := range lines {
			lines[i] = "> " + strings.TrimSpace(line)
		}
		return "\n" + strings.Join(lines, "\n") + "\n"
	})

	// Handle templates (basic - just remove or convert to notes)
	text = p.patterns["template"].ReplaceAllStringFunc(text, func(match string) string {
		matches := p.patterns["template"].FindStringSubmatch(match)
		if len(matches) >= 2 {
			content := strings.TrimSpace(matches[1])
			// Check for common templates
			if strings.HasPrefix(strings.ToLower(content), "note") ||
				strings.HasPrefix(strings.ToLower(content), "warning") ||
				strings.HasPrefix(strings.ToLower(content), "info") {
				parts := strings.SplitN(content, "|", 2)
				if len(parts) >= 2 {
					return ":::note\n" + strings.TrimSpace(parts[1]) + "\n:::"
				}
			}
			// For other templates, just return the content
			return ""
		}
		return ""
	})

	// Clean up HTML tags
	text = p.patterns["br"].ReplaceAllString(text, "\n")
	text = p.patterns["hr"].ReplaceAllString(text, "\n\n---\n\n")
	text = p.patterns["p"].ReplaceAllString(text, "")
	text = p.patterns["small"].ReplaceAllString(text, "")
	text = p.patterns["sub"].ReplaceAllString(text, "")
	text = p.patterns["div"].ReplaceAllString(text, "")
	text = p.patterns["span"].ReplaceAllString(text, "")
	text = p.patterns["sidebarmenu"].ReplaceAllString(text, "")

	return text
}

func cleanLinkLabel(label string) string {
	label = strings.TrimSpace(label)
	label = strings.ReplaceAll(label, "|", "")
	label = strings.ReplaceAll(label, ">", "")
	label = strings.ReplaceAll(label, "<", "")
	label = strings.ReplaceAll(label, "-", "")
	label = strings.ReplaceAll(label, "/", "")
	label = strings.TrimSpace(label)

	return label
}

func (p *WikiParser) convertExternalLink(text string) string {
	text = p.patterns["extLink"].ReplaceAllStringFunc(text, func(match string) string {
		matches := p.patterns["extLink"].FindStringSubmatch(match)
		if len(matches) >= 3 {
			url := matches[1]
			label := cleanLinkLabel(matches[2])
			return "[" + label + "](" + url + ")"
		}
		return match
	})

	return text
}

// convertInternalLink converts MediaWiki internal links to Markdown links
func (p *WikiParser) convertInternalLink(text string) string {
	text = p.patterns["intLink"].ReplaceAllStringFunc(text, func(match string) string {
		matches := p.patterns["intLink"].FindStringSubmatch(match)
		if len(matches) >= 2 {
			target := matches[1]
			label := target
			if len(matches) >= 3 && matches[2] != "" {
				label = matches[2]
			}
			if strings.HasPrefix(target, "Категория") || strings.HasPrefix(target, "Category") {
				return "" // Skip categories links
			}

			// Resolve redirects (including chains)
			target = p.resolveRedirect(target)

			// Convert to relative link
			link := p.convertInternalLinkTarget(target)
			label = cleanLinkLabel(label)
			return "[" + label + "](" + link + ")"
		}
		return match
	})

	return strings.TrimSpace(text)
}

// resolveRedirect resolves a page name through redirect chains
// Returns the final target page after following all redirects
func (p *WikiParser) resolveRedirect(pageName string) string {
	if len(p.redirects) == 0 {
		return pageName
	}

	visited := make(map[string]bool)
	current := strings.TrimSpace(pageName)

	// Follow redirect chain, protecting against infinite loops
	for {
		// Check if we've already visited this page (circular redirect)
		if visited[current] {
			// Circular redirect detected, return current to avoid infinite loop
			return current
		}

		visited[current] = true

		// Look for a redirect from current page
		found := false
		for _, redirect := range p.redirects {
			if strings.EqualFold(strings.TrimSpace(redirect.From), current) {
				current = strings.TrimSpace(redirect.To)
				found = true
				break
			}
		}

		// No more redirects found, we've reached the final target
		if !found {
			return current
		}
	}
}

// codeifyNonHTMLTags wraps non-HTML tags in inline code blocks
// This prevents MDX from trying to parse custom wiki tags as JSX
func (p *WikiParser) codeifyNonHTMLTags(text string) string {
	return p.patterns["nonHTMLTags"].ReplaceAllStringFunc(text, func(match string) string {
		parts := p.patterns["nonHTMLTags"].FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}

		tagName := strings.ToLower(parts[2])
		if _, ok := htmlTagNames[tagName]; ok {
			return match
		}

		return wrapInlineCode(match)
	})
}

// wrapInlineCode wraps content in backticks, using enough backticks to avoid conflicts
func wrapInlineCode(content string) string {
	content = strings.TrimSpace(content)

	maxRun := 0
	current := 0
	for _, r := range content {
		if r == '`' {
			current++
			if current > maxRun {
				maxRun = current
			}
			continue
		}
		current = 0
	}

	fence := strings.Repeat("`", maxRun+1)

	return fence + content + fence
}

// convertFontTags converts HTML font tags with color to MDX-compatible span elements
func (p *WikiParser) convertFontTags(text string) string {
	return p.patterns["font"].ReplaceAllStringFunc(text, func(match string) string {
		parts := p.patterns["font"].FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}

		color := parts[1]
		content := parts[2]

		// Convert to MDX format with inline style object
		return fmt.Sprintf(`<span style={{color: '%s'}}>%s</span>`, color, content)
	})
}

// convertHTMLStylesToMDX converts HTML style attributes to MDX format
// Converts style="property: value;" to style={{property: 'value'}}
func (p *WikiParser) convertHTMLStylesToMDX(text string) string {
	// Pattern to match style attributes in HTML tags
	stylePattern := regexp.MustCompile(`(?i)\s+style\s*=\s*"([^"]*)"`)

	return stylePattern.ReplaceAllStringFunc(text, func(match string) string {
		parts := stylePattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			return match
		}

		styleContent := strings.TrimSpace(parts[1])
		if styleContent == "" {
			return match
		}

		// Parse CSS style declarations
		declarations := strings.Split(styleContent, ";")
		var mdxProps []string

		for _, decl := range declarations {
			decl = strings.TrimSpace(decl)
			if decl == "" {
				continue
			}

			// Split property and value
			parts := strings.SplitN(decl, ":", 2)
			if len(parts) != 2 {
				continue
			}

			property := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Convert CSS property names to camelCase for MDX
			property = cssToCamelCase(property)

			// Format as JavaScript object property
			mdxProps = append(mdxProps, fmt.Sprintf("%s: '%s'", property, value))
		}

		if len(mdxProps) == 0 {
			return match
		}

		// Return MDX-formatted style attribute
		return fmt.Sprintf(` style={{%s}}`, strings.Join(mdxProps, ", "))
	})
}

// cssToCamelCase converts CSS property names to camelCase
// e.g., "background-color" -> "backgroundColor"
func cssToCamelCase(cssProperty string) string {
	parts := strings.Split(cssProperty, "-")
	if len(parts) == 1 {
		return parts[0]
	}

	result := parts[0]
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			result += strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}

	return result
}

// convertTables converts MediaWiki table syntax to Markdown tables.
func (p *WikiParser) convertTables(text string) string {
	return convertAllMediaWikiTables(text)
}

// convertLists converts wiki lists to markdown lists
func (p *WikiParser) convertLists(text string) string {
	lines := strings.Split(text, "\n")
	var result []string

	for _, line := range lines {
		// Unordered lists
		if strings.HasPrefix(line, "*") {
			level := 0
			for i := 0; i < len(line) && line[i] == '*'; i++ {
				level++
			}
			indent := strings.Repeat("  ", level-1)
			content := strings.TrimSpace(line[level:])
			result = append(result, indent+"- "+content)
			continue
		}

		// Ordered lists
		if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "##") {
			level := 0
			for i := 0; i < len(line) && line[i] == '#'; i++ {
				level++
			}
			indent := strings.Repeat("  ", level-1)
			content := strings.TrimSpace(line[level:])
			result = append(result, indent+"1. "+content)
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// convertHTMLTables converts HTML <table> markup to Markdown tables.
func (p *WikiParser) convertHTMLTables(text string) string {
	tableRe := regexp.MustCompile(`(?is)<table\b[^>]*>(.*?)</table>`)
	rowRe := regexp.MustCompile(`(?is)<tr\b[^>]*>(.*?)</tr>`)
	cellRe := regexp.MustCompile(`(?is)<(th|td)\b[^>]*>(.*?)</(th|td)>`)
	stripTagsRe := regexp.MustCompile(`(?is)<[^>]+>`)

	return tableRe.ReplaceAllStringFunc(text, func(tableMatch string) string {
		tableParts := tableRe.FindStringSubmatch(tableMatch)
		if len(tableParts) < 2 {
			return tableMatch
		}

		rowsRaw := rowRe.FindAllStringSubmatch(tableParts[1], -1)
		if len(rowsRaw) == 0 {
			return ""
		}

		var rows [][]string
		hasHeader := false
		maxCols := 0

		for _, rowRaw := range rowsRaw {
			if len(rowRaw) < 2 {
				continue
			}

			cellsRaw := cellRe.FindAllStringSubmatch(rowRaw[1], -1)
			if len(cellsRaw) == 0 {
				continue
			}

			row := make([]string, 0, len(cellsRaw))
			rowHasTH := false

			for _, c := range cellsRaw {
				if len(c) < 3 {
					continue
				}
				if strings.EqualFold(c[1], "th") {
					rowHasTH = true
				}

				content := strings.TrimSpace(c[2])
				content = p.patterns["br"].ReplaceAllString(content, " <br/> ")
				content = stripTagsRe.ReplaceAllString(content, "")
				content = html.UnescapeString(strings.TrimSpace(content))
				content = strings.ReplaceAll(content, "|", `\|`)
				row = append(row, content)
			}

			if len(row) == 0 {
				continue
			}

			if rowHasTH {
				hasHeader = true
			}

			if len(row) > maxCols {
				maxCols = len(row)
			}
			rows = append(rows, row)
		}

		if len(rows) == 0 || maxCols == 0 {
			return ""
		}

		for i := range rows {
			for len(rows[i]) < maxCols {
				rows[i] = append(rows[i], "")
			}
		}

		header := rows[0]
		body := rows[1:]
		if !hasHeader {
			body = rows[1:]
		}

		var sb strings.Builder
		sb.WriteString("| " + strings.Join(header, " | ") + " |\n")
		sb.WriteString("|")
		for i := 0; i < maxCols; i++ {
			sb.WriteString(" --- |")
		}
		sb.WriteString("\n")

		for _, row := range body {
			sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
		}

		return strings.TrimSpace(sb.String())
	})
}

// convertHTMLLists converts simple HTML <ul>/<ol>/<li> lists to Markdown lists.
func (p *WikiParser) convertHTMLLists(text string) string {
	matches := p.patterns["htmlList"].FindAllStringSubmatchIndex(text, -1)
	if len(matches) == 0 {
		return text
	}

	var out strings.Builder
	var item strings.Builder
	listStack := make([]string, 0)
	inItem := false
	itemDepth := 0
	itemOrdered := false

	writeItem := func() {
		content := strings.TrimSpace(item.String())
		item.Reset()
		if content == "" {
			return
		}

		if out.Len() > 0 && !strings.HasSuffix(out.String(), "\n") {
			out.WriteString("\n")
		}

		indent := ""
		if itemDepth > 1 {
			indent = strings.Repeat("  ", itemDepth-1)
		}
		marker := "-"
		if itemOrdered {
			marker = "1."
		}

		out.WriteString(indent + marker + " " + content + "\n")
	}

	prevEnd := 0
	for _, m := range matches {
		seg := text[prevEnd:m[0]]
		if inItem {
			item.WriteString(seg)
		} else {
			out.WriteString(seg)
		}

		isClose := m[2] != -1 && text[m[2]:m[3]] == "/"
		tagName := strings.ToLower(text[m[4]:m[5]])

		switch tagName {
		case "ul", "ol":
			if isClose {
				if len(listStack) > 0 {
					listStack = listStack[:len(listStack)-1]
				}
			} else {
				listStack = append(listStack, tagName)
			}
		case "li":
			if isClose {
				if inItem {
					writeItem()
					inItem = false
				}
			} else {
				if inItem {
					writeItem()
				}
				inItem = true
				itemDepth = len(listStack)
				itemOrdered = len(listStack) > 0 && listStack[len(listStack)-1] == "ol"
			}
		}

		prevEnd = m[1]
	}

	rest := text[prevEnd:]
	if inItem {
		item.WriteString(rest)
		writeItem()
	} else {
		out.WriteString(rest)
	}

	return out.String()
}

// convertAssets converts MediaWiki image syntax to Markdown
func (p *WikiParser) convertAssets(text string) string {
	text = p.patterns["imageFile"].ReplaceAllStringFunc(text, func(match string) string {
		matches := p.patterns["imageFile"].FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}

		filename := strings.TrimSpace(matches[2])
		options := ""
		if len(matches) >= 4 {
			options = matches[3]
		}

		// Parse options and caption
		caption := ""
		altText := filename
		imageLink := ""
		var optionParts []string
		if options != "" {
			optionParts = strings.Split(options, "|")
		}

		for _, rawPart := range optionParts {
			part := strings.TrimSpace(rawPart)
			if part == "" {
				continue
			}
			lowerPart := strings.ToLower(part)
			if strings.HasPrefix(lowerPart, "link=") {
				rawLink := strings.TrimSpace(part[len("link="):])
				if rawLink == "" {
					continue
				}
				if strings.HasPrefix(rawLink, "http://") || strings.HasPrefix(rawLink, "https://") {
					imageLink = rawLink
				} else {
					imageLink = p.convertInternalLinkTarget(rawLink)
				}
			}
		}

		// Extract caption (usually the last non-option part)
		for i := len(optionParts) - 1; i >= 0; i-- {
			part := strings.TrimSpace(optionParts[i])
			// Skip known options
			if !p.isImageOption(part) {
				caption = part
				altText = part
				break
			}
		}

		if caption == "" {
			caption = filename
			altText = filename
		}

		filename = normalizeAssetName(filename)
		if filename == "" {
			return ""
		}

		// Collect image filenames
		p.assets = append(p.assets, filename)

		assetURL := p.generateAssetURL(filename)

		if isImageAsset(filename) {
			// Return markdown image syntax, optionally wrapped in a link.
			imageMarkdown := fmt.Sprintf("![%s](%s)", altText, assetURL)
			if imageLink != "" {
				return fmt.Sprintf("[%s](%s)", imageMarkdown, imageLink)
			}
			return imageMarkdown
		}

		// Non-image wiki attachments should become regular links in markdown.
		linkText := caption
		if linkText == "" {
			linkText = filename
		}

		return fmt.Sprintf("[%s](%s)", linkText, assetURL)
	})

	return text
}

// isImageOption checks if a string is a known MediaWiki image option
func (p *WikiParser) isImageOption(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	options := []string{
		"thumb", "thumbnail", "frame", "frameless",
		"border", "left", "right", "center", "none",
		"baseline", "middle", "sub", "super", "text-top",
		"text-bottom", "top", "bottom",
	}

	for _, opt := range options {
		if s == opt {
			return true
		}
	}

	// Check for size options like "200px" or "x300px"
	if strings.HasSuffix(s, "px") {
		return true
	}

	return false
}

// generateAssetURL generates the URL for an asset based on its type
func (p *WikiParser) generateAssetURL(filename string) string {
	if isImageAsset(filename) {
		return strings.TrimRight(p.imageBaseURL, "/") + "/" + filename
	}

	return strings.TrimRight(p.fileBaseURL, "/") + "/" + filename
}

// assetsUnique returns a unique set of all assets found during parsing
func (p *WikiParser) assetsUnique() map[string]struct{} {
	m := make(map[string]struct{})
	for _, asset := range p.assets {
		m[asset] = struct{}{}
	}

	return m
}

// convertInternalLink converts a MediaWiki internal link to a relative path
func (p *WikiParser) convertInternalLinkTarget(target string) string {
	target = strings.TrimSpace(target)

	// Handle anchors (sections within pages)
	// Example: [[Article#Section]] -> article#section
	var anchor string
	if strings.Contains(target, "#") {
		parts := strings.SplitN(target, "#", 2)
		target = parts[0]
		anchor = "#" + strings.ToLower(strings.ReplaceAll(parts[1], " ", "-"))
	}

	// Handle namespace prefixes
	// Example: [[Help:Getting Started]] -> /help/getting-started
	var prefix string
	if strings.Contains(target, ":") {
		parts := strings.SplitN(target, ":", 2)
		namespace := strings.ToLower(parts[0])
		target = parts[1]
		target = strings.Trim(target, ":")

		// Map common namespaces to paths
		switch namespace {
		case "file", "image":
			filename := normalizeAssetName(target)
			if filename == "" {
				return ""
			}
			if isImageAsset(filename) {
				return strings.TrimRight(p.imageBaseURL, "/") + "/" + filename + anchor
			}
			return strings.TrimRight(p.fileBaseURL, "/") + "/" + filename + anchor
		default:
			// Unknown namespace, keep as-is
			prefix = namespace + "/"
		}
	}

	// Convert to slug format
	slug := strings.ToLower(target)
	// Transliterate Cyrillic to Latin
	slug = transliterateCyrillic(slug)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	slug = strings.ReplaceAll(slug, "/", "-")
	slug = strings.ReplaceAll(slug, ":", "-")
	slug = strings.ReplaceAll(slug, ".", "-")
	slug = strings.ReplaceAll(slug, "--", "-")
	slug = strings.TrimSpace(slug)
	slug = strings.Trim(slug, "-")

	// Remove special characters except hyphens
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, slug)

	// Remove multiple consecutive hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	// Build final link
	if prefix != "" {
		return prefix + slug + anchor
	}
	return slug + anchor
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

// wrapMultiLineCode wraps content in a code fence with optional language
func wrapMultiLineCode(content string, lang string) string {
	content = strings.TrimSpace(content)
	return fmt.Sprintf("\n```%s\n%s\n```\n", lang, content)
}
