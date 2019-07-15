kubectl unfork

A kubectl plugin to find forked helm charts running in a cluster and migrate them off of forks, back to upstream with kustomize patches.


Usage:

```
kubectl unfork
```

Connects to the cluster in kubeContext, listing all installed Helm Charts

