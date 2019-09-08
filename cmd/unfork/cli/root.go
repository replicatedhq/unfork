package cli

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	ui "github.com/gizak/termui/v3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/unfork/pkg/chartindex"
	"github.com/replicatedhq/unfork/pkg/unforker"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	currentPage = "home"

	unforkClient          *unforker.Unforker
	kubernetesConfigFlags *genericclioptions.ConfigFlags
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
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			_ = viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
				if err != nil {
					return errors.Cause(err)
				}
				indexFile := path.Join(dir, "charts.json")
				fi, err := os.Stat(indexFile)
				fetchIndex := false
				if os.IsNotExist(err) {
					fmt.Println("\nBuilding a local index of available Helm charts. This is needed to find the best upstream, and will only take a few seconds")
					fetchIndex = true
				} else if err != nil {
					return err
				} else {
					isOld := time.Now().Sub(fi.ModTime())
					if isOld > time.Hour*24*14 {
						fmt.Println("\nYour local index of available Helm charts is out of date. Updating them, this will only take a few seconds")
						fetchIndex = true
					}
				}

				if fetchIndex {
					index := chartindex.ChartIndex{}
					if err := index.Build(); err != nil {
						return errors.Cause(err)
					}

					if err := index.Save(indexFile); err != nil {
						return errors.Cause(err)
					}
				}

				hasTiller, err := unforker.HasTiller(kubernetesConfigFlags)
				if err != nil {
					return errors.Wrap(err, "failed to connect to cluster looking for tiller")
				}
				if !hasTiller {
					return errors.New("Unable to find a ready Tiller pod in the current cluster. Do you need to set a --kubeconfig?")
				}

				if err := ui.Init(); err != nil {
					return errors.Wrap(err, "failed to init the ui")
				}
				defer ui.Close()

				uiCh := make(chan unforker.UIEvent)

				u, err := unforker.NewUnforker(kubernetesConfigFlags, uiCh)
				if err != nil {
					return errors.Wrap(err, "failed to create unforker")
				}
				unforkClient = u

				go func() {
					err := unforkClient.StartDiscovery()
					if err != nil {
						ui.Close()
						fmt.Printf("%s\n", errors.Cause(err))
						os.Exit(1)
					}
				}()

				unforkUI := UnforkUI{
					home: createHome(uiCh),
					uiCh: uiCh,
				}

				if err := unforkUI.render(); err != nil {
					return errors.Wrap(err, "failed to render")
				}
				if err := unforkUI.eventLoop(); err != nil {
					return errors.Wrap(err, "error in event loop")
				}

				return nil
			}
			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	kubernetesConfigFlags = genericclioptions.NewConfigFlags(false)
	kubernetesConfigFlags.AddFlags(cmd.Flags())

	cmd.AddCommand(IndexCmd())

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

func (u *UnforkUI) render() error {
	if currentPage == "home" {
		if err := u.home.render(); err != nil {
			return errors.Wrap(err, "failed to render")
		}
	}

	return nil
}

func (u *UnforkUI) eventLoop() error {
	for e := range ui.PollEvents() {
		if currentPage == "home" {
			exit, err := u.home.handleEvent(e)
			if err != nil {
				return errors.Wrap(err, "failed to handle event")
			}
			if exit {
				return nil
			}
			continue
		}
	}

	return nil
}
