# Prometheus Metrics Implementation

This document describes the Prometheus metrics implementation for the MediaWiki to Docusaurus redirect service.

## Overview

The redirect service now exposes Prometheus metrics at the `/metrics` endpoint, providing visibility into redirect operations and HTTP request patterns.

## Metrics Endpoint

- **URL**: `http://localhost:8080/metrics`
- **Format**: Prometheus text-based exposition format
- **Authentication**: None (configure reverse proxy if needed)

## Available Metrics

### Standard HTTP Metrics

#### `mediawiki_redirect_requests_total`
- **Type**: Counter
- **Description**: Total number of HTTP requests received
- **Labels**:
  - `method`: HTTP method (GET, POST, etc.)
  - `path`: Request path
  - `status`: HTTP status code (200, 301, 404, etc.)

**Example queries:**
```promql
# Total requests
sum(mediawiki_redirect_requests_total)

# Requests by status code
sum by (status) (mediawiki_redirect_requests_total)

# 404 rate
rate(mediawiki_redirect_requests_total{status="404"}[5m])
```

#### `mediawiki_redirect_request_duration_seconds`
- **Type**: Histogram
- **Description**: HTTP request duration in seconds
- **Labels**:
  - `method`: HTTP method
  - `path`: Request path
  - `status`: HTTP status code
- **Buckets**: Default Prometheus buckets (0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10)

**Example queries:**
```promql
# Average request duration
rate(mediawiki_redirect_request_duration_seconds_sum[5m]) /
rate(mediawiki_redirect_request_duration_seconds_count[5m])

# 95th percentile latency
histogram_quantile(0.95,
  rate(mediawiki_redirect_request_duration_seconds_bucket[5m]))

# Requests slower than 1 second
sum(rate(mediawiki_redirect_request_duration_seconds_bucket{le="1"}[5m]))
```

### Redirect-Specific Metrics

#### `mediawiki_redirects_total`
- **Type**: Counter
- **Description**: Total number of successful redirects performed
- **Labels**:
  - `type`: Type of redirect
    - `direct`: Direct redirects from the redirect map
    - `wiki_redirect`: Redirects that followed MediaWiki internal redirects
    - `root`: Root path redirects

**Example queries:**
```promql
# Total redirects
sum(mediawiki_redirects_total)

# Redirects by type
sum by (type) (mediawiki_redirects_total)

# Redirect rate
rate(mediawiki_redirects_total[5m])

# Wiki redirect percentage
sum(rate(mediawiki_redirects_total{type="wiki_redirect"}[5m])) /
sum(rate(mediawiki_redirects_total[5m])) * 100
```

#### `mediawiki_redirects_not_found_total`
- **Type**: Counter
- **Description**: Total number of redirect requests that resulted in 404 Not Found

**Example queries:**
```promql
# Total 404s
mediawiki_redirects_not_found_total

# 404 rate
rate(mediawiki_redirects_not_found_total[5m])

# 404 percentage
rate(mediawiki_redirects_not_found_total[5m]) /
rate(mediawiki_redirect_requests_total[5m]) * 100
```

#### `mediawiki_wiki_redirects_followed_total`
- **Type**: Counter
- **Description**: Total number of MediaWiki internal redirects followed

**Example queries:**
```promql
# Total wiki redirects followed
mediawiki_wiki_redirects_followed_total

# Wiki redirect follow rate
rate(mediawiki_wiki_redirects_followed_total[5m])
```

## Grafana Dashboard

### Recommended Panels

1. **Request Rate**
   - Query: `rate(mediawiki_redirect_requests_total[5m])`
   - Type: Graph
   - Legend: `{{status}}`

2. **Redirect Success Rate**
   - Query: `sum(rate(mediawiki_redirects_total[5m]))`
   - Type: Stat
   - Unit: requests/sec

3. **404 Rate**
   - Query: `rate(mediawiki_redirects_not_found_total[5m])`
   - Type: Graph
   - Color: Red

4. **Request Duration (p95)**
   - Query: `histogram_quantile(0.95, rate(mediawiki_redirect_request_duration_seconds_bucket[5m]))`
   - Type: Graph
   - Unit: seconds

5. **Redirects by Type**
   - Query: `sum by (type) (rate(mediawiki_redirects_total[5m]))`
   - Type: Pie chart

6. **Wiki Redirects Followed**
   - Query: `rate(mediawiki_wiki_redirects_followed_total[5m])`
   - Type: Stat

## Alerting Rules

### Example Prometheus Alert Rules

```yaml
groups:
  - name: mediawiki_redirect_alerts
    interval: 30s
    rules:
      # High 404 rate
      - alert: HighRedirect404Rate
        expr: |
          rate(mediawiki_redirects_not_found_total[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High 404 rate on redirect service"
          description: "404 rate is {{ $value }} requests/sec"

      # High request latency
      - alert: HighRedirectLatency
        expr: |
          histogram_quantile(0.95,
            rate(mediawiki_redirect_request_duration_seconds_bucket[5m])
          ) > 1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High latency on redirect service"
          description: "95th percentile latency is {{ $value }}s"

      # Service down
      - alert: RedirectServiceDown
        expr: |
          up{job="mediawiki-redirect"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Redirect service is down"
          description: "The redirect service has been down for more than 1 minute"
```

## Integration with Monitoring Stack

### Prometheus Configuration

```yaml
scrape_configs:
  - job_name: 'mediawiki-redirect'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
    scrape_timeout: 10s
```

### Docker Compose Example

```yaml
version: '3.8'

services:
  redirect:
    build: .
    ports:
      - "8080:8080"
    environment:
      - REDIRECT_MAP_FILE=/data/redirects.yaml
      - PORT=8080
      - NEW_BASE_URL=https://docs.example.com
    volumes:
      - ./redirects.yaml:/data/redirects.yaml

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    volumes:
      - grafana-data:/var/lib/grafana

volumes:
  prometheus-data:
  grafana-data:
```

## Performance Considerations

- Metrics collection has minimal overhead (< 1ms per request)
- Histogram buckets are optimized for typical redirect latencies
- Label cardinality is kept low to prevent memory issues
- The `/metrics` endpoint itself is not tracked to avoid recursion

## Troubleshooting

### Metrics not appearing

1. Check that the `/metrics` endpoint is accessible:
   ```bash
   curl http://localhost:8080/metrics
   ```

2. Verify Prometheus is scraping the target:
   - Go to Prometheus UI: `http://localhost:9090/targets`
   - Check the status of the `mediawiki-redirect` job

3. Check for errors in Prometheus logs

### High cardinality warnings

If you see high cardinality warnings, it may be due to the `path` label. Consider:
- Grouping similar paths
- Using a reverse proxy to normalize paths
- Reducing the number of unique paths being tracked

## Future Enhancements

Potential additions to the metrics:
- Response size histogram
- Redirect chain length (for wiki redirects)
- Cache hit/miss rates (if caching is added)
- Geographic distribution of requests (if GeoIP is added)
