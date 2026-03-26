package linkpreview

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/enzyme/server/internal/testutil"
)

func TestExtractFirstURL(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"no url", "hello world", ""},
		{"simple url", "check https://example.com for details", "https://example.com"},
		{"http url", "see http://example.com", "http://example.com"},
		{"url with path", "go to https://example.com/page?q=1", "https://example.com/page?q=1"},
		{"skip internal api", "see /api/files or https://api.example.com/api/test then https://real.com", "https://real.com"},
		{"first url wins", "https://first.com and https://second.com", "https://first.com"},
		{"url with trailing punctuation", "Visit https://example.com.", "https://example.com"},
		{"url in parens", "(https://example.com)", "https://example.com"},
		{"only internal url", "https://myapp.com/api/files/123", ""},
		{"bracket link", "<https://github.com/org/repo|https://github.com/org/repo>", "https://github.com/org/repo"},
		{"bracket link with label", "<https://example.com/page|Example Page>", "https://example.com/page"},
		{"bracket link internal", "<https://app.com/api/files/1|file>", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractFirstURL(tt.content)
			if got != tt.want {
				t.Errorf("ExtractFirstURL(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestFetchOG_FullMeta(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head>
			<meta property="og:title" content="Test Title">
			<meta property="og:description" content="Test Description">
			<meta property="og:image" content="https://example.com/img.png">
			<meta property="og:site_name" content="TestSite">
		</head><body></body></html>`)
	}))
	defer srv.Close()

	db := testutil.TestDB(t)
	repo := NewRepository(db)
	f := NewFetcherWithClient(repo, &http.Client{Timeout: fetchTimeout})

	preview, err := f.FetchPreview(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchPreview: %v", err)
	}
	if preview == nil {
		t.Fatal("expected preview, got nil")
	}
	if preview.Title != "Test Title" {
		t.Errorf("title = %q, want %q", preview.Title, "Test Title")
	}
	if preview.Description != "Test Description" {
		t.Errorf("description = %q, want %q", preview.Description, "Test Description")
	}
	if preview.ImageURL != "https://example.com/img.png" {
		t.Errorf("image_url = %q, want %q", preview.ImageURL, "https://example.com/img.png")
	}
	if preview.SiteName != "TestSite" {
		t.Errorf("site_name = %q, want %q", preview.SiteName, "TestSite")
	}
}

func TestFetchOG_FallbackTitle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head>
			<title>Fallback Title</title>
			<meta name="description" content="Fallback Description">
		</head><body></body></html>`)
	}))
	defer srv.Close()

	db := testutil.TestDB(t)
	repo := NewRepository(db)
	f := NewFetcherWithClient(repo, &http.Client{Timeout: fetchTimeout})

	preview, err := f.FetchPreview(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchPreview: %v", err)
	}
	if preview == nil {
		t.Fatal("expected preview, got nil")
	}
	if preview.Title != "Fallback Title" {
		t.Errorf("title = %q, want %q", preview.Title, "Fallback Title")
	}
	if preview.Description != "Fallback Description" {
		t.Errorf("description = %q, want %q", preview.Description, "Fallback Description")
	}
}

func TestFetchOG_NonHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"key": "value"}`)
	}))
	defer srv.Close()

	db := testutil.TestDB(t)
	repo := NewRepository(db)
	f := NewFetcherWithClient(repo, &http.Client{Timeout: fetchTimeout})

	preview, err := f.FetchPreview(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchPreview: %v", err)
	}
	if preview != nil {
		t.Errorf("expected nil for non-HTML, got %+v", preview)
	}
}

func TestFetchOG_Non2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	db := testutil.TestDB(t)
	repo := NewRepository(db)
	f := NewFetcherWithClient(repo, &http.Client{Timeout: fetchTimeout})

	preview, err := f.FetchPreview(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchPreview: %v", err)
	}
	if preview != nil {
		t.Errorf("expected nil for 404, got %+v", preview)
	}

	// Should be cached as error in the database
	var fetchError string
	_ = db.QueryRow("SELECT fetch_error FROM link_preview_cache WHERE url = ?", srv.URL).Scan(&fetchError)
	if fetchError == "" {
		t.Error("expected fetch_error to be set in cache")
	}
}

func TestFetchOG_BodySizeLimit(t *testing.T) {
	// Create a response larger than maxBodySize
	largeHead := strings.Repeat("x", maxBodySize+1000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><head><title>Big Page</title><!-- %s --></head><body></body></html>`, largeHead)
	}))
	defer srv.Close()

	db := testutil.TestDB(t)
	repo := NewRepository(db)
	f := NewFetcherWithClient(repo, &http.Client{Timeout: fetchTimeout})

	// Should not crash, just may not find the title if it's beyond the limit
	preview, err := f.FetchPreview(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("FetchPreview: %v", err)
	}
	// The title comes before the large content, so it should be found
	if preview == nil {
		t.Fatal("expected preview even with large body")
	}
	if preview.Title != "Big Page" {
		t.Errorf("title = %q, want %q", preview.Title, "Big Page")
	}
}

func TestFetchOG_CacheHit(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<html><head><meta property="og:title" content="Cached"></head><body></body></html>`)
	}))
	defer srv.Close()

	db := testutil.TestDB(t)
	repo := NewRepository(db)
	f := NewFetcherWithClient(repo, &http.Client{Timeout: fetchTimeout})
	ctx := context.Background()

	// First call fetches from server
	p1, _ := f.FetchPreview(ctx, srv.URL)
	if p1 == nil || p1.Title != "Cached" {
		t.Fatal("first fetch failed")
	}

	// Second call should use cache
	p2, _ := f.FetchPreview(ctx, srv.URL)
	if p2 == nil || p2.Title != "Cached" {
		t.Fatal("second fetch failed")
	}

	if callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", callCount)
	}
}

func TestPrivateIPRejection(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.ip, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse IP %s", tt.ip)
			}
			got := isPrivateIP(ip)
			if got != tt.want {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestParseOG_StopsAtBody(t *testing.T) {
	html := `<html><head>
		<meta property="og:title" content="Head Title">
	</head><body>
		<meta property="og:title" content="Body Title">
	</body></html>`

	data, err := parseOG(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parseOG: %v", err)
	}
	if data.Title != "Head Title" {
		t.Errorf("title = %q, want %q", data.Title, "Head Title")
	}
}
