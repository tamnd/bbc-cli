package bbc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(srv *httptest.Server) *Client {
	cfg := DefaultConfig()
	cfg.BaseURL = srv.URL
	cfg.Rate = 0
	cfg.Retries = 0
	return NewClientWithConfig(cfg)
}

const sampleRSS = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
<channel>
  <title>BBC News</title>
  <item>
    <title>First headline</title>
    <description>First description</description>
    <link>https://www.bbc.co.uk/news/article-1</link>
    <pubDate>Mon, 01 Jan 2024 12:00:00 +0000</pubDate>
  </item>
  <item>
    <title>Second headline</title>
    <description><![CDATA[<p>Second <b>description</b></p>]]></description>
    <link>https://www.bbc.co.uk/news/article-2</link>
    <pubDate>Mon, 01 Jan 2024 11:00:00 +0000</pubDate>
  </item>
  <item>
    <title>Third headline</title>
    <description>Third description</description>
    <link>https://www.bbc.co.uk/news/article-3</link>
    <pubDate>Mon, 01 Jan 2024 10:00:00 +0000</pubDate>
  </item>
</channel>
</rss>`

func TestNews(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(sampleRSS))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	items, err := c.News(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}
	if items[0].Title != "First headline" {
		t.Errorf("Title = %q, want First headline", items[0].Title)
	}
}

func TestNewsLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		_, _ = w.Write([]byte(sampleRSS))
	}))
	defer srv.Close()

	c := newTestClient(srv)
	items, err := c.News(context.Background(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Errorf("got %d items, want 2 (limit)", len(items))
	}
}

func TestStripTags(t *testing.T) {
	got := stripTags("<p>Hello <b>world</b></p>")
	if got != "Hello world" {
		t.Errorf("stripTags = %q, want Hello world", got)
	}
}
