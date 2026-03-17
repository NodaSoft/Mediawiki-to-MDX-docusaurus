package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/nodasoft/Mediawiki-to-MDX-docusaurus/internal/redirect"
	"github.com/nodasoft/Mediawiki-to-MDX-docusaurus/internal/wikireader"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v3"
)

const (
	defaultPort = "8080"
	defaultFile = "redirects.yaml"
)

func main() {
	// Define subcommands
	generateCmd := flag.NewFlagSet("generate", flag.ExitOnError)
	serveCmd := flag.NewFlagSet("serve", flag.ExitOnError)

	// Generate command flags
	genDBHost := generateCmd.String("db-host", envOrDefault("WIKI_DB_HOST", "localhost"), "MediaWiki database host")
	genDBPort := generateCmd.String("db-port", envOrDefault("WIKI_DB_PORT", "3306"), "MediaWiki database port")
	genDBUser := generateCmd.String("db-user", envOrDefault("WIKI_DB_USER", "root"), "MediaWiki database user")
	genDBPass := generateCmd.String("db-pass", envOrDefault("WIKI_DB_PASS", ""), "MediaWiki database password")
	genDBName := generateCmd.String("db-name", envOrDefault("WIKI_DB_NAME", "mediawiki"), "MediaWiki database name")
	genTablePrefix := generateCmd.String("table-prefix", envOrDefault("WIKI_TABLE_PREFIX", ""), "MediaWiki table prefix")
	genOutput := generateCmd.String("output", envOrDefault("REDIRECT_MAP_FILE", defaultFile), "Output YAML file for redirect map")
	genPageBaseURL := generateCmd.String("page-base-url", envOrDefault("PAGE_BASE_URL", "/docs"), "Base URL path for new Docusaurus site")
	genImageBaseURL := generateCmd.String("image-base-url", envOrDefault("IMAGE_BASE_URL", ""), "Base URL for images")
	genFileBaseURL := generateCmd.String("file-base-url", envOrDefault("FILE_BASE_URL", ""), "Base URL for files")
	genNamespace := generateCmd.String("namespace", envOrDefault("NAMESPACE", ""), "Filter by namespace (0=main, 1=talk, etc). Empty for all")
	genVerbose := generateCmd.Bool("verbose", envBoolOrDefault("VERBOSE", false), "Verbose output")

	// Serve command flags
	serveMapFile := serveCmd.String("map", envOrDefault("REDIRECT_MAP_FILE", defaultFile), "YAML file with redirect mappings")
	servePort := serveCmd.String("port", envOrDefault("PORT", defaultPort), "Port to listen on")
	serveBaseURL := serveCmd.String("base-url", envOrDefault("NEW_BASE_URL", "https://docs.example.com"), "Full base URL for new site")
	serveVerbose := serveCmd.Bool("verbose", envBoolOrDefault("VERBOSE", false), "Verbose logging")

	// Check for subcommand
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "generate":
		if err := generateCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Failed to parse generate flags: %v", err)
		}
		if *genDBPass == "" {
			log.Fatal("Database password is required (use -db-pass or WIKI_DB_PASS)")
		}
		config := redirect.Config{
			DBConfig: wikireader.DBConfig{
				DBHost:      *genDBHost,
				DBPort:      *genDBPort,
				DBUser:      *genDBUser,
				DBPass:      *genDBPass,
				DBName:      *genDBName,
				TablePrefix: *genTablePrefix,
				Verbose:     *genVerbose,
			},
			Namespace:      *genNamespace,
			OutputFile:     *genOutput,
			PageBaseURL:    *genPageBaseURL,
			ImageAssetsURL: *genImageBaseURL,
			FileAssetsURL:  *genFileBaseURL,
		}
		if err := runGenerate(config); err != nil {
			log.Fatalf("Generate failed: %v", err)
		}

	case "serve":
		if err := serveCmd.Parse(os.Args[2:]); err != nil {
			log.Fatalf("Failed to parse serve flags: %v", err)
		}
		if err := runServe(*serveMapFile, *servePort, *serveBaseURL, *serveVerbose); err != nil {
			log.Fatalf("Serve failed: %v", err)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("MediaWiki to Docusaurus Redirecter")
	fmt.Println("\nUsage:")
	fmt.Println("  redirect generate [flags]  - Generate redirect map from MediaWiki database")
	fmt.Println("  redirect serve [flags]     - Run redirect server")
	fmt.Println("\nGenerate flags:")
	fmt.Println("  -db-host string        MediaWiki database host (default: localhost)")
	fmt.Println("  -db-port string        MediaWiki database port (default: 3306)")
	fmt.Println("  -db-user string        MediaWiki database user (default: root)")
	fmt.Println("  -db-pass string        MediaWiki database password")
	fmt.Println("  -db-name string        MediaWiki database name (default: mediawiki)")
	fmt.Println("  -table-prefix string   MediaWiki table prefix")
	fmt.Println("  -output string         Output YAML file (default: redirects.yaml)")
	fmt.Println("  -page-base-url string  Base URL path for new Docusaurus site (default: /docs)")
	fmt.Println("  -image-base-url string Base URL for images")
	fmt.Println("  -file-base-url string  Base URL for files")
	fmt.Println("  -namespace string      Filter by namespace (0=main, 1=talk, etc). Empty for all")
	fmt.Println("  -verbose               Verbose output")
	fmt.Println("\nServe flags:")
	fmt.Println("  -map string            YAML file with redirect mappings (default: redirects.yaml)")
	fmt.Println("  -port string           Port to listen on (default: 8080)")
	fmt.Println("  -base-url string       Full base URL for new site (default: https://docs.example.com)")
	fmt.Println("  -verbose               Verbose logging")
	fmt.Println("\nEnvironment variables:")
	fmt.Println("  WIKI_DB_HOST, WIKI_DB_PORT, WIKI_DB_USER, WIKI_DB_PASS, WIKI_DB_NAME")
	fmt.Println("  WIKI_TABLE_PREFIX, REDIRECT_MAP_FILE, PAGE_BASE_URL, IMAGE_BASE_URL")
	fmt.Println("  FILE_BASE_URL, NAMESPACE, NEW_BASE_URL, PORT, VERBOSE")
}

func runGenerate(config redirect.Config) error {
	generator := redirect.NewGenerator(config)
	if err := generator.Run(); err != nil {
		return fmt.Errorf("failed to run generator: %w", err)
	}
	return nil
}

func runServe(mapFile, port string, baseURL string, verbose bool) error {
	fmt.Printf("Starting redirect server on port %s...\n", port)

	// Load redirect map
	redirectMap, err := loadRedirectMap(mapFile)
	if err != nil {
		return fmt.Errorf("failed to load redirect map: %w", err)
	}

	fmt.Printf("Loaded %d redirects and %d wiki redirects\n", len(redirectMap.Redirects), len(redirectMap.WikiRedirects))

	// Create Prometheus metrics
	metrics := redirect.NewMetrics()

	// Create redirect handler
	handler := redirect.NewRedirectHandler(redirectMap, baseURL, verbose, metrics)

	// Setup HTTP server
	http.HandleFunc("/", handler.ServeHTTP)

	// Add health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":         "ok",
			"redirects":      len(redirectMap.Redirects),
			"wiki_redirects": len(redirectMap.WikiRedirects),
		})
	})

	// Add Prometheus metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	fmt.Printf("Server ready at http://localhost:%s\n", port)
	fmt.Printf("Health check: http://localhost:%s/health\n", port)
	fmt.Printf("Metrics: http://localhost:%s/metrics\n", port)

	return http.ListenAndServe(":"+port, nil)
}

// loadRedirectMap loads the redirect map from a YAML file
func loadRedirectMap(filename string) (*redirect.Map, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var redirectMap redirect.Map
	if err := yaml.Unmarshal(data, &redirectMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &redirectMap, nil
}

// envOrDefault returns environment variable value or default
func envOrDefault(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	return fallback
}

// envBoolOrDefault returns environment variable as bool or default
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
