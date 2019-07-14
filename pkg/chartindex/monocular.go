package chartindex

import (
	"time"
)

type MonocularResponse struct {
	Data []MonocularChartData `json:"data"`
	Meta MonocularMeta        `json:"meta"`
}

type MonocularMeta struct {
	TotalPages int `json:"totalPages"`
}

type MonocularChartData struct {
	ID            string                      `json:"id"`
	Type          string                      `json:"type"`
	Attributes    MonocularChartAttributes    `json:"attributes"`
	Relationships MonocularChartRelationships `json:"relationships"`
}

type MonocularChartAttributes struct {
	Name     string             `json:"name"`
	Keywords []string           `json:"keywords"`
	Repo     MonocularChartRepo `json:"repo"`
}

type MonocularChartRepo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type MonocularChartRelationships struct {
	LatestChartVersion MonocularChartVersion `json:"latestChartVersion"`
}

type MonocularChartVersion struct {
	Data MonocularChartVersionData `json:"data"`
}

type MonocularChartVersionData struct {
	Version    string    `json:"version"`
	AppVersion string    `json:"app_version"`
	Created    time.Time `json:"created"`
	Readme     string    `json:"readme"`
	Values     string    `json:"values"`
}
