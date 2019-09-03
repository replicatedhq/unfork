package unforker

import (
	"k8s.io/helm/pkg/proto/hapi/chart"
)

type UIEvent struct {
	EventName string
	Payload   interface{}
}

type LocalChart struct {
	IsTiller     bool
	HelmName     string
	ChartName    string
	ChartVersion string
	AppVersion   string
	Keywords     []string
	Templates    []*chart.Template
	Values       map[string]*chart.Value
}
