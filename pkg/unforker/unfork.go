package unforker

import (
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/unfork/pkg/chartindex"
	"github.com/replicatedhq/unfork/pkg/util"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	"k8s.io/helm/pkg/timeconv"
)

func Unfork(localChart *LocalChart, upstreamChartMatch chartindex.ChartMatch) error {
	// write this out to a replicatedhq/kots compatible structure
	unforkPath := path.Join(util.HomeDir(), localChart.HelmName)
	_, err := os.Stat(unforkPath)
	if !os.IsNotExist(err) {
		// dir exists, uverwriting is a TODO
		return errors.Errorf("path %q already exists or cannot open", path.Join(util.HomeDir(), localChart.HelmName))
	}

	pullOptions := pull.PullOptions{
		Downstreams:         []string{"local"},
		ExcludeKotsKinds:    true,
		RootDir:             unforkPath,
		ExcludeAdminConsole: true,
		CreateAppDir:        false,
		Silent:              true,
	}

	if _, err := pull.Pull(fmt.Sprintf("helm://%s/%s", upstreamChartMatch.Repo, upstreamChartMatch.Name), pullOptions); err != nil {
		return errors.Wrap(err, "failed to pull upstream")
	}

	// upstreamChart, err := fetchUpstreamChart(getKnownHelmRepoURI(upstreamChartMatch.Repo), upstreamChartMatch.Name, upstreamChartMatch.ChartVersion)
	// if err != nil {
	// 	return errors.Wrap(err, "failed to fetch upstream chart")
	// }
	// upstreamRoot := path.Join(util.HomeDir(), localChart.HelmName, "upstream")
	// if err := os.MkdirAll(upstreamRoot, 0755); err != nil {
	// 	return errors.Wrap(err, "failed to create upstream dir")
	// }
	// for name, content := range upstreamChart {
	// 	f := path.Join(upstreamRoot, name)
	// 	d, _ := path.Split(f)
	// 	if _, err := os.Stat(d); os.IsNotExist(err) {
	// 		if err := os.MkdirAll(d, 0755); err != nil {
	// 			return errors.Wrap(err, "failed to create upstream file dir")
	// 		}
	// 	}
	// 	if err := ioutil.WriteFile(f, content, 0644); err != nil {
	// 		return errors.Wrap(err, "failed to write file")
	// 	}
	// }

	// forkedManifests, err := renderChart(localChart.HelmName, localChart.Namespace, localChart.Chart, localChart.Templates, localChart.Values)
	// if err != nil {
	// 	return errors.Wrap(err, "failed to render local chart")
	// }

	return nil
}

func renderChart(helmName string, namespace string, c *chart.Chart, templates []*chart.Template, values map[string]*chart.Value) (map[string]string, error) {
	config := &chart.Config{Raw: string(""), Values: values}

	renderOpts := renderutil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      helmName,
			IsInstall: true,
			IsUpgrade: false,
			Time:      timeconv.Now(),
			Namespace: namespace,
		},
		KubeVersion: "1.16.0",
	}

	rendered, err := renderutil.Render(c, config, renderOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render chart")
	}

	return rendered, nil
}
