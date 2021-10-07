module github.com/RedHatInsights/clowder

go 1.15

require (
	github.com/RedHatInsights/cyndi-operator v0.1.4
	github.com/RedHatInsights/go-difflib v1.0.0
	github.com/RedHatInsights/strimzi-client-go v0.24.0-1
	github.com/go-logr/logr v0.3.0
	github.com/minio/minio-go/v7 v7.0.10
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.47.1
	github.com/prometheus/client_golang v1.7.1
	github.com/segmentio/kafka-go v0.4.16
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.15.0
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v0.20.2
	sigs.k8s.io/controller-runtime v0.8.3
)
