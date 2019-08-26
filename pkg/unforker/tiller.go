package unforker

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/replicatedhq/unfork/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/release"
)

func (u *Unforker) getTillerPodName() (string, string, error) {
	selector := labels.Set{"app": "helm", "name": "tiller"}
	pods, err := u.client.CoreV1().Pods("kube-system").List(metav1.ListOptions{LabelSelector: selector.AsSelector().String()})
	if err != nil {
		return "", "", err
	}

	if len(pods.Items) < 1 {
		return "", "", nil
	}

	for _, pod := range pods.Items {
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady {
				return pod.Name, pod.Namespace, nil
			}
		}
	}

	return "", "", nil
}

func (u *Unforker) queryTillerForCharts(tillerPodName string, tillerNamespace string) ([]*LocalChart, error) {
	// Pick a random port for local forward. this will explode sometimes
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	localPort := 30000 + r.Intn(999)

	// fmt.Println("connecting to tiller")
	_, err := k8sutil.PortForward(u.kubecontext, localPort, 44134, tillerNamespace, tillerPodName)
	if err != nil {
		return nil, err
	}
	// fmt.Println("connected to tiller")

	// fmt.Println("requesting a list of all deployed releases from tiller")
	helmOptions := []helm.Option{helm.Host(fmt.Sprintf("localhost:%d", localPort)), helm.ConnectTimeout(5)}
	helmClient := helm.NewClient(helmOptions...)
	listReleaseOptions := helm.ReleaseListStatuses([]release.Status_Code{release.Status_DEPLOYED})
	response, err := helmClient.ListReleases(listReleaseOptions)
	if err != nil {
		return nil, err
	}

	tillerCharts := make([]*LocalChart, 0, 0)
	for _, tillerRelease := range response.Releases {
		chart := LocalChart{
			IsTiller:     true,
			HelmName:     tillerRelease.Name,
			ChartName:    tillerRelease.GetChart().GetMetadata().Name,
			ChartVersion: tillerRelease.GetChart().GetMetadata().Version,
			AppVersion:   tillerRelease.GetChart().GetMetadata().GetAppVersion(),
			Keywords:     tillerRelease.GetChart().GetMetadata().GetKeywords(),
			Templates:    tillerRelease.GetChart().GetTemplates(),
			Values:       tillerRelease.GetChart().GetValues().GetValues(),
		}

		tillerCharts = append(tillerCharts, &chart)
	}

	return tillerCharts, nil
}
