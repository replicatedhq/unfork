package cli

import (
	"os"
	"path/filepath"

	"github.com/replicatedhq/unfork/pkg/chartindex"
	"github.com/spf13/cobra"
)

func IndexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Update the index",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			index := chartindex.ChartIndex{}
			if err := index.Build(); err != nil {
				return err
			}

			dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
			if err != nil {
				return err
			}

			indexFile := filepath.Join(dir, "charts.json")

			if err := index.Save(indexFile); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
