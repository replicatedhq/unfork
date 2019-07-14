package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/replicatedhq/unfork/pkg/unforker"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unfork",
		Short: "Convert forked Helm charts to Kustomize overlays",
		Long: `A kubectl plugin to find forked helm charts running in a cluster and migrate
them off of forks, back to upstream with kustomize patches.`,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// v := viper.GetViper()

			if len(args) == 0 {
				fmt.Println("Finding forked Helm Charts")
				// attempt to list forked charts
				unforkClient := unforker.NewUnforker()
				if err := unforkClient.FindAndListForksSync(); err != nil {
					return err
				}

				return nil
			} else {
				// attempt to unfork
			}
			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	cmd.Flags().String("namespace", "default", "namespace the collectors can be found in")
	cmd.Flags().String("kubecontext", filepath.Join(homeDir(), ".kube", "config"), "the kubecontext to use when connecting")

	viper.BindPFlags(cmd.Flags())

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}

func InitAndExecute() {
	if err := RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetEnvPrefix("UNFORK")
	viper.AutomaticEnv()
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
