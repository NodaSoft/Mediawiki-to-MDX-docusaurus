package wikiconverter

import (
	"bufio"
	"regexp"
	"strings"
)

var tableAttrOnlyPattern = regexp.MustCompile(`(?i)^([a-z_:][-a-z0-9_:.]*\s*=\s*(?:"[^"]*"|'[^']*'|[^\s]+)\s*)+$`)

// convertAllMediaWikiTables converts all MediaWiki tables in a document to Markdown.
func convertAllMediaWikiTables(input string) string {
	var output strings.Builder
	var tableBuffer strings.Builder

	inTable := false
	scanner := bufio.NewScanner(strings.NewReader(input))

	for scanner.Scan() {
		line := scanner.Text()
		normalized := normalizeTableEscapes(line)
		trimmed := strings.TrimSpace(normalized)

		if strings.HasPrefix(trimmed, "{|") {
			if inlineLines, ok := splitInlineTableBlock(trimmed); ok {
				tableBuffer.Reset()
				tableBuffer.WriteString(strings.Join(inlineLines, "\n"))
				tableBuffer.WriteString("\n")
				output.WriteString(convertSingleTable(tableBuffer.String()) + "\n")
				inTable = false
				continue
			}

			inTable = true
			tableBuffer.Reset()
			tableBuffer.WriteString(normalized + "\n")
			continue
		}

		if inTable {
			tableBuffer.WriteString(normalized + "\n")

			if strings.HasPrefix(trimmed, "|}") {
				// End of table — convert and append
				md := convertSingleTable(tableBuffer.String())
				output.WriteString(md + "\n")
				inTable = false
			}
			continue
		}

		// Non-table content
		output.WriteString(line + "\n")
	}

	return output.String()
}

// normalizeTableEscapes removes backslash escapes from MediaWiki table syntax
func normalizeTableEscapes(line string) string {
	line = strings.TrimSpace(line)
	line = strings.ReplaceAll(line, `{\|`, `{|`)
	line = strings.ReplaceAll(line, `\|}`, `|}`)

	trimmed := strings.TrimLeft(line, " \t")
	indent := line[:len(line)-len(trimmed)]
	switch {
	case strings.HasPrefix(trimmed, `\|-`):
		return indent + "|-" + trimmed[3:]
	case strings.HasPrefix(trimmed, `\|+`):
		return indent + "|+" + trimmed[3:]
	case strings.HasPrefix(trimmed, `\|`):
		return indent + "|" + trimmed[2:]
	case strings.HasPrefix(trimmed, `\!`):
		return indent + "!" + trimmed[2:]
	}

	return line
}

// splitInlineTableBlock splits a single-line table into multiple lines
// Returns the split lines and true if the line contains a complete inline table
func splitInlineTableBlock(line string) ([]string, bool) {
	if !strings.HasPrefix(line, "{|") {
		return nil, false
	}

	closeIdx := strings.LastIndex(line, "|}")
	if closeIdx == -1 {
		return nil, false
	}

	if strings.TrimSpace(line[closeIdx+2:]) != "" {
		return nil, false
	}

	body := strings.TrimSpace(line[2:closeIdx])
	if body == "" {
		return []string{"{|", "|}"}, true
	}

	segments := splitTableRowsIgnoringBackticks(body)
	out := []string{"{|"}
	for idx, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}

		if idx > 0 {
			out = append(out, "|-")
		}

		if idx == 0 {
			start := firstInlineRowStart(segment)
			if start == -1 {
				continue
			}
			segment = strings.TrimSpace(segment[start:])
			if segment == "" {
				continue
			}
		}

		if strings.HasPrefix(segment, "|+") || strings.HasPrefix(segment, "!") || strings.HasPrefix(segment, "|") {
			out = append(out, segment)
		}
	}

	out = append(out, "|}")
	return out, true
}

// splitTableRowsIgnoringBackticks splits table body by "|-" delimiter
// while ignoring any "|-" that appears within backticks (code blocks)
func splitTableRowsIgnoringBackticks(body string) []string {
	var segments []string
	var currentSegment strings.Builder
	inBackticks := false

	for i := 0; i < len(body); i++ {
		if body[i] == '`' {
			inBackticks = !inBackticks
			currentSegment.WriteByte(body[i])
			continue
		}

		// Check for "|-" delimiter only if not inside backticks
		if !inBackticks && i+1 < len(body) && body[i] == '|' && body[i+1] == '-' {
			// Found a row delimiter outside of backticks
			segments = append(segments, currentSegment.String())
			currentSegment.Reset()
			i++ // Skip the '-' character
			continue
		}

		currentSegment.WriteByte(body[i])
	}

	// Add the last segment
	if currentSegment.Len() > 0 {
		segments = append(segments, currentSegment.String())
	}

	return segments
}

// firstInlineRowStart finds the position where the first table row starts
func firstInlineRowStart(segment string) int {
	for i := 0; i < len(segment); i++ {
		if segment[i] == '!' {
			return i
		}
		if segment[i] == '|' && (i+1 >= len(segment) || segment[i+1] != '}') {
			return i
		}
	}
	return -1
}

// convertSingleTable converts a single MediaWiki table to Markdown format
func convertSingleTable(input string) string {
	var rows [][]string
	var currentRow []string
	hasHeader := false

	scanner := bufio.NewScanner(strings.NewReader(input))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "{|") || strings.HasPrefix(line, "|}") {
			continue
		}

		if strings.HasPrefix(line, "|-") {
			if len(currentRow) > 0 {
				rows = append(rows, currentRow)
				currentRow = nil
			}
			continue
		}

		if strings.HasPrefix(line, "!") {
			hasHeader = true
			line = strings.TrimPrefix(line, "!")
			cells := splitCells(line, "!!")
			currentRow = append(currentRow, cells...)
			continue
		}

		if strings.HasPrefix(line, "|") {
			line = strings.TrimPrefix(line, "|")
			cells := splitCells(line, "||")
			currentRow = append(currentRow, cells...)
			continue
		}
	}

	if len(currentRow) > 0 {
		rows = append(rows, currentRow)
	}

	if len(rows) == 0 {
		return ""
	}

	var sb strings.Builder

	// Header
	sb.WriteString("| " + strings.Join(rows[0], " | ") + " |\n")
	sb.WriteString("|")
	for range rows[0] {
		sb.WriteString(" --- |")
	}
	sb.WriteString("\n")

	start := 1
	if !hasHeader {
		start = 1
	}

	for _, row := range rows[start:] {
		sb.WriteString("| " + strings.Join(row, " | ") + " |\n")
	}

	return sb.String()
}

// splitCells splits a table row into cells and removes MediaWiki attributes
func splitCells(line, sep string) []string {
	parts := splitCellsIgnoringBackticks(line, sep)
	for i := range parts {
		cell := strings.TrimSpace(parts[i])

		// Remove attribute prefixes like: style="..." | Value
		// But only if the pipe is not inside backticks
		if pipeIdx := findFirstPipeOutsideBackticks(cell); pipeIdx != -1 {
			cell = strings.TrimSpace(cell[pipeIdx+1:])
		} else if tableAttrOnlyPattern.MatchString(cell) {
			// Attribute-only cell, no visible content.
			cell = ""
		}

		// Escape ALL pipe characters in cell content for Markdown
		cell = strings.ReplaceAll(cell, "|", `\|`)

		parts[i] = cell
	}
	return parts
}

// splitCellsIgnoringBackticks splits a line by separator while ignoring separators inside backticks
func splitCellsIgnoringBackticks(line, sep string) []string {
	var parts []string
	var currentPart strings.Builder
	inBackticks := false
	sepLen := len(sep)

	for i := 0; i < len(line); i++ {
		if line[i] == '`' {
			inBackticks = !inBackticks
			currentPart.WriteByte(line[i])
			continue
		}

		// Check for separator only if not inside backticks
		if !inBackticks && i+sepLen <= len(line) && line[i:i+sepLen] == sep {
			parts = append(parts, currentPart.String())
			currentPart.Reset()
			i += sepLen - 1 // Skip the separator (minus 1 because loop will increment)
			continue
		}

		currentPart.WriteByte(line[i])
	}

	// Add the last part
	parts = append(parts, currentPart.String())
	return parts
}

// findFirstPipeOutsideBackticks finds the first pipe character that's not inside backticks
func findFirstPipeOutsideBackticks(text string) int {
	inBackticks := false
	for i := 0; i < len(text); i++ {
		if text[i] == '`' {
			inBackticks = !inBackticks
			continue
		}
		if !inBackticks && text[i] == '|' {
			return i
		}
	}
	return -1
}
