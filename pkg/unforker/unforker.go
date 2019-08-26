package unforker

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Unforker struct {
	kubecontext        string
	client             *kubernetes.Clientset
	charts             map[*LocalChart][]*PublishedChart
	uiCh               chan UIEvent
	monocularCompleted bool
	publishedCharts    []*PublishedChart
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
		charts:      make(map[*LocalChart][]*PublishedChart),
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
			u.charts[localChart] = make([]*PublishedChart, 0, 0)
		}
	}

	return nil
}

func (u *Unforker) LocalChartAtIndex(idx int) (*LocalChart, []*PublishedChart, error) {
	localCharts := make([]*LocalChart, 0, 0)
	for k := range u.charts {
		localCharts = append(localCharts, k)
	}

	return localCharts[idx], u.charts[localCharts[idx]], nil
}
