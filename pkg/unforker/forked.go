package unforker

import (
	"k8s.io/helm/pkg/proto/hapi/chart"
)

type UIEventPublishedFound struct {
	LocalChartName        string
	PublishedChartRepo    string
	PublishedChartVersion string
	PublishedAppVersion   string
}

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

type PublishedChart struct {
	ChartName                       string
	ChartVersion                    string
	AppVersion                      string
	Keywords                        []string
	Similarity                      int
	Templates                       []*chart.Template
	Values                          []*chart.Value
	RepoName                        string
	RepoURI                         string
	ChartAppVersions                map[string]string
	ClosestChartVersion             string
	ClosestAppVersion               string
	PatchCount                      int
	LineCount                       int
	HasExactMatchChartVersion       bool
	HasExactMatchAppVersion         bool
	HasExactMatchAppAndChartVersion bool
}
