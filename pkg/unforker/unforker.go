package unforker

import (
	"github.com/pkg/errors"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

type Unforker struct {
	configFlags *genericclioptions.ConfigFlags
	client      *kubernetes.Clientset
	uiCh        chan UIEvent
}

func NewUnforker(configFlags *genericclioptions.ConfigFlags, uiCh chan UIEvent) (*Unforker, error) {
	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read kubeconfig")
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	u := &Unforker{
		configFlags: configFlags,
		client:      client,
		uiCh:        uiCh,
	}

	return u, nil
}

func (u *Unforker) StartDiscovery() error {
	if err := u.findAndListChartsSync(); err != nil {
		return errors.Wrap(err, "failed to find charts")
	}

	return nil
}

func (u *Unforker) findAndListChartsSync() error {
	tillerPodName, tillerNamespace, err := u.getTillerPodName()
	if err != nil {
		return errors.Wrap(err, "failed to get tiller pod")
	}

	if tillerPodName != "" {
		tillerCharts, err := u.queryTillerForCharts(tillerPodName, tillerNamespace)
		if err != nil {
			return errors.Wrap(err, "failed to query tiller")
		}

		for _, localChart := range tillerCharts {
			uiEvent := UIEvent{
				EventName: "new_chart",
				Payload:   localChart,
			}
			u.uiCh <- uiEvent
		}
	}

	return nil
}
