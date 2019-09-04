package unforker

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	kotsk8sutil "github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/pull"
	kotsutil "github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/unfork/pkg/chartindex"
	"github.com/replicatedhq/unfork/pkg/util"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	"k8s.io/helm/pkg/timeconv"
	kustomizetypes "sigs.k8s.io/kustomize/v3/pkg/types"
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
		Downstreams:         []string{"unforked"},
		ExcludeKotsKinds:    true,
		RootDir:             unforkPath,
		ExcludeAdminConsole: true,
		CreateAppDir:        false,
		Silent:              true,
	}

	if _, err := pull.Pull(fmt.Sprintf("helm://%s/%s", upstreamChartMatch.Repo, upstreamChartMatch.Name), pullOptions); err != nil {
		return errors.Wrap(err, "failed to pull upstream")
	}

	forkedRoot, err := ioutil.TempDir("", "unfork")
	if err != nil {
		return errors.Wrap(err, "failed to create forked root")
	}
	// defer os.RemoveAll(forkedRoot)
	forkedManifests, err := renderChart(localChart.HelmName, localChart.Namespace, localChart.Chart, localChart.Templates, localChart.Values)
	if err != nil {
		return errors.Wrap(err, "failed to render forked chart")
	}
	for name, content := range forkedManifests {
		f := path.Join(forkedRoot, name)
		d, _ := path.Split(f)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0755); err != nil {
				return errors.Wrap(err, "failed to create forked file dir")
			}
		}
		if err := ioutil.WriteFile(f, []byte(content), 0644); err != nil {
			return errors.Wrap(err, "failed to write file")
		}
	}

	// Unfork the content in forkedRoot from the base in the pull.  this will extract patches
	// write them to downstreams/unforked
	resources, patches, err := createPatches(forkedRoot, path.Join(unforkPath, "base"))
	if err != nil {
		return errors.Wrap(err, "faield to create patches")
	}

	unforkPatchDir := path.Join(unforkPath, "overlays", "downstreams", "unforked")
	patchesForKustomization := []string{}
	resourcesForKustomization := []string{}

	for filename, content := range resources {
		filePath := path.Join(unforkPatchDir, filename)
		d, f := path.Split(filePath)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0755); err != nil {
				return errors.Wrap(err, "failed to make dir")
			}
		}

		if err := ioutil.WriteFile(path.Join(unforkPatchDir, filename), content, 0644); err != nil {
			return errors.Wrap(err, "failed to write resource")
		}

		resourcesForKustomization = append(resourcesForKustomization, f)
	}

	for filename, content := range patches {
		filePath := path.Join(unforkPatchDir, filename)
		d, f := path.Split(filePath)
		if _, err := os.Stat(d); os.IsNotExist(err) {
			if err := os.MkdirAll(d, 0755); err != nil {
				return errors.Wrap(err, "failed to make dir")
			}
		}

		if err := ioutil.WriteFile(path.Join(unforkPatchDir, filename), content, 0644); err != nil {
			return errors.Wrap(err, "failed to write patch")
		}

		patchesForKustomization = append(patchesForKustomization, f)
	}

	k, err := kotsk8sutil.ReadKustomizationFromFile(path.Join(unforkPatchDir, "kustomization.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to read kustomization")
	}

	for _, f := range patchesForKustomization {
		k.PatchesStrategicMerge = append(k.PatchesStrategicMerge, kustomizetypes.PatchStrategicMerge(f))
	}
	for _, r := range resourcesForKustomization {
		k.Resources = append(k.Resources, r)
	}
	if err := kotsk8sutil.WriteKustomizationToFile(k, path.Join(unforkPatchDir, "kustomization.yaml")); err != nil {
		return errors.Wrap(err, "failed to write kustomization")
	}

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

	// Remove common prefixes from these files to make it easier to manage later
	var commonPrefix []string

	for filename, _ := range rendered {
		d, _ := path.Split(filename)
		dirs := strings.Split(d, string(os.PathSeparator))
		if commonPrefix == nil {
			commonPrefix = dirs
		}
		commonPrefix = kotsutil.CommonSlicePrefix(commonPrefix, dirs)
	}

	cleanedRendered := map[string]string{}
	for filename, content := range rendered {
		d, f := path.Split(filename)
		d2 := strings.Split(d, string(os.PathSeparator))

		d2 = d2[len(commonPrefix):]
		cleanedRendered[path.Join(path.Join(d2...), f)] = content
	}

	return cleanedRendered, nil
}
