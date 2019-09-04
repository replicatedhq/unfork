package unforker

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/base"
	"gopkg.in/yaml.v2"
	// "k8s.io/apimachinery/pkg/runtime/schema"
)

type MinimalK8sYaml struct {
	Kind     string             `json:"kind" yaml:"kind"`
	Metadata MinimalK8sMetadata `json:"metadata" yaml:"metadata"`
}

type MinimalK8sMetadata struct {
	Name      string `json:"name" yaml:"name"`
	Namespace string `json:"namespace" yaml:"namespace"`
}

func createPatches(forkedPath string, upstreamPath string) error {
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
		if err := yaml.Unmarshal(content, &o); err != nil {
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
		if err := yaml.Unmarshal(content, &o); err != nil {
			return nil
		}
		if o.APIVersion == "" || o.Kind == "" {
			return nil
		}

		upstreamFiles[filename] = content
		return nil
	})

	// Walk all in the fork, creating patches as needed
	for _, content := range forkedFiles {
		upstreamPath, err := findMatchingUpstreamPath(upstreamFiles, content)
		if err != nil {
			return errors.Wrap(err, "failed to find upstream path")
		}

		if upstreamPath != "" {
			panic(upstreamPath)
		}
	}

	return nil
}

func findMatchingUpstreamPath(upstreamFiles map[string][]byte, forkedContent []byte) (string, error) {
	f := MinimalK8sYaml{}
	if err := yaml.Unmarshal(forkedContent, &f); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal forked yaml")
	}

	for upstreamFilename, upstreamContents := range upstreamFiles {
		u := MinimalK8sYaml{}
		if err := yaml.Unmarshal(upstreamContents, &u); err != nil {
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
