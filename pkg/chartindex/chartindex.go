package chartindex

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
)

type ChartIndex struct {
	charts []ChartAndVersions
}

type ChartAndVersions struct {
	Repo     string         `json:"repo"`
	Name     string         `json:"name"`
	URI      string         `json:"uri"`
	Versions []ChartVersion `json:"versions"`
	Keywords []string       `json:"keywords"`
}

type ChartVersion struct {
	ChartVersion string `json:"chartVersion"`
	AppVersion   string `json:"appVersion"`
}

func loadIndex() (*ChartIndex, error) {
	index := ChartIndex{}

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return nil, err
	}

	indexFile := filepath.Join(dir, "charts.json")

	b, err := ioutil.ReadFile(indexFile)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(b, &index.charts); err != nil {
		return nil, err
	}

	return &index, nil
}
