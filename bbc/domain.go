package bbc

import (
	"context"

	"github.com/tamnd/any-cli/kit"
)

func init() { kit.Register(Domain{}) }

type Domain struct{}

func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "bbc",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "bbc",
			Short:  "A command line for BBC News.",
			Long: `A command line for BBC News.

Fetch the latest BBC News headlines via RSS. No API key required.`,
			Site: "https://www.bbc.co.uk/news",
			Repo: "https://github.com/tamnd/bbc-cli",
		},
	}
}

func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "news", Group: "news", List: true,
		URIType: "article", Summary: "Latest BBC News headlines"}, newsCmd)
}

func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := NewClientWithConfig(DefaultConfig())
	if cfg.UserAgent != "" {
		c.cfg.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.cfg.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.cfg.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.cfg.Timeout = cfg.Timeout
		c.http.Timeout = cfg.Timeout
	}
	return c, nil
}

type listIn struct {
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

func newsCmd(ctx context.Context, in listIn, emit func(*NewsItem) error) error {
	items, err := in.Client.News(ctx, in.Limit)
	if err != nil {
		return err
	}
	for _, item := range items {
		if err := emit(item); err != nil {
			return err
		}
	}
	return nil
}
