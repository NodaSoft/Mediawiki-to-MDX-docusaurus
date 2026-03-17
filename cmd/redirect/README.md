# MediaWiki to Docusaurus Redirecter

A tool to help migrate from MediaWiki to Docusaurus by providing URL redirection capabilities.

## Features

- **Two modes of operation:**
  1. **Generate mode**: Creates a redirect map from MediaWiki database
  2. **Serve mode**: Runs a web server that redirects old MediaWiki URLs to new Docusaurus URLs

- **Supports MediaWiki redirects**: Handles internal MediaWiki page redirects
- **Multiple URL formats**: Supports various MediaWiki URL patterns:
  - `/Page_Title`
  - `/index.php?title=Page_Title`
  - `/index.php/Page_Title`

## Installation

Build the binary:

```bash
go build -o redirect ./cmd/redirect
```

## Usage

### Generate Mode

Generate a redirect map from your MediaWiki database:

```bash
./redirect generate \
  -db-host localhost \
  -db-port 3306 \
  -db-user root \
  -db-pass yourpassword \
  -db-name mediawiki \
  -output redirects.yaml \
  -page-base-url https://docs.example.com/docs \
  -image-base-url https://wiki.example.com/images \
  -file-base-url https://wiki.example.com/files \
  -verbose
```

**Flags:**
- `-db-host`: MediaWiki database host (default: localhost)
- `-db-port`: MediaWiki database port (default: 3306)
- `-db-user`: MediaWiki database user (default: root)
- `-db-pass`: MediaWiki database password (required)
- `-db-name`: MediaWiki database name (default: mediawiki)
- `-table-prefix`: MediaWiki table prefix (if any)
- `-output`: Output YAML file for redirect map (default: redirects.yaml)
- `-page-base-url`: Base URL path for new Docusaurus site (default: /docs)
- `-image-base-url`: Base URL for images
- `-file-base-url`: Base URL for files
- `-verbose`: Enable verbose output

**Environment Variables:**

You can also use environment variables instead of flags:
- `WIKI_DB_HOST`
- `WIKI_DB_PORT`
- `WIKI_DB_USER`
- `WIKI_DB_PASS`
- `WIKI_DB_NAME`
- `WIKI_TABLE_PREFIX`
- `REDIRECT_MAP_FILE`
- `PAGE_BASE_URL`
- `IMAGE_BASE_URL`
- `FILE_BASE_URL`
- `VERBOSE`

### Serve Mode

Run the redirect server using the generated redirect map:

```bash
./redirect serve \
  -map redirects.yaml \
  -port 8080 \
  -new-base-url https://docs.example.com \
  -verbose
```

**Flags:**
- `-map`: YAML file with redirect mappings (default: redirects.yaml)
- `-port`: Port to listen on (default: 8080)
- `-new-base-url`: Full base URL for new Docusaurus site (default: https://docs.example.com)
- `-verbose`: Enable verbose logging

**Environment Variables:**
- `REDIRECT_MAP_FILE`
- `PORT`
- `NEW_BASE_URL`
- `VERBOSE`

## Example Workflow

1. **Generate the redirect map** from your MediaWiki database:

```bash
./redirect generate \
  -db-pass mypassword \
  -new-base-url https://newdocs.example.com/docs \
  -output redirects.yaml
```

2. **Deploy the redirect server** on your old MediaWiki domain:

```bash
./redirect serve \
  -map redirects.yaml \
  -port 80 \
  -new-base-url https://newdocs.example.com
```

3. **Configure your web server** (nginx/apache) to proxy requests to the redirect server, or run it directly.

## Redirect Map Format

The generated YAML file contains two sections:

```yaml
redirects:
  /wiki/Main_Page: https://newdocs.example.com/docs/main-page
  /wiki/Help:Getting_Started: https://newdocs.example.com/docs/help/getting-started
  # ... more redirects

wiki_redirects:
  Old Page Name: New Page Name
  Deprecated Article: Current Article
  # ... MediaWiki internal redirects
```

- **redirects**: Maps old MediaWiki URLs to new Docusaurus URLs
- **wiki_redirects**: Maps MediaWiki redirect pages to their target pages

## Health Check

The server provides a health check endpoint at `/health`:

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "ok",
  "redirects": 1234,
  "wiki_redirects": 56
}
```

## Prometheus Metrics

The server exposes Prometheus metrics at `/metrics`:

```bash
curl http://localhost:8080/metrics
```

### Available Metrics

**Standard HTTP Metrics:**
- `mediawiki_redirect_requests_total` - Total number of HTTP requests received (labels: method, path, status)
- `mediawiki_redirect_request_duration_seconds` - HTTP request duration in seconds (labels: method, path, status)

**Redirect-Specific Metrics:**
- `mediawiki_redirects_total` - Total number of successful redirects performed (labels: type)
  - `type="direct"` - Direct redirects from the redirect map
  - `type="wiki_redirect"` - Redirects that followed MediaWiki internal redirects
  - `type="root"` - Root path redirects
- `mediawiki_redirects_not_found_total` - Total number of redirect requests that resulted in 404 Not Found
- `mediawiki_wiki_redirects_followed_total` - Total number of MediaWiki internal redirects followed

### Example Prometheus Configuration

Add this to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'mediawiki-redirect'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

## Deployment Examples

### Using systemd

Create a systemd service file `/etc/systemd/system/mediawiki-redirect.service`:

```ini
[Unit]
Description=MediaWiki to Docusaurus Redirecter
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/redirect
ExecStart=/opt/redirect/redirect serve -map /opt/redirect/redirects.yaml -port 8080 -new-base-url https://newdocs.example.com
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable mediawiki-redirect
sudo systemctl start mediawiki-redirect
```

### Using nginx as reverse proxy

```nginx
server {
    listen 80;
    server_name oldwiki.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Using Docker

Create a `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o redirect ./cmd/redirect

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/redirect .
COPY redirects.yaml .
EXPOSE 8080
CMD ["./redirect", "serve", "-map", "redirects.yaml", "-port", "8080", "-new-base-url", "https://newdocs.example.com"]
```

Build and run:
```bash
docker build -t mediawiki-redirect .
docker run -p 8080:8080 mediawiki-redirect
```

## How It Works

1. **Generate Mode**:
   - Connects to MediaWiki database
   - Fetches all pages with their namespaces
   - Converts page titles to both old MediaWiki URLs and new Docusaurus URLs
   - Identifies MediaWiki redirect pages
   - Saves mappings to YAML file

2. **Serve Mode**:
   - Loads redirect mappings from YAML file
   - Listens for HTTP requests
   - Parses incoming MediaWiki URLs
   - Looks up redirect target
   - Follows MediaWiki redirect chains if needed
   - Returns 301 permanent redirect to new URL

## URL Conversion

The tool uses the same conversion logic as the main converter to ensure consistency:

- **MediaWiki URL**: `/Page_Title` or `/index.php?title=Page_Title`
- **Docusaurus URL**: `/docs/page-title`

Features:
- Cyrillic transliteration
- Namespace handling (Help, Template, Category, etc.)
- Special character normalization
- Consistent slug generation

## Notes

- The server returns **301 Permanent Redirect** status codes, which tells search engines to update their indexes
- MediaWiki redirect chains are followed automatically
- The health check endpoint can be used for monitoring and load balancer health checks
- Verbose mode logs all requests and redirects for debugging
