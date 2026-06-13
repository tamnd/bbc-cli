// Package bbc is the library behind the bbc command: the HTTP client,
// RSS parser, and typed data models for BBC News feeds.
//
// All data comes from the official BBC RSS feeds at feeds.bbci.co.uk.
// No API key or authentication is required.
package bbc

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to BBC servers.
const DefaultUserAgent = "bbc/dev (+https://github.com/tamnd/bbc-cli)"

// Config holds constructor parameters for Client.
type Config struct {
	// BaseURL is the scheme+host prefix for all feed URLs.
	// Default: "http://feeds.bbci.co.uk"
	BaseURL   string
	UserAgent string
	// Rate is the minimum gap between requests. Zero means no pacing.
	Rate    time.Duration
	Retries int
	Timeout time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "http://feeds.bbci.co.uk",
		UserAgent: DefaultUserAgent,
		Rate:      500 * time.Millisecond,
		Retries:   3,
		Timeout:   30 * time.Second,
	}
}

// Client talks to the BBC RSS feeds.
type Client struct {
	httpClient *http.Client
	baseURL    string
	userAgent  string
	rate       time.Duration
	retries    int
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client with the given config.
func NewClient(cfg Config) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: cfg.Timeout},
		baseURL:    cfg.BaseURL,
		userAgent:  cfg.UserAgent,
		rate:       cfg.Rate,
		retries:    cfg.Retries,
	}
}

// Feed fetches the RSS feed at path (e.g. "/news/rss.xml") relative to BaseURL
// and returns up to limit Article records. limit=0 returns all items.
func (c *Client) Feed(ctx context.Context, path string, limit int) ([]Article, error) {
	u := c.baseURL + path
	body, err := c.get(ctx, u)
	if err != nil {
		return nil, err
	}
	var feed rssFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parse %s: %w", u, err)
	}
	items := feed.Items
	if limit > 0 && limit < len(items) {
		items = items[:limit]
	}
	out := make([]Article, len(items))
	for i, it := range items {
		out[i] = wireToArticle(it, i+1)
	}
	return out, nil
}

// SectionPath maps a section name to a URL path suffix suitable for Feed().
// Unknown names fall back to /news/<name>/rss.xml.
func SectionPath(section string) string {
	switch section {
	case "top", "home", "news":
		return "/news/rss.xml"
	case "world":
		return "/news/world/rss.xml"
	case "uk":
		return "/news/uk/rss.xml"
	case "tech", "technology":
		return "/news/technology/rss.xml"
	case "science":
		return "/news/science_and_environment/rss.xml"
	case "health":
		return "/news/health/rss.xml"
	case "business":
		return "/news/business/rss.xml"
	case "entertainment":
		return "/news/entertainment_and_arts/rss.xml"
	case "sport":
		return "/sport/rss.xml"
	case "politics":
		return "/news/politics/rss.xml"
	default:
		return "/news/" + section + "/rss.xml"
	}
}

// KnownSections returns all built-in Section records in a stable order.
func KnownSections() []Section {
	out := make([]Section, len(knownSections))
	copy(out, knownSections)
	return out
}

// ─── HTTP internals ───────────────────────────────────────────────────────────

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.rate <= 0 {
		return
	}
	if wait := c.rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}
