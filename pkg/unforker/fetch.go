package unforker

// This is a pretty blatent copy from replicatehq/kots for now
// we should vendor in kots and use it instead, but it's not possible
// at this level yet

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
	"k8s.io/helm/cmd/helm/search"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
)

func fetchUpstreamChart(repoURI string, chartName string, chartVersion string) (map[string][]byte, error) {
	helmHome, err := ioutil.TempDir("", "unfork")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temporary helm home")
	}
	defer os.RemoveAll(helmHome)

	if err := os.MkdirAll(filepath.Join(helmHome, "repository"), 0755); err != nil {
		return nil, errors.Wrap(err, "failed to make directory for helm home")
	}
	reposFile := filepath.Join(helmHome, "repository", "repositories.yaml")

	repoIndexFile, err := ioutil.TempFile("", "index")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temporary index file")
	}
	defer os.Remove(repoIndexFile.Name())

	cacheIndexFile, err := ioutil.TempFile("", "cache")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cache index file")
	}
	defer os.Remove(cacheIndexFile.Name())

	repoYAML := `apiVersion: v1
generated: "2019-05-29T14:31:58.906598702Z"
repositories: []`
	if err := ioutil.WriteFile(reposFile, []byte(repoYAML), 0644); err != nil {
		return nil, err
	}

	c := repo.Entry{
		Name:  "unfork",
		Cache: repoIndexFile.Name(),
		URL:   repoURI,
	}
	r, err := repo.NewChartRepository(&c, getter.All(environment.EnvSettings{}))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chart repository")
	}
	if err := r.DownloadIndexFile(cacheIndexFile.Name()); err != nil {
		return nil, errors.Wrap(err, "failed to download index file")
	}

	rf, err := repo.LoadRepositoriesFile(reposFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load repositories file")
	}
	rf.Update(&c)

	i := search.NewIndex()
	for _, re := range rf.Repositories {
		n := re.Name
		ind, err := repo.LoadIndexFile(repoIndexFile.Name())
		if err != nil {
			return nil, errors.Wrap(err, "failed to load index file")
		}

		i.AddRepo(n, ind, true)
	}

	if chartVersion == "" {
		highestChartVersion := semver.MustParse("0.0.0")
		for _, result := range i.All() {
			if result.Chart.GetName() != chartName {
				continue
			}

			v, err := semver.NewVersion(result.Chart.GetVersion())
			if err != nil {
				return nil, errors.Wrap(err, "unable to parse chart version")
			}

			if v.GreaterThan(highestChartVersion) {
				highestChartVersion = v
			}
		}

		chartVersion = highestChartVersion.String()
	}

	for _, result := range i.All() {
		if result.Chart.GetName() != chartName {
			continue
		}

		if result.Chart.GetVersion() != chartVersion {
			continue
		}

		dl := downloader.ChartDownloader{
			HelmHome: helmpath.Home(helmHome),
			Out:      os.Stdout,
			Getters:  getter.All(environment.EnvSettings{}),
		}

		archiveDir, err := ioutil.TempDir("", "archive")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create archive directory for chart")
		}
		defer os.RemoveAll(archiveDir)

		chartRef, err := repo.FindChartInRepoURL(repoURI, result.Chart.GetName(), chartVersion, "", "", "", getter.All(environment.EnvSettings{}))
		if err != nil {
			return nil, errors.Wrap(err, "failed to find chart in repo url")
		}

		_, _, err = dl.DownloadTo(chartRef, result.Chart.GetVersion(), archiveDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to download chart")
		}

		files, err := readTarGz(path.Join(archiveDir, fmt.Sprintf("%s-%s.tgz", chartName, chartVersion)))
		if err != nil {
			return nil, errors.Wrap(err, "failed to read chart archive")
		}

		return files, nil
	}

	return nil, errors.New("chart version not found")
}

func readTarGz(source string) (map[string][]byte, error) {
	f, err := os.Open(source)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open archive")
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gzip reader")
	}

	tarReader := tar.NewReader(gzf)

	type upstreamFile struct {
		Path    string
		Content []byte
	}
	files := []upstreamFile{}
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to advance in tar archive")
		}

		name := header.Name

		switch header.Typeflag {
		case tar.TypeReg:
			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(tarReader)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read file from tar archive")
			}
			file := upstreamFile{
				Path:    name,
				Content: buf.Bytes(),
			}

			files = append(files, file)
		default:
			continue
		}
	}

	// remove any common prefix from all files
	cleanedFiles := map[string][]byte{}

	if len(files) > 0 {
		firstFileDir, _ := path.Split(files[0].Path)
		commonPrefix := strings.Split(firstFileDir, string(os.PathSeparator))

		for _, file := range files {
			d, _ := path.Split(file.Path)
			dirs := strings.Split(d, string(os.PathSeparator))

			commonPrefix = util.CommonSlicePrefix(commonPrefix, dirs)

		}

		for _, file := range files {
			d, f := path.Split(file.Path)
			d2 := strings.Split(d, string(os.PathSeparator))

			d2 = d2[len(commonPrefix):]

			cleanedFiles[path.Join(path.Join(d2...), f)] = file.Content
		}
	}

	return cleanedFiles, nil
}
