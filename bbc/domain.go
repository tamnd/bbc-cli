package bbc

import (
	"context"

	"github.com/tamnd/any-cli/kit"
)

func init() { kit.Register(Domain{}) }

// Domain is the bbc kit driver. It carries no state; the per-run Client is
// built by the factory Register hands kit.
type Domain struct{}

// Info describes the scheme, the hostnames a pasted link is matched against,
// and the identity reused for the binary's help and version.
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

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	kit.Handle(app, kit.OpMeta{Name: "news", Group: "news", List: true,
		URIType: "article", Summary: "Latest BBC News headlines"}, newsCmd)
}

func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := NewClient(DefaultConfig())
	if cfg.UserAgent != "" {
		c.userAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.httpClient.Timeout = cfg.Timeout
	}
	return c, nil
}

type newsIn struct {
	Limit  int     `kit:"flag,inherit" help:"max results"`
	Client *Client `kit:"inject"`
}

func newsCmd(ctx context.Context, in newsIn, emit func(*Article) error) error {
	articles, err := in.Client.News(ctx, in.Limit)
	if err != nil {
		return err
	}
	for i := range articles {
		if err := emit(&articles[i]); err != nil {
			return err
		}
	}
	return nil
}
