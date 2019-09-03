package unforker

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Unforker struct {
	kubecontext string
	client      *kubernetes.Clientset
	uiCh        chan UIEvent
}

func NewUnforker(kubecontext string, uiCh chan UIEvent) (*Unforker, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubecontext)
	if err != nil {
		return nil, err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	u := &Unforker{
		kubecontext: kubecontext,
		client:      client,
		uiCh:        uiCh,
	}

	return u, nil
}

func (u *Unforker) StartDiscovery() error {
	if err := u.findAndListChartsSync(); err != nil {
		return err
	}

	return nil
}

func (u *Unforker) findAndListChartsSync() error {
	tillerPodName, tillerNamespace, err := u.getTillerPodName()
	if err != nil {
		return err
	}

	if tillerPodName != "" {
		tillerCharts, err := u.queryTillerForCharts(tillerPodName, tillerNamespace)
		if err != nil {
			return err
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
