package chartindex

import (
	"github.com/Masterminds/semver"
)

type ChartMatch struct {
	Repo               string
	Name               string
	ChartVersion       string
	AppVersion         string
	LatestChartVersion string
	LatestAppVersion   string
}

func FindBestUpstreamMatches(chartName string, chartVersion string, appVersion string) ([]ChartMatch, error) {
	chartMatches := []ChartMatch{}

	chartIndex, err := loadIndex()
	if err != nil {
		return nil, err
	}

	for _, indexChart := range chartIndex.charts {
		if indexChart.Name == chartName {
			var chartMatch *ChartMatch

			highestChartVersion := semver.MustParse("0.0.0")
			highestAppVersion := ""

			for _, version := range indexChart.Versions {
				parsedChartVersion := semver.MustParse(version.ChartVersion)
				if parsedChartVersion.GreaterThan(highestChartVersion) {
					highestChartVersion = parsedChartVersion
					highestAppVersion = version.AppVersion
				}

				if version.ChartVersion == chartVersion {
					if version.AppVersion == appVersion {

						chartMatch = &ChartMatch{
							Repo:         indexChart.Repo,
							Name:         indexChart.Name,
							ChartVersion: version.ChartVersion,
							AppVersion:   version.AppVersion,
						}
					}
				}
			}

			if chartMatch != nil {
				chartMatch.LatestChartVersion = highestChartVersion.String()
				chartMatch.LatestAppVersion = highestAppVersion
				chartMatches = append(chartMatches, *chartMatch)
			}
		}
	}
	return chartMatches, nil
}
