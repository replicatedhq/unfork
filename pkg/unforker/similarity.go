package unforker

type SimilarityMatch struct {
	ChartVersion string
	AppVersion   string
	PatchCount   int
	LineCount    int
}

// Helm doesn't store the upstream source in the cluster or in tiller
// and forks are even harder... so find the most similar involves calcularing
// a similarity score and going that way
func calculateSimilarityWithMetadata(localChart *LocalChart, publishedChart *PublishedChart) (bool, error) {
	// If the name, chart version and app version are identical, let's call it match early
	if localChart.ChartName == publishedChart.ChartName {
		if localChart.ChartVersion == publishedChart.ChartVersion {
			if localChart.AppVersion == publishedChart.AppVersion {
				return true, nil
			}
		}
	}

	keywordsMatch, err := keywordComparison(localChart.Keywords, publishedChart.Keywords)
	if err != nil {
		return false, err
	}

	isMatch := false
	if keywordsMatch > 50 {
		// 50% of the keywords in the chart match is compelling
		isMatch = true
	}
	return isMatch, nil
}

func keywordComparison(keywords1 []string, keywords2 []string) (int, error) {
	common := []string{}
	for _, keyword1 := range keywords1 {
		for _, keyword2 := range keywords2 {
			if keyword2 == keyword1 {
				common = append(common, keyword1)
			}
		}
	}

	for _, keyword2 := range keywords2 {
		for _, keyword1 := range keywords1 {
			if keyword1 == keyword2 {
				// add if not already added
				for _, c := range common {
					if c == keyword2 {
						goto Next
					}
				}

				common = append(common, keyword2)
			}
		Next:
		}
	}

	all := []string{}
	all = append(all, keywords1...)
	for _, keyword2 := range keywords2 {
		for _, a := range all {
			if a == keyword2 {
				goto AlreadyAdded
			}
		}

		all = append(all, keyword2)
	AlreadyAdded:
	}
	// keyword comparison can be 0-keywordSimilarityWeight
	// number of matches as a % of len of keywords1
	pct := 100 * float64(len(common)) / float64(len(all))

	return int(pct), nil
}

func calculateSimilarityWithChart(localChart *LocalChart, publishedChart *PublishedChart) (*SimilarityMatch, error) {
	similarityMatch := SimilarityMatch{
		ChartVersion: "",
		AppVersion:   "",
		PatchCount:   -1,
		LineCount:    -1,
	}

	for chartVersion, appVersion := range publishedChart.ChartAppVersions {
		if localChart.ChartVersion == chartVersion {
			if localChart.AppVersion == appVersion {
				// high likely hook of a match
				similarityMatch.ChartVersion = chartVersion
				similarityMatch.AppVersion = appVersion
				similarityMatch.PatchCount = 9
				similarityMatch.LineCount = 9
			}
		}
	}

	return &similarityMatch, nil
}
