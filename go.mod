module cloud.redhat.com/clowder/v2

go 1.13

require (
	github.com/RedHatInsights/strimzi-client-go v0.21.1-2
	github.com/go-logr/logr v0.2.1
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/minio/minio-go/v7 v7.0.5
	github.com/prometheus/client_golang v1.8.0
	github.com/segmentio/kafka-go v0.4.8
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.16.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	knative.dev/pkg v0.0.0-20210204171111-887806985c09
	sigs.k8s.io/controller-runtime v0.6.2
)
