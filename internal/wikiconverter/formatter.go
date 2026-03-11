package wikiconverter

import (
	"fmt"
	"strings"
)

// DocusaurusDoc represents a Docusaurus document
type DocusaurusDoc struct {
	Title       string
	ID          string
	Description string
	Sidebar     string
	Tags        []string
	Content     string
}

// DocusaurusFormatter formats documents for Docusaurus
type DocusaurusFormatter struct{}

// NewDocusaurusFormatter creates a new formatter
func NewDocusaurusFormatter() *DocusaurusFormatter {
	return &DocusaurusFormatter{}
}

// Format converts a document to Docusaurus MDX format
func (f *DocusaurusFormatter) Format(doc DocusaurusDoc) string {
	var sb strings.Builder

	// Write frontmatter
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", doc.ID))
	sb.WriteString(fmt.Sprintf("title: %s\n", f.escapeYAML(doc.Title)))

	if doc.Description != "" {
		sb.WriteString(fmt.Sprintf("description: %s\n", f.escapeYAML(doc.Description)))
	}

	if doc.Sidebar != "" {
		sb.WriteString(fmt.Sprintf("sidebar_position: %s\n", doc.Sidebar))
	}

	if len(doc.Tags) > 0 {
		sb.WriteString("tags:\n")
		for _, tag := range doc.Tags {
			sb.WriteString(fmt.Sprintf("  - %s\n", f.escapeYAML(tag)))
		}
	}

	sb.WriteString("---\n\n")

	// Write content
	sb.WriteString(doc.Content)
	sb.WriteString("\n")

	return sb.String()
}

// escapeYAML escapes special characters in YAML strings
func (f *DocusaurusFormatter) escapeYAML(s string) string {
	// If string contains special characters, quote it
	if strings.ContainsAny(s, ":#[]{}|>\"'") || strings.Contains(s, "\n") {
		// Escape quotes
		s = strings.ReplaceAll(s, "\"", "\\\"")
		return "\"" + s + "\""
	}
	return s
}
