package unforker

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/base"
	yamlv2 "gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/kustomize/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/pkg/resource"
)

type MinimalK8sYaml struct {
	Kind     string             `json:"kind" yaml:"kind"`
	Metadata MinimalK8sMetadata `json:"metadata" yaml:"metadata"`
}

type MinimalK8sMetadata struct {
	Name      string `json:"name" yaml:"name"`
	Namespace string `json:"namespace" yaml:"namespace"`
}

func createPatches(forkedPath string, upstreamPath string) (map[string][]byte, error) {
	upstreamFiles := map[string][]byte{}
	forkedFiles := map[string][]byte{}

	filepath.Walk(forkedPath, func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}

		if info.IsDir() {
			return nil
		}

		content, err := ioutil.ReadFile(filename)
		if err != nil {
			return errors.Wrap(err, "failed to read file")
		}

		// ignore files that don't have a gvk-yaml
		o := base.OverlySimpleGVK{}
		if err := yamlv2.Unmarshal(content, &o); err != nil {
			return nil
		}
		if o.APIVersion == "" || o.Kind == "" {
			return nil
		}

		forkedFiles[filename] = content
		return nil
	})

	filepath.Walk(upstreamPath, func(filename string, info os.FileInfo, err error) error {
		if err != nil {
			panic(err)
		}

		if info.IsDir() {
			return nil
		}

		content, err := ioutil.ReadFile(filename)
		if err != nil {
			return errors.Wrap(err, "failed to read file")
		}

		// ignore files that don't have a gvk-yaml
		o := base.OverlySimpleGVK{}
		if err := yamlv2.Unmarshal(content, &o); err != nil {
			return nil
		}
		if o.APIVersion == "" || o.Kind == "" {
			return nil
		}

		upstreamFiles[filename] = content
		return nil
	})

	// Walk all in the fork, creating patches as needed
	patches := map[string][]byte{}
	for filename, content := range forkedFiles {
		upstreamPath, err := findMatchingUpstreamPath(upstreamFiles, content)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find upstream path")
		}

		patch, err := createTwoWayMergePatch(upstreamFiles[upstreamPath], content)
		if err != nil {
			continue
			// Helm templates. You know?
			// return nil, errors.Wrap(err, "failed to create patch")
		}

		include, err := containsNonGVK(patch)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check if should include patch")
		}

		if include {
			_, n := path.Split(filename)
			patches[n] = patch
		}
	}

	return patches, nil
}

func findMatchingUpstreamPath(upstreamFiles map[string][]byte, forkedContent []byte) (string, error) {
	f := MinimalK8sYaml{}
	if err := yamlv2.Unmarshal(forkedContent, &f); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal forked yaml")
	}

	for upstreamFilename, upstreamContents := range upstreamFiles {
		u := MinimalK8sYaml{}
		if err := yamlv2.Unmarshal(upstreamContents, &u); err != nil {
			return "", errors.Wrap(err, "failed to unmarshal uupstream yaml")
		}

		if u.Kind == f.Kind {
			if u.Metadata.Name == f.Metadata.Name {
				// namespaces match only if they both have one?
				if u.Metadata.Namespace == "" || f.Metadata.Namespace == "" {
					return upstreamFilename, nil
				}

				if u.Metadata.Namespace == f.Metadata.Namespace {
					return upstreamFilename, nil
				}
			}
		}
	}

	return "", nil
}

func createTwoWayMergePatch(original []byte, modified []byte) ([]byte, error) {
	originalJSON, err := yaml.YAMLToJSON(original)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert original yaml to json")
	}

	modifiedJSON, err := yaml.YAMLToJSON(modified)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert modified yaml to json")
	}

	resourceFactory := resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl())
	resources, err := resourceFactory.SliceFromBytes(originalJSON)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse original json")
	}
	if len(resources) != 1 {
		return nil, errors.New("cannot handle > 1 resource")
	}
	originalResource := resources[0]

	versionedObj, err := scheme.Scheme.New(schema.GroupVersionKind{
		Group:   originalResource.Id().Gvk().Group,
		Version: originalResource.Id().Gvk().Version,
		Kind:    originalResource.Id().Gvk().Kind,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to read gvk from original resource")
	}

	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(originalJSON, modifiedJSON, versionedObj)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create two way merge patch")
	}

	// modifiedPatchJSON, err := p.writeHeaderToPatch(originalJSON, patchBytes)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "write original header to patch")
	// }

	patch, err := yaml.JSONToYAML(patchBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert patch to yaml")
	}

	return patch, nil
}

func containsNonGVK(data []byte) (bool, error) {
	gvk := []string{
		"apiVersion",
		"kind",
		"metadata",
	}

	unmarshalled := make(map[string]interface{})
	err := yaml.Unmarshal(data, &unmarshalled)
	if err != nil {
		return false, errors.Wrap(err, "failed to unmarshal patch")
	}

	keys := make([]string, 0, 0)
	for k := range unmarshalled {
		keys = append(keys, k)
	}

	for key := range keys {
		isGvk := false
		for gvkKey := range gvk {
			if key == gvkKey {
				isGvk = true
			}
		}

		if !isGvk {
			return true, nil
		}
	}

	return false, nil
}
