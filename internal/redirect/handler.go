package redirect

import (
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// RedirectHandler handles HTTP redirects
type RedirectHandler struct {
	redirectMap *Map
	newBaseURL  string
	verbose     bool
	metrics     *Metrics
}

func NewRedirectHandler(redirectMap *Map, newBaseURL string, verbose bool, metrics *Metrics) *RedirectHandler {
	return &RedirectHandler{
		redirectMap: redirectMap,
		newBaseURL:  newBaseURL,
		verbose:     verbose,
		metrics:     metrics,
	}
}

func (h *RedirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Track request start time for duration metric
	startTime := time.Now()

	if h.verbose {
		log.Printf("Request: %s %s", r.Method, r.URL.Path)
	}

	// Parse the request URL
	requestPath := r.URL.Path
	queryParams := r.URL.Query()

	// Try to find redirect target
	var targetURL string
	var found bool
	var redirectType string

	// MediaWiki URLs can be in different formats:
	// 1. /wiki/Page_Title
	// 2. /index.php?title=Page_Title
	// 3. /index.php/Page_Title

	if strings.HasPrefix(requestPath, "/index.php") {
		// Check for ?title= parameter
		if title := queryParams.Get("title"); title != "" {
			targetURL, found, redirectType = h.findRedirect(title)
		} else {
			// Format: /index.php/Page_Title
			pageName := strings.TrimPrefix(requestPath, "/index.php/")
			if pageName != "" {
				targetURL, found, redirectType = h.findRedirect(pageName)
			}
		}
	} else if requestPath == "/" || requestPath == "" {
		// Redirect root to new base URL
		targetURL = h.newBaseURL
		found = true
		redirectType = "root"
	} else {
		// Try direct path match
		targetURL, found, redirectType = h.findRedirect(strings.TrimPrefix(requestPath, "/"))
	}

	if !found {
		if h.verbose {
			log.Printf("  No redirect found for: %s", requestPath)
		}

		// Track 404 metrics
		statusCode := http.StatusNotFound
		h.metrics.RedirectsNotFound.Inc()
		h.metrics.RequestsTotal.WithLabelValues(r.Method, requestPath, strconv.Itoa(statusCode)).Inc()
		h.metrics.RequestDuration.WithLabelValues(r.Method, requestPath, strconv.Itoa(statusCode)).Observe(time.Since(startTime).Seconds())

		http.NotFound(w, r)
		return
	}

	// Ensure target URL is absolute
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = h.newBaseURL + "/" + strings.TrimPrefix(targetURL, "/")
	}

	if h.verbose {
		log.Printf("  Redirecting to: %s", targetURL)
	}

	// Track successful redirect metrics
	statusCode := http.StatusFound
	h.metrics.RedirectsTotal.WithLabelValues(redirectType).Inc()
	h.metrics.RequestsTotal.WithLabelValues(r.Method, requestPath, strconv.Itoa(statusCode)).Inc()
	h.metrics.RequestDuration.WithLabelValues(r.Method, requestPath, strconv.Itoa(statusCode)).Observe(time.Since(startTime).Seconds())

	// Perform 302 temporary redirect
	http.Redirect(w, r, targetURL, http.StatusFound)
}

func (h *RedirectHandler) findRedirect(pageName string) (string, bool, string) {
	// Decode URL encoding
	decoded, err := url.QueryUnescape(pageName)
	if err == nil {
		pageName = decoded
	}

	// Replace underscores with spaces (MediaWiki convention)
	pageName = strings.ReplaceAll(pageName, "_", " ")

	// Try direct lookup in redirect map
	if target, ok := h.redirectMap.Redirects[strings.ReplaceAll(pageName, " ", "_")]; ok {
		return target, true, "direct"
	}

	// Check if this is a wiki redirect (page that redirects to another page)
	if redirectTarget, ok := h.redirectMap.WikiRedirects[pageName]; ok {
		// Follow the redirect chain
		if h.verbose {
			log.Printf("  Following wiki redirect: %s -> %s", pageName, redirectTarget)
		}
		// Track wiki redirect followed
		h.metrics.WikiRedirectsFollowed.Inc()

		// Convert the redirect target to new URL
		targetURL := "" //convertToDocusaurusURL(redirectTarget, 0, h.newBaseURL)
		return targetURL, true, "wiki_redirect"
	}

	// Try converting the page name to new URL format
	targetURL := "" //convertToDocusaurusURL(pageName, 0, h.newBaseURL)
	return targetURL, true, "direct"
}
