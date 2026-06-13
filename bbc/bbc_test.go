package bbc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeRSS builds a minimal RSS 2.0 document containing the given raw <item> blobs.
func fakeRSS(items ...string) string {
	header := `<?xml version="1.0" encoding="UTF-8"?><rss version="2.0"><channel><title>BBC News</title>`
	footer := `</channel></rss>`
	return header + strings.Join(items, "") + footer
}

const sampleItem = `<item>
<title>Test Headline</title>
<description>Short summary of the story.</description>
<link>https://www.bbc.co.uk/news/test-123</link>
<pubDate>Sun, 14 Jun 2026 10:30:00 GMT</pubDate>
</item>`

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 3
	return NewClient(cfg)
}

func TestFeedReturnsArticles(t *testing.T) {
	xml := fakeRSS(sampleItem, sampleItem)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(xml))
	})
	arts, err := c.Feed(context.Background(), "/news/rss.xml", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 2 {
		t.Fatalf("got %d articles, want 2", len(arts))
	}
	if arts[0].Title != "Test Headline" {
		t.Errorf("title = %q", arts[0].Title)
	}
	if arts[0].URL != "https://www.bbc.co.uk/news/test-123" {
		t.Errorf("url = %q", arts[0].URL)
	}
	if arts[0].Published != "2026-06-14 10:30" {
		t.Errorf("published = %q", arts[0].Published)
	}
	if arts[0].Rank != 1 {
		t.Errorf("rank = %d, want 1", arts[0].Rank)
	}
}

func TestFeedLimit(t *testing.T) {
	items := strings.Repeat(sampleItem, 5)
	xml := fakeRSS(items)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(xml))
	})
	arts, err := c.Feed(context.Background(), "/news/rss.xml", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 2 {
		t.Fatalf("got %d articles with limit=2, want 2", len(arts))
	}
}

func TestFeedSendsUserAgent(t *testing.T) {
	var gotUA string
	xml := fakeRSS(sampleItem)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(xml))
	})
	_, err := c.Feed(context.Background(), "/news/rss.xml", 0)
	if err != nil {
		t.Fatal(err)
	}
	if gotUA == "" {
		t.Error("request carried no User-Agent")
	}
}

func TestFeedRetriesOn503(t *testing.T) {
	var hits int
	xml := fakeRSS(sampleItem)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(xml))
	})
	start := time.Now()
	arts, err := c.Feed(context.Background(), "/news/rss.xml", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) == 0 {
		t.Error("got no articles after retries")
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestFeedEmptyChannel(t *testing.T) {
	xml := fakeRSS() // no items
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(xml))
	})
	arts, err := c.Feed(context.Background(), "/news/rss.xml", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(arts) != 0 {
		t.Errorf("got %d articles from empty feed, want 0", len(arts))
	}
}

func TestFeedInvalidXML(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not xml at all!!!"))
	})
	_, err := c.Feed(context.Background(), "/news/rss.xml", 0)
	if err == nil {
		t.Error("expected error for invalid XML, got nil")
	}
}

func TestSectionPath(t *testing.T) {
	cases := []struct {
		name string
		want string
	}{
		{"top", "/news/rss.xml"},
		{"home", "/news/rss.xml"},
		{"news", "/news/rss.xml"},
		{"world", "/news/world/rss.xml"},
		{"uk", "/news/uk/rss.xml"},
		{"tech", "/news/technology/rss.xml"},
		{"technology", "/news/technology/rss.xml"},
		{"science", "/news/science_and_environment/rss.xml"},
		{"health", "/news/health/rss.xml"},
		{"business", "/news/business/rss.xml"},
		{"entertainment", "/news/entertainment_and_arts/rss.xml"},
		{"sport", "/sport/rss.xml"},
		{"politics", "/news/politics/rss.xml"},
	}
	for _, tc := range cases {
		got := SectionPath(tc.name)
		if got != tc.want {
			t.Errorf("SectionPath(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestSectionPathUnknown(t *testing.T) {
	got := SectionPath("weather")
	want := "/news/weather/rss.xml"
	if got != want {
		t.Errorf("SectionPath(unknown) = %q, want %q", got, want)
	}
}

func TestKnownSections(t *testing.T) {
	secs := KnownSections()
	if len(secs) != 10 {
		t.Fatalf("got %d sections, want 10", len(secs))
	}
	for _, s := range secs {
		if s.Name == "" {
			t.Error("section has empty name")
		}
		if s.URL == "" {
			t.Errorf("section %q has empty URL", s.Name)
		}
	}
}
