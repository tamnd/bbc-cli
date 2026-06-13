package cli

import (
	"github.com/spf13/cobra"
)

// sectionCmd builds a command that fetches a fixed BBC RSS feed path.
func (a *App) sectionCmd(use, short, path string, defaultLimit int) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			n := a.effectiveLimit(defaultLimit)
			a.progressf("fetching %s...", use)
			arts, err := a.client.Feed(cmd.Context(), path, n)
			if err != nil {
				return codeError(exitError, err)
			}
			return a.renderOrEmpty(arts, len(arts))
		},
	}
}
