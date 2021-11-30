module github.com/RedHatInsights/clowder

go 1.15

require (
	github.com/RedHatInsights/cyndi-operator v0.1.4
	github.com/RedHatInsights/go-difflib v1.0.0
	github.com/RedHatInsights/rhc-osdk-utils v0.2.0
	github.com/RedHatInsights/strimzi-client-go v0.24.0-1
	github.com/go-logr/logr v0.4.0
	github.com/kedacore/keda/v2 v2.4.0
	github.com/lib/pq v1.10.3
	github.com/minio/minio-go/v7 v7.0.10
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.47.1
	github.com/prometheus/client_golang v1.11.0
	github.com/segmentio/kafka-go v0.4.16
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.19.0
	k8s.io/api v0.22.1
	k8s.io/apiextensions-apiserver v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v11.0.1-0.20190805182717-6502b5e7b1b5+incompatible
	sigs.k8s.io/controller-runtime v0.10.0
)

replace k8s.io/client-go => k8s.io/client-go v0.22.1
