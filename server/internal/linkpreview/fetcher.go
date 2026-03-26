package linkpreview

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const (
	maxBodySize    = 1 << 20 // 1 MB
	fetchTimeout   = 5 * time.Second
	maxRedirects   = 3
	userAgentValue = "Enzymebot/1.0 (+https://github.com/nicholasgriffintn/enzyme)"
)

// bracketLinkPattern matches <url|label> markup from the rich text editor.
var bracketLinkPattern = regexp.MustCompile(`<(https?://[^|>]+)\|[^>]*>`)

// urlPattern matches http(s) URLs in message text.
var urlPattern = regexp.MustCompile(`https?://[^\s<>"'\)\]]+`)

// ExtractFirstURL returns the first external URL in content, or "".
func ExtractFirstURL(content string) string {
	// First try bracket-link syntax: <url|label>
	if m := bracketLinkPattern.FindStringSubmatch(content); len(m) > 1 {
		u := m[1]
		if !strings.Contains(u, "/api/") {
			return u
		}
	}

	// Fall back to plain URL matching.
	matches := urlPattern.FindAllString(content, -1)
	for _, u := range matches {
		// Strip trailing punctuation that is not part of URLs.
		u = strings.TrimRight(u, ".,;:!?")
		// Skip internal API paths.
		if strings.Contains(u, "/api/") {
			continue
		}
		return u
	}
	return ""
}

// ogData holds parsed Open Graph metadata.
type ogData struct {
	Title       string
	Description string
	ImageURL    string
	SiteName    string
}

// Fetcher fetches and caches link previews.
type Fetcher struct {
	repo   *Repository
	client *http.Client
}

// NewFetcher creates a Fetcher with an SSRF-safe HTTP client.
func NewFetcher(repo *Repository) *Fetcher {
	return NewFetcherWithClient(repo, nil)
}

// NewFetcherWithClient creates a Fetcher with a custom HTTP client.
// If client is nil, a default SSRF-safe client is used.
func NewFetcherWithClient(repo *Repository, client *http.Client) *Fetcher {
	if client == nil {
		transport := &http.Transport{
			DialContext: safeDialContext,
		}

		client = &http.Client{
			Timeout:   fetchTimeout,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= maxRedirects {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		}
	}

	return &Fetcher{repo: repo, client: client}
}

// FetchPreview returns a Preview for the URL, using the cache when possible.
// Returns nil if the URL could not be fetched or has no useful OG data.
func (f *Fetcher) FetchPreview(ctx context.Context, url string) (*Preview, error) {
	// Check cache.
	cached, err := f.repo.GetCachedURL(ctx, url)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		if cached.FetchError != "" {
			return nil, nil // cached error — skip
		}
		return &Preview{
			URL:         cached.URL,
			Title:       cached.Title,
			Description: cached.Description,
			ImageURL:    cached.ImageURL,
			SiteName:    cached.SiteName,
		}, nil
	}

	// Fetch OG data.
	og, fetchErr := f.fetchOG(ctx, url)

	now := time.Now().UTC()
	entry := &CacheEntry{
		URL:       url,
		FetchedAt: now,
	}

	if fetchErr != nil {
		entry.FetchError = fetchErr.Error()
		entry.ExpiresAt = now.Add(ErrorCacheTTL)
		_ = f.repo.SetCachedURL(ctx, entry)
		return nil, nil
	}

	if og == nil || (og.Title == "" && og.Description == "") {
		// No useful data — cache as error to avoid retrying.
		entry.FetchError = "no og data"
		entry.ExpiresAt = now.Add(ErrorCacheTTL)
		_ = f.repo.SetCachedURL(ctx, entry)
		return nil, nil
	}

	entry.Title = og.Title
	entry.Description = og.Description
	entry.ImageURL = og.ImageURL
	entry.SiteName = og.SiteName
	entry.ExpiresAt = now.Add(CacheTTL)
	if err := f.repo.SetCachedURL(ctx, entry); err != nil {
		return nil, err
	}

	return &Preview{
		URL:         url,
		Title:       og.Title,
		Description: og.Description,
		ImageURL:    og.ImageURL,
		SiteName:    og.SiteName,
	}, nil
}

// fetchOG performs an HTTP GET and parses OG meta tags.
func (f *Fetcher) fetchOG(ctx context.Context, url string) (*ogData, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgentValue)
	req.Header.Set("Accept", "text/html")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		return nil, nil
	}

	body := io.LimitReader(resp.Body, maxBodySize)
	return parseOG(body)
}

// parseOG extracts og:* meta tags and falls back to <title> / <meta name="description">.
func parseOG(r io.Reader) (*ogData, error) {
	tokenizer := html.NewTokenizer(r)
	data := &ogData{}
	var fallbackTitle string
	var fallbackDesc string

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			err := tokenizer.Err()
			if err == io.EOF {
				applyFallbacks(data, fallbackTitle, fallbackDesc)
				return data, nil
			}
			applyFallbacks(data, fallbackTitle, fallbackDesc)
			return data, nil

		case html.StartTagToken, html.SelfClosingTagToken:
			tn, hasAttr := tokenizer.TagName()
			tag := string(tn)

			if tag == "body" {
				// Stop parsing at <body>.
				applyFallbacks(data, fallbackTitle, fallbackDesc)
				return data, nil
			}

			if tag == "title" && fallbackTitle == "" {
				// Read title text content.
				if tokenizer.Next() == html.TextToken {
					fallbackTitle = strings.TrimSpace(string(tokenizer.Text()))
				}
				continue
			}

			if tag == "meta" && hasAttr {
				attrs := readAttrs(tokenizer)
				prop := attrs["property"]
				name := attrs["name"]
				content := attrs["content"]

				switch prop {
				case "og:title":
					data.Title = content
				case "og:description":
					data.Description = content
				case "og:image":
					data.ImageURL = content
				case "og:site_name":
					data.SiteName = content
				}

				if name == "description" && fallbackDesc == "" {
					fallbackDesc = content
				}
			}
		}
	}
}

func applyFallbacks(data *ogData, fallbackTitle, fallbackDesc string) {
	if data.Title == "" {
		data.Title = fallbackTitle
	}
	if data.Description == "" {
		data.Description = fallbackDesc
	}
}

// readAttrs collects all attributes from the current tag token.
func readAttrs(z *html.Tokenizer) map[string]string {
	attrs := make(map[string]string)
	for {
		key, val, more := z.TagAttr()
		k := string(key)
		if k != "" {
			attrs[k] = string(val)
		}
		if !more {
			break
		}
	}
	return attrs
}

// privateRanges are CIDR blocks for private / loopback IPs.
var privateRanges []*net.IPNet

func init() {
	for _, cidr := range []string{
		"127.0.0.0/8",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	} {
		_, block, _ := net.ParseCIDR(cidr)
		privateRanges = append(privateRanges, block)
	}
}

func isPrivateIP(ip net.IP) bool {
	for _, block := range privateRanges {
		if block.Contains(ip) {
			return true
		}
	}
	return false
}

// safeDialContext resolves DNS then rejects private IPs before connecting.
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}

	for _, ip := range ips {
		if isPrivateIP(ip.IP) {
			return nil, fmt.Errorf("connection to private IP %s is not allowed", ip.IP)
		}
	}

	// Connect to the first resolved IP.
	dialer := &net.Dialer{Timeout: fetchTimeout}
	return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
}
