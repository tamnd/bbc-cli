package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/tamnd/bbc-cli/bbc"
)

func (a *App) feedCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "feed <section>",
		Short: "Fetch any BBC News section by name",
		Long: `Fetch any BBC News section by name. Known names: top, world, uk, tech,
science, health, business, entertainment, sport, politics.
Unknown names fall back to /news/<name>/rss.xml.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return codeError(exitUsage, fmt.Errorf("feed requires exactly one section argument"))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			section := args[0]
			path := bbc.SectionPath(section)
			n := a.effectiveLimit(20)
			a.progressf("fetching section %q...", section)
			arts, err := a.client.Feed(cmd.Context(), path, n)
			if err != nil {
				return codeError(exitError, err)
			}
			return a.renderOrEmpty(arts, len(arts))
		},
	}
}
