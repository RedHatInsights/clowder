module cloud.redhat.com/clowder/v2

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/minio/minio-go/v7 v7.0.5
	github.com/prometheus/client_golang v1.0.0
	github.com/segmentio/kafka-go v0.4.8
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.10.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.2
)
