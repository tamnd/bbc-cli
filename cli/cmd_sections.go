package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/bbc-cli/bbc"
)

func (a *App) sectionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sections",
		Short: "List all known BBC News sections and their feed URLs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			secs := bbc.KnownSections()
			return a.renderOrEmpty(secs, len(secs))
		},
	}
}
