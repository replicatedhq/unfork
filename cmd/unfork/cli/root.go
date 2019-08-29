package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	ui "github.com/gizak/termui/v3"
	"github.com/replicatedhq/unfork/pkg/unforker"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	currentPage = "home"

	unforkClient *unforker.Unforker
)

type UnforkUI struct {
	home *Home
	uiCh chan unforker.UIEvent
}

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unfork",
		Short: "Convert forked Helm charts to Kustomize overlays",
		Long: `A kubectl plugin to find forked helm charts running in a cluster and migrate
them off of forks, back to upstream with kustomize patches.`,
		PreRun: func(cmd *cobra.Command, args []string) {
			_ = viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) == 0 {
				if err := ui.Init(); err != nil {
					return err
				}
				defer ui.Close()

				uiCh := make(chan unforker.UIEvent)

				u, err := unforker.NewUnforker(v.GetString("kubecontext"), uiCh)
				if err != nil {
					return err
				}
				unforkClient = u

				go func() {
					_ = unforkClient.StartDiscovery()
				}()

				unforkUI := UnforkUI{
					home: createHome(uiCh),
					uiCh: uiCh,
				}

				if err := unforkUI.render(); err != nil {
					return err
				}
				if err := unforkUI.eventLoop(); err != nil {
					return err
				}

				return nil
			}
			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	cmd.AddCommand(IndexCmd())

	cmd.Flags().String("namespace", "default", "namespace the collectors can be found in")
	cmd.Flags().String("kubecontext", filepath.Join(homeDir(), ".kube", "config"), "the kubecontext to use when connecting")

	_ = viper.BindPFlags(cmd.Flags())

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

func (u *UnforkUI) render() error {
	if currentPage == "home" {
		if err := u.home.render(); err != nil {
			return err
		}
	}

	return nil
}

func (u *UnforkUI) eventLoop() error {
	for e := range ui.PollEvents() {
		if currentPage == "home" {
			exit, err := u.home.handleEvent(e)
			if err != nil {
				return err
			}
			if exit {
				return nil
			}
			continue
		}
	}

	return nil
}
