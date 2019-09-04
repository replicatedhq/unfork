package cli

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/unfork/pkg/chartindex"
	"github.com/spf13/cobra"
)

func IndexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "index",
		Short:  "Update the index",
		Long:   ``,
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			index := chartindex.ChartIndex{}
			if err := index.Build(); err != nil {
				return errors.Cause(err)
			}

			dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
			if err != nil {
				return errors.Cause(err)
			}

			indexFile := filepath.Join(dir, "charts.json")

			if err := index.Save(indexFile); err != nil {
				return errors.Cause(err)
			}

			return nil
		},
	}

	return cmd
}
