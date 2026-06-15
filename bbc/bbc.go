package bbc

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const Host = "feeds.bbci.co.uk"
const rssURL = "http://feeds.bbci.co.uk/news/rss.xml"
const DefaultUserAgent = "Mozilla/5.0 (compatible; bbc-cli/0.1; +https://github.com/tamnd/bbc-cli)"

type Config struct {
	BaseURL   string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
	UserAgent string
}

func DefaultConfig() Config {
	return Config{
		BaseURL:   rssURL,
		Rate:      2 * time.Second,
		Retries:   3,
		Timeout:   30 * time.Second,
		UserAgent: DefaultUserAgent,
	}
}

type Client struct {
	cfg  Config
	http *http.Client
	last time.Time
}

func NewClient() *Client { return NewClientWithConfig(DefaultConfig()) }

func NewClientWithConfig(cfg Config) *Client {
	return &Client{cfg: cfg, http: &http.Client{Timeout: cfg.Timeout}}
}

var tagRE = regexp.MustCompile(`<[^>]+>`)

func stripTags(s string) string {
	return strings.TrimSpace(tagRE.ReplaceAllString(s, ""))
}

type rssChannel struct {
	Items []rssItem `xml:"channel>item"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
}

// NewsItem is one BBC news article.
type NewsItem struct {
	Title       string `json:"title"       kit:"id" table:"title"`
	Description string `json:"description"          table:"description"`
	Link        string `json:"link"                 table:"url,url"`
	PubDate     string `json:"pub_date"             table:"date"`
}

func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
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
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")

	resp, err := c.http.Do(req)
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
	b, err := io.ReadAll(resp.Body)
	return b, err != nil, err
}

func (c *Client) pace() {
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
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

// News fetches BBC News headlines.
func (c *Client) News(ctx context.Context, limit int) ([]*NewsItem, error) {
	feedURL := c.cfg.BaseURL
	if feedURL == "" {
		feedURL = rssURL
	}
	body, err := c.get(ctx, feedURL)
	if err != nil {
		return nil, err
	}
	var feed rssChannel
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parse rss: %w", err)
	}
	items := feed.Items
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	out := make([]*NewsItem, 0, len(items))
	for _, item := range items {
		out = append(out, &NewsItem{
			Title:       stripTags(item.Title),
			Description: stripTags(item.Description),
			Link:        item.Link,
			PubDate:     item.PubDate,
		})
	}
	return out, nil
}
