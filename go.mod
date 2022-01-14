module github.com/RedHatInsights/clowder

go 1.16

require (
	github.com/RedHatInsights/cyndi-operator v0.1.4
	github.com/RedHatInsights/go-difflib v1.0.0
	github.com/RedHatInsights/rhc-osdk-utils v0.4.1
	github.com/RedHatInsights/strimzi-client-go v0.26.0
	github.com/go-logr/logr v0.4.0
	github.com/go-logr/zapr v0.4.0
	github.com/kedacore/keda/v2 v2.5.0
	github.com/lib/pq v1.10.4
	github.com/minio/minio-go/v7 v7.0.10
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.17.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.47.1
	github.com/prometheus/client_golang v1.11.0
	github.com/segmentio/kafka-go v0.4.16
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.19.1
	k8s.io/api v0.22.4
	k8s.io/apiextensions-apiserver v0.22.4
	k8s.io/apimachinery v0.22.4
	k8s.io/client-go v0.22.4
	sigs.k8s.io/cluster-api v1.0.1
	sigs.k8s.io/controller-runtime v0.10.3
)
