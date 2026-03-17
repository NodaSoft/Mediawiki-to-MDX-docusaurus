# MediaWiki to MDX/Docusaurus Converter

A Go-based tool to convert MediaWiki content to MDX format for use with Docusaurus. This tool connects directly to a MediaWiki database and exports articles as MDX files, optionally downloading associated assets (images and files).

## Features

- 📄 Converts MediaWiki articles to MDX format compatible with Docusaurus
- 🖼️ Optional asset downloading (images and files)
- 🗂️ Namespace filtering support
- 🔧 Configurable via command-line flags or environment variables
- 📊 Detailed conversion statistics
- 🚀 Direct database access for efficient conversion
- 🔀 **URL Redirecter** - Redirect old MediaWiki URLs to new Docusaurus URLs

## Prerequisites

- Go 1.25.7 or higher
- Access to a MediaWiki MySQL/MariaDB database
- (Optional) Access to MediaWiki asset URLs for downloading images and files

## Installation

### From Source

```bash
git clone https://github.com/nodasoft/Mediawiki-to-MDX-docusaurus.git
cd Mediawiki-to-MDX-docusaurus

# Build the converter
go build -o bin/wikiToMdx ./cmd/converter

# Build the redirecter
go build -o bin/redirect ./cmd/redirect
```

### Using Make

```bash
make build
```

## Usage

### Basic Usage

```bash
./wikiToMdx \
  -db-host localhost \
  -db-port 3306 \
  -db-user root \
  -db-pass your_password \
  -db-name mediawiki \
  -output ./docs
```

### With Asset Download

```bash
./wikiToMdx \
  -db-host localhost \
  -db-pass your_password \
  -db-name mediawiki \
  -output ./docs \
  -download-assets \
  -asset-url https://wiki.example.com/images \
  -images-dir ./static/img/wiki \
  -files-dir ./static/files/wiki
```

### Using Environment Variables

Create a `.env` file or export environment variables:

```bash
export WIKI_DB_HOST=localhost
export WIKI_DB_PORT=3306
export WIKI_DB_USER=root
export WIKI_DB_PASS=your_password
export WIKI_DB_NAME=mediawiki
export OUTPUT_DIR=./docs
export DOWNLOAD_ASSETS=true
export ASSET_BASE_URL=https://wiki.example.com/images
export VERBOSE=true

./bin/wikiToMdx
```

## Configuration Options

| Flag | Environment Variable | Default | Description |
|------|---------------------|---------|-------------|
| `-db-host` | `WIKI_DB_HOST` | `localhost` | MediaWiki database host |
| `-db-port` | `WIKI_DB_PORT` | `3306` | MediaWiki database port |
| `-db-user` | `WIKI_DB_USER` | `root` | MediaWiki database user |
| `-db-pass` | `WIKI_DB_PASS` | *(required)* | MediaWiki database password |
| `-db-name` | `WIKI_DB_NAME` | `mediawiki` | MediaWiki database name |
| `-table-prefix` | `WIKI_TABLE_PREFIX` | *(empty)* | MediaWiki table prefix (e.g., `mw_`) |
| `-output` | `OUTPUT_DIR` | `./docs` | Output directory for Docusaurus docs |
| `-images-dir` | `IMAGES_DIR` | `./static/img/wiki` | Directory for downloaded images |
| `-files-dir` | `FILES_DIR` | `./static/files/wiki` | Directory for downloaded files |
| `-namespace` | `NAMESPACE` | *(empty)* | Filter by namespace (0=main, 1=talk, etc.) |
| `-asset-url` | `ASSET_BASE_URL` | *(empty)* | Base URL for assets |
| `-download-assets` | `DOWNLOAD_ASSETS` | `false` | Download wiki assets |
| `-image-url` | `IMAGE_BASE_URL` | *(empty)* | Base URL for images |
| `-file-url` | `FILE_BASE_URL` | *(empty)* | Base URL for files |
| `-verbose` | `VERBOSE` | `false` | Verbose output |

## Development

### Running the Linter

```bash
# Run all linters
make lint

# Run individual linters
make fmt
make vet
make golangci-lint

# Install golangci-lint if not already installed
make install-linter
```

### Project Structure

```
.
├── cmd/
│   ├── converter/           # Converter tool
│   │   └── main.go
│   └── redirect/            # Redirecter tool
│       ├── main.go
│       ├── README.md
│       └── redirects.example.yaml
├── internal/
│   └── wikiconverter/       # Core conversion logic
│       ├── converter.go     # Main converter
│       ├── parser.go        # Wiki markup parser
│       ├── formatter.go     # MDX formatter
│       ├── downloader.go    # Asset downloader
│       ├── wikireader.go    # Database reader
│       ├── helpers.go       # Helper functions
│       └── table_parser.go  # Table parser
├── Makefile                 # Build and lint commands
├── go.mod                   # Go module definition
└── README.md               # This file
```

## URL Redirecter

After converting your MediaWiki content to Docusaurus, you'll want to redirect old URLs to the new ones. The included redirecter tool helps with this:

### Generate Redirect Map

```bash
./redirect generate \
  -db-pass your_password \
  -new-base-url https://newdocs.example.com/docs \
  -output redirects.yaml
```

### Run Redirect Server

Deploy this on your old MediaWiki domain to redirect traffic:

```bash
./redirect serve \
  -map redirects.yaml \
  -port 80 \
  -new-base-url https://newdocs.example.com
```

For detailed documentation, see [cmd/redirect/README.md](cmd/redirect/README.md).

## Output

The converter generates:

- **MDX files**: One file per MediaWiki article in the output directory
- **Images**: Downloaded to the images directory (if `-download-assets` is enabled)
- **Files**: Downloaded to the files directory (if `-download-assets` is enabled)
- **Statistics**: Summary of conversion results

The redirecter generates:

- **YAML redirect map**: Mapping of old URLs to new URLs
- **HTTP redirect server**: 301 redirects for old MediaWiki URLs

### Example Output

```
Starting MediaWiki to Docusaurus conversion...

=== Conversion Complete ===
Total articles processed: 150
Successfully converted: 145
Skipped: 3
Failed: 2
Assets downloaded: 87
Asset download failed: 5
Images directory: ./static/img/wiki
Files directory: ./static/files/wiki
Output directory: ./docs
```

## MediaWiki Namespaces

Common namespace IDs for filtering:

- `0` - Main/Article
- `1` - Talk
- `2` - User
- `3` - User talk
- `4` - Project
- `6` - File
- `10` - Template
- `14` - Category

## Troubleshooting

### Database Connection Issues

- Ensure the database credentials are correct
- Verify the database host is accessible
- Check if the database user has read permissions

### Asset Download Issues

- Verify the asset URL is correct and accessible
- Ensure you have write permissions to the output directories
- Check network connectivity to the MediaWiki server

### Conversion Errors

- Use `-verbose` flag for detailed logging
- Check the MediaWiki database schema matches expectations
- Verify the table prefix is correct if your MediaWiki uses one

## Migration Workflow

1. **Convert your MediaWiki content**:
   ```bash
   ./wikiToMdx -db-pass password -output ./docs -download-assets
   ```

2. **Generate redirect map**:
   ```bash
   ./redirect generate -db-pass password -new-base-url https://newdocs.example.com/docs
   ```

3. **Deploy your Docusaurus site** with the converted content

4. **Deploy the redirect server** on your old MediaWiki domain:
   ```bash
   ./redirect serve -map redirects.yaml -new-base-url https://newdocs.example.com
   ```

5. **Update DNS** (optional) to point old domain to redirect server

## TODO

- [ ] Read wiki articles by HTTP (API-based access as alternative to direct database connection)
- [x] Support redirect pages (implemented in redirecter tool)
- [ ] Tests for edge cases and error handling
- [ ] Interfaces for better extensibility

## License

This project is licensed under the MIT License with Non-Commercial Clause - see the [LICENSE](LICENSE) file for details.

**Summary**: You are free to use, modify, and distribute this software for non-commercial purposes. Commercial use (selling or incorporating into commercial products) requires explicit written permission.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues and questions, please open an issue on the GitHub repository.
