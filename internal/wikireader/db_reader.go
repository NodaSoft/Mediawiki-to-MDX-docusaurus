package wikireader

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/nodasoft/Mediawiki-to-MDX-docusaurus/internal/config"
)

// WikiDBReader reads pages from a MediaWiki database
type WikiDBReader struct {
	db     *sql.DB
	config config.Config
}

// NewWikiDBReader creates a new WikiDBReader and connects to the database
func NewWikiDBReader(config config.Config) (*WikiDBReader, error) {
	if !isValidTablePrefix(config.TablePrefix) {
		return nil, fmt.Errorf("invalid table prefix %q: only letters, numbers, and underscore are allowed", config.TablePrefix)
	}

	// Connect to database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true",
		config.DBUser, config.DBPass, config.DBHost, config.DBPort, config.DBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if config.Verbose {
		log.Printf("Connected to MediaWiki database: %s@%s:%s/%s",
			config.DBUser, config.DBHost, config.DBPort, config.DBName)
	}

	return &WikiDBReader{
		db:     db,
		config: config,
	}, nil
}

// tableName returns the full table name with prefix
func (c *WikiDBReader) tableName(name string) string {
	return c.config.TablePrefix + name
}

// FetchPages retrieves pages from MediaWiki database
func (c *WikiDBReader) FetchPages() ([]WikiPage, error) {
	query := `
		SELECT
			p.page_id,
			p.page_namespace,
			p.page_title,
			p.page_is_redirect,
			COALESCE(t.old_text, '') AS old_text,
			r.rev_timestamp
		FROM ` + c.tableName("page") + ` p
		JOIN ` + c.tableName("revision") + ` r ON p.page_latest = r.rev_id
		JOIN ` + c.tableName("slots") + ` s ON r.rev_id = s.slot_revision_id
		JOIN ` + c.tableName("content") + ` c ON s.slot_content_id = c.content_id
		LEFT JOIN ` + c.tableName("text") + ` t ON t.old_id = CAST(SUBSTRING(c.content_address, 4) AS UNSIGNED)
		WHERE s.slot_role_id = 1 AND c.content_address LIKE 'tt:%'
	`

	args := []interface{}{}

	// Filter by namespace if specified
	if c.config.Namespace != "" {
		query += " AND p.page_namespace = ?"
		args = append(args, c.config.Namespace)
	}

	query += " ORDER BY p.page_namespace, p.page_title"

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	var pages []WikiPage
	for rows.Next() {
		var page WikiPage
		var isRedirect int
		err := rows.Scan(
			&page.ID,
			&page.Namespace,
			&page.Title,
			&isRedirect,
			&page.Content,
			&page.Timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		page.IsRedirect = isRedirect == 1
		// Replace underscores with spaces in title
		page.Title = strings.ReplaceAll(page.Title, "_", " ")
		pages = append(pages, page)
	}

	return pages, rows.Err()
}

// Close closes the database connection
func (c *WikiDBReader) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

// isValidTablePrefix validates that a table prefix only contains safe characters
func isValidTablePrefix(prefix string) bool {
	if prefix == "" {
		return true
	}
	return regexp.MustCompile(`^[a-zA-Z0-9_]+$`).MatchString(prefix)
}
