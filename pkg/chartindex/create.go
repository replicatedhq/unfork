package chartindex

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/helm/cmd/helm/search"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/repo"
)

var (
	kubeAppsPageSize = 100
)

func (i *ChartIndex) Save(filename string) error {
	b, err := json.Marshal(i.charts)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(filename, b, 0644); err != nil {
		return err
	}

	return nil
}

func (i *ChartIndex) Build() error {
	charts := []MonocularChartData{}

	currentPage := 1

	fmt.Printf("requesting repos from monocular\n")
	for {
		resp, err := http.Get(fmt.Sprintf("https://hub.kubeapps.com/api/chartsvc/v1/charts?size=%d&page=%d", kubeAppsPageSize, currentPage))
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		monocularResponse := MonocularResponse{}
		if err := json.Unmarshal(body, &monocularResponse); err != nil {
			return err
		}

		if currentPage > monocularResponse.Meta.TotalPages {
			break
		}

		charts = append(charts, monocularResponse.Data...)

		currentPage = currentPage + 1
	}

	i.charts = []ChartAndVersions{}

	searchedRepos := map[string]bool{}
	for _, chart := range charts {
		if _, ok := searchedRepos[chart.Attributes.Repo.URL]; ok {
			continue
		}

		versions, err := queryRepoForChartAndAppVersions(chart.Attributes.Repo.Name, chart.Attributes.Repo.URL)
		if err != nil {
			fmt.Printf("failed to fetch charts in repo %s, error was %#v\n", chart.Attributes.Repo.Name, err)
			continue
		}

		for chartAndRepoName, history := range versions {
			chartAndVersion := ChartAndVersions{
				Repo:     chart.Attributes.Repo.Name,
				Name:     strings.Split(chartAndRepoName, "/")[1],
				URI:      chart.Attributes.Repo.URL,
				Versions: history,
				Keywords: chart.Attributes.Keywords,
			}

			i.charts = append(i.charts, chartAndVersion)
		}

		searchedRepos[chart.Attributes.Repo.URL] = true
	}

	totalVersionCount := 0
	for _, item := range i.charts {
		totalVersionCount += len(item.Versions)
	}

	fmt.Printf("found %d total repos, and %d total versions\n", len(i.charts), totalVersionCount)
	return nil
}

func queryRepoForChartAndAppVersions(repoName string, repoURI string) (map[string][]ChartVersion, error) {
	fmt.Printf("getting version info for charts in repo %s (%s)\n", repoName, repoURI)
	helmHome, err := ioutil.TempDir("", "unfork")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(helmHome)
	if err := os.MkdirAll(filepath.Join(helmHome, "repository"), 0755); err != nil {
		return nil, err
	}
	reposFile := filepath.Join(helmHome, "repository", "repositories.yaml")

	repoIndexFile, err := ioutil.TempFile("", "index")
	if err != nil {
		return nil, err
	}
	defer os.Remove(repoIndexFile.Name())

	cacheIndexFile, err := ioutil.TempFile("", "cache")
	if err != nil {
		return nil, err
	}
	defer os.Remove(cacheIndexFile.Name())

	repoYAML := `apiVersion: v1
generated: "2019-05-29T14:31:58.906598702Z"
repositories: []`
	if err := ioutil.WriteFile(reposFile, []byte(repoYAML), 0644); err != nil {
		return nil, err
	}

	c := repo.Entry{
		Name:  repoName,
		Cache: repoIndexFile.Name(),
		URL:   repoURI,
	}
	r, err := repo.NewChartRepository(&c, getter.All(environment.EnvSettings{}))
	if err != nil {
		return nil, err
	}
	if err := r.DownloadIndexFile(cacheIndexFile.Name()); err != nil {
		return nil, err
	}

	rf, err := repo.LoadRepositoriesFile(reposFile)
	if err != nil {
		return nil, err
	}
	rf.Update(&c)

	i := search.NewIndex()
	for _, re := range rf.Repositories {
		n := re.Name
		ind, err := repo.LoadIndexFile(repoIndexFile.Name())
		if err != nil {
			return nil, err
		}

		i.AddRepo(n, ind, true)
	}

	chartAppVersions := make(map[string][]ChartVersion)
	for _, result := range i.All() {
		// dl := downloader.ChartDownloader{
		// 	HelmHome: helmpath.Home(helmHome),
		// 	Out:      os.Stdout,
		// 	Getters:  getter.All(environment.EnvSettings{}),
		// }

		key := fmt.Sprintf("%s/%s", repoName, result.Chart.GetName())
		versions, ok := chartAppVersions[key]
		if !ok {
			versions = []ChartVersion{}
		}
		versions = append(versions, ChartVersion{
			ChartVersion: result.Chart.GetVersion(),
			AppVersion:   result.Chart.GetAppVersion(),
		})
		chartAppVersions[key] = versions

		// archiveDir, err := ioutil.TempDir("", "archive")
		// if err != nil {
		// 	return nil, err
		// }
		// defer os.RemoveAll(archiveDir)

		// _, err = repo.FindChartInRepoURL(repoURI, result.Chart.GetName(), result.Chart.GetVersion(), "", "", "", getter.All(environment.EnvSettings{}))
		// if err != nil {
		// 	return nil, err
		// }

		// _, _, err = dl.DownloadTo(chartRef, result.Chart.GetVersion(), archiveDir)
		// if err != nil {
		// 	return nil, err
		// }
		// chartAppVersions[result.Chart.GetVersion()] = result.Chart.GetAppVersion()
	}

	return chartAppVersions, nil
}
