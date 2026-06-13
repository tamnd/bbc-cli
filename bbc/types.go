package bbc

import (
	"encoding/xml"
	"strings"
	"time"
)

// Article is the record emitted for a BBC news item.
type Article struct {
	Rank        int    `json:"rank"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Published   string `json:"published"`
	URL         string `json:"url"`
}

// Section is the record emitted by the sections command.
type Section struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// ─── wire types ──────────────────────────────────────────────────────────────

// rssItem is the XML wire type for a single <item> element.
type rssItem struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Link        string `xml:"link"`
	PubDate     string `xml:"pubDate"`
	GUID        string `xml:"guid"`
}

// rssFeed is the XML wire type for the top-level <rss> element.
type rssFeed struct {
	XMLName xml.Name  `xml:"rss"`
	Items   []rssItem `xml:"channel>item"`
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// parseDate parses a BBC pubDate string (RFC1123 with possible timezone name)
// and returns it formatted as "2006-01-02 15:04". Falls back to the raw string
// when parsing fails.
func parseDate(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	t, err := time.Parse(time.RFC1123, s)
	if err != nil {
		// Try RFC1123Z (numeric timezone like +0000)
		t, err = time.Parse(time.RFC1123Z, s)
		if err != nil {
			return s
		}
	}
	return t.UTC().Format("2006-01-02 15:04")
}

// wireToArticle converts a raw RSS item to an Article at the given rank.
func wireToArticle(it rssItem, rank int) Article {
	u := strings.TrimSpace(it.Link)
	if u == "" {
		u = strings.TrimSpace(it.GUID)
	}
	return Article{
		Rank:        rank,
		Title:       strings.TrimSpace(it.Title),
		Description: strings.TrimSpace(it.Description),
		Published:   parseDate(it.PubDate),
		URL:         u,
	}
}

// knownSections is the canonical list of BBC News sections.
var knownSections = []Section{
	{Name: "top", URL: "http://feeds.bbci.co.uk/news/rss.xml"},
	{Name: "world", URL: "http://feeds.bbci.co.uk/news/world/rss.xml"},
	{Name: "uk", URL: "http://feeds.bbci.co.uk/news/uk/rss.xml"},
	{Name: "tech", URL: "http://feeds.bbci.co.uk/news/technology/rss.xml"},
	{Name: "science", URL: "http://feeds.bbci.co.uk/news/science_and_environment/rss.xml"},
	{Name: "health", URL: "http://feeds.bbci.co.uk/news/health/rss.xml"},
	{Name: "business", URL: "http://feeds.bbci.co.uk/news/business/rss.xml"},
	{Name: "entertainment", URL: "http://feeds.bbci.co.uk/news/entertainment_and_arts/rss.xml"},
	{Name: "sport", URL: "http://feeds.bbci.co.uk/sport/rss.xml"},
	{Name: "politics", URL: "http://feeds.bbci.co.uk/news/politics/rss.xml"},
}
