# `kubectl unfork`

A kubectl plugin to find forked helm charts running in a cluster, extract [Kustomize](https://kustomize.io) compatible patches, and allow you to delete the fork and return to the upstream Chart, while preserving your patches.

Usage:

```
curl https://unfork.io/install | bash
kubectl unfork
```

This plugin will:
- Connect to your Kubernetes cluster and search for a Helm Tiller pod.
- Connect to your Tiller using the Helm GRPC API and query to receive a list of all installed Helm Charts.
- Meanwhile, Unfork will download a list of all known Helm Charts from [Monocular](https://hub.helm.sh/).
- Comparing your Helm charts with the Monocular index, Unfork will attempt to determine which upstream your fork is from.
- Once you've confirmed the best upstream, Unfork will convert your custom changes into [Kustomize](https://kustomize.io) patches and resources.
- You can now update the Helm chart to the latest version, and re-apply your patches.

Note: Unfork does **not** make any changes to the applications running in your cluster. Unfork only needs access to your cluster in order to port-forward and gain access to Tiller.



