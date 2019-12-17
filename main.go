package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/helm/pkg/kube"
	releaseV2 "k8s.io/helm/pkg/proto/hapi/release"
	driverv2 "k8s.io/helm/pkg/storage/driver"

	releaseV3 "helm.sh/helm/v3/pkg/release"
	driverv3 "helm.sh/helm/v3/pkg/storage/driver"
)

var (
	addr = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")

	releaseInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "helm_release_info",
		Help: "Helm Release Information",
	}, []string{
		"release_namespace",
		"release_name",
		"helm_version",
		"storage_driver",
		"chart_name",
	})
	releaseStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "helm_release_status",
		Help: "Helm Release Status",
	}, []string{
		"release_namespace",
		"release_name",
		"release_status",
	})
	chartVersion = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "helm_release_chart_version",
		Help: "Helm Release Chart Version",
	}, []string{
		"release_namespace",
		"release_name",
		"chart_version",
		"chart",
	})

	statusCodeV2V3 = map[releaseV2.Status_Code]releaseV3.Status{
		releaseV2.Status_UNKNOWN:          releaseV3.StatusUnknown,
		releaseV2.Status_DEPLOYED:         releaseV3.StatusDeployed,
		releaseV2.Status_DELETED:          releaseV3.StatusUninstalled,
		releaseV2.Status_SUPERSEDED:       releaseV3.StatusSuperseded,
		releaseV2.Status_FAILED:           releaseV3.StatusFailed,
		releaseV2.Status_DELETING:         releaseV3.StatusUninstalling,
		releaseV2.Status_PENDING_INSTALL:  releaseV3.StatusPendingInstall,
		releaseV2.Status_PENDING_UPGRADE:  releaseV3.StatusPendingUpgrade,
		releaseV2.Status_PENDING_ROLLBACK: releaseV3.StatusPendingRollback,
	}
)

func main() {
	flag.Parse()

	log.Printf("Starting to listen address %s ...", *addr)

	http.HandleFunc("/", redirectHandler)
	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/healthz", func(_ http.ResponseWriter, _ *http.Request) {})

	_, err := kube.New(nil).KubernetesClientSet()
	if err != nil {
		log.Fatalf("Cannot initialize Kubernetes connection: %s", err)
	}

	log.Fatal(http.ListenAndServe(*addr, nil))
}

func redirectHandler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/metrics", http.StatusMovedPermanently)
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Handling metrics request ...")

	releaseInfo.Reset()
	releaseStatus.Reset()
	chartVersion.Reset()

	clientSet, err := kube.New(nil).KubernetesClientSet()
	if err != nil {
		log.Fatalf("Cannot initialize Kubernetes connection: %s", err)
	}

	namespaceList, err := clientSet.CoreV1().Namespaces().List(v1.ListOptions{})
	if err != nil {
		log.Fatalf("Cannot list namespaces: %s", err)
	}

	var wg sync.WaitGroup
	for _, ns := range namespaceList.Items {
		namespace := ns.GetName()
		cfgmapsv2 := driverv2.NewConfigMaps(clientSet.CoreV1().ConfigMaps(namespace))
		secretsv2 := driverv2.NewSecrets(clientSet.CoreV1().Secrets(namespace))
		for _, d := range [2]driverv2.Driver{cfgmapsv2, secretsv2} {
			wg.Add(1)
			go addDriverV2Values(&wg, d, namespace)
		}
		cfgmapsv3 := driverv3.NewConfigMaps(clientSet.CoreV1().ConfigMaps(namespace))
		secretsv3 := driverv3.NewSecrets(clientSet.CoreV1().Secrets(namespace))
		for _, d := range [2]driverv3.Driver{cfgmapsv3, secretsv3} {
			wg.Add(1)
			go addDriverV3Values(&wg, d, namespace)
		}
	}
	wg.Wait()

	promhttp.Handler().ServeHTTP(w, r)
	log.Println("Handled metrics request.")
}

func addDriverV2Values(wg *sync.WaitGroup, d driverv2.Driver, namespace string) {
	defer wg.Done()

	lastReleases := make(map[string]*releaseV2.Release)

	releaseList, err := d.List(func(_ *releaseV2.Release) bool { return true })
	if err != nil {
		log.Printf("Cannot list v2 releases in %s namespace stored in %s: %s", namespace, d.Name(), err)
	}
	for _, r := range releaseList {
		if lastRelease, ok := lastReleases[r.Name]; ok {
			if lastRelease.Version < r.Version {
				lastReleases[r.Name] = r
			}
		} else {
			lastReleases[r.Name] = r
		}
	}
	for _, r := range lastReleases {
		statusCode := r.Info.Status.Code
		// Map v3 Status_Code to v3 Status
		v3status := statusCodeV2V3[statusCode]
		// This field is added for convenience
		chart := fmt.Sprintf("%s-%s", r.Chart.Metadata.Name, r.Chart.Metadata.Version)
		releaseInfo.WithLabelValues(r.Namespace, r.Name, "v2", d.Name(), r.Chart.Metadata.Name).Set(float64(r.Version))
		releaseStatus.WithLabelValues(r.Namespace, r.Name, v3status.String()).Set(1)
		chartVersion.WithLabelValues(r.Namespace, r.Name, r.Chart.Metadata.Version, chart).Set(1)
	}
}

func addDriverV3Values(wg *sync.WaitGroup, d driverv3.Driver, namespace string) {
	defer wg.Done()

	lastReleases := make(map[string]*releaseV3.Release)

	releaseList, err := d.List(func(_ *releaseV3.Release) bool { return true })
	if err != nil {
		log.Printf("Cannot list v3 releases in %s namespace stored in %s: %s", namespace, d.Name(), err)
	}
	for _, r := range releaseList {
		if lastRelease, ok := lastReleases[r.Name]; ok {
			if lastRelease.Version < r.Version {
				lastReleases[r.Name] = r
			}
		} else {
			lastReleases[r.Name] = r
		}
	}
	for _, r := range lastReleases {
		status := r.Info.Status
		// This field is added for convenience
		chart := fmt.Sprintf("%s-%s", r.Chart.Metadata.Name, r.Chart.Metadata.Version)
		releaseInfo.WithLabelValues(r.Namespace, r.Name, "v3", d.Name(), r.Chart.Metadata.Name).Set(float64(r.Version))
		releaseStatus.WithLabelValues(r.Namespace, r.Name, status.String()).Set(1)
		chartVersion.WithLabelValues(r.Namespace, r.Name, r.Chart.Metadata.Version, chart).Set(1)
	}
}
