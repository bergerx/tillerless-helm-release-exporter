# tillerless-helm-release-exporter
Helm (both v2 and v3) exporter which collects information directly from [Release](https://helm.sh/docs/glossary/#release)'s ConfigMap or Secret storage backend without the need for a running Tiller component.

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/3276/badge)](https://bestpractices.coreinfrastructure.org/projects/3276)
[![Build Status](https://travis-ci.com/bergerx/tillerless-helm-release-exporter.svg?branch=master)](https://travis-ci.com/bergerx/tillerless-helm-release-exporter)
[![](https://images.microbadger.com/badges/image/bergerx/tillerless-helm-release-exporter.svg)](https://microbadger.com/images/bergerx/tillerless-helm-release-exporter "Get your own image badge on microbadger.com")
[![Docker pulls](https://img.shields.io/docker/pulls/bergerx/tillerless-helm-release-exporter.svg)](https://hub.docker.com/r/bergerx/tillerless-helm-release-exporter/)
[![Known Vulnerabilities](https://snyk.io//test/github/bergerx/tillerless-helm-release-exporter/badge.svg?targetFile=Gopkg.lock)](https://snyk.io//test/github/bergerx/tillerless-helm-release-exporter?targetFile=Gopkg.lock)
[![Go Report Card](https://goreportcard.com/badge/github.com/bergerx/tillerless-helm-release-exporter)](https://goreportcard.com/report/github.com/bergerx/tillerless-helm-release-exporter)

Collects metrics for Releases:
* deployed with Helm v2 with Tiller,
* in all Namespaces in the Kubernetes cluster indepdendently from where the Tiller is running,
* deployed with Helm v2 without in-cluster Tiller (e.g. [Tillerless Helm](https://github.com/rimusz/helm-tiller)),
* deployed Helm v3

Doesn't collect metrics for Releases:
* deployed with `helm template ... | kubectl apply -f -` (e.g. [Istio Install with Helm, Option 1](https://istio.io/docs/setup/install/helm/#option-1-install-with-helm-via-helm-template)). Since these are not really a [Helm Release](https://helm.sh/docs/glossary/#release), they are not stored in either ConfigMap or a Secret backend.

# Collected metrics
| Metric name| Metric type | Value |
|:---:|:---:|:---:|
| helm_release_info | Gauge | &lt;release-version&gt; |

Labels:
* **`chart_name`**
* **`chart_version`**
* **`release_name`**
* **`release_namespace`**
* **`helm_version`** Either "v2" or "v3"
* **`release_status`**: Uses Helm 3 status names as default. Helm 2 status names are mapped to Helm 3 statuses (e.g. "uninstalled" instead of "DELETED")
* **`storage_driver`**: Either `ConfigMap` or `Secret`, only these two storage drivers are collected.
* **`chart`** ("&lt;chart_name&gt;-&lt;chart_version&gt;") This is actually a redundant field but kept for convenience to write easier join queries since the `chart`/`helm.sh/chart` field has the value in this format.

# Example metrics
```
helm_release_info{chart="cert-manager-v0.9.1",chart_name="cert-manager",chart_version="v0.9.1",helm_version="v2",release_name="cert-manager",release_namespace="cert-manager",release_status="deployed",storage_driver="ConfigMap"} 16
helm_release_info{chart="grafana-3.8.15",chart_name="grafana",chart_version="3.8.15",helm_version="v2",release_name="grafana",release_namespace="prom",release_status="deployed",storage_driver="ConfigMap"} 3
helm_release_info{chart="heapster-1.0.1",chart_name="heapster",chart_version="1.0.1",helm_version="v2",release_name="heapster",release_namespace="kube-system",release_status="deployed",storage_driver="ConfigMap"} 30
helm_release_info{chart="helm-exporter-0.3.1",chart_name="helm-exporter",chart_version="0.3.1",helm_version="v2",release_name="helm-exporter",release_namespace="default",release_status="uninstalled",storage_driver="ConfigMap"} 15
helm_release_info{chart="home-assistant-0.9.7",chart_name="home-assistant",chart_version="0.9.7",helm_version="v2",release_name="home-assistant",release_namespace="home-assistant",release_status="deployed",storage_driver="ConfigMap"} 4
helm_release_info{chart="kubernetes-dashboard-1.9.0",chart_name="kubernetes-dashboard",chart_version="1.9.0",helm_version="v2",release_name="ui",release_namespace="kube-system",release_status="deployed",storage_driver="ConfigMap"} 29
helm_release_info{chart="memcached-3.0.0",chart_name="memcached",chart_version="3.0.0",helm_version="v3",release_name="memc",release_namespace="test",release_status="deployed",storage_driver="Secret"} 1
helm_release_info{chart="nginx-ingress-1.22.0",chart_name="nginx-ingress",chart_version="1.22.0",helm_version="v2",release_name="ingress",release_namespace="kube-system",release_status="failed",storage_driver="ConfigMap"} 50
helm_release_info{chart="prometheus-9.1.1",chart_name="prometheus",chart_version="9.1.1",helm_version="v2",release_name="prometheus",release_namespace="prom",release_status="deployed",storage_driver="ConfigMap"} 4
```

# Why a new exporter
* Supports both Helm 2 and Helm 3 releases,
* Works without a Tiller (e.g. works even when you used `helm tiller` plugin, aka tillerless-helm),
* No need to deal with Tiller tls certs,
* Exports both chart and release name and versions in resulting metrics, chart version is significant to see the effective version, release version is actually a signal to annotate when the last Helm deployment is done,
* Release versions as the metric's value, so you can annotate release dates in grafana.

# Alternatives
* https://github.com/sstarcher/helm-exporter
* https://github.com/Kubedex/exporter
