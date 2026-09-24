package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSearchAnnotationSecret_MatchFound(t *testing.T) {
	secrets := []core.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myapp-db-prod",
				Namespace: "test-ns",
				Annotations: map[string]string{
					DatabaseAnnotationKey: "myapp",
				},
			},
			Data: map[string][]byte{
				"db.host":     []byte("myapp-db-prod.rds.example.com"),
				"db.port":     []byte("5432"),
				"db.user":     []byte("admin"),
				"db.password": []byte("secret"),
				"db.name":     []byte("myappdb"),
			},
		},
	}

	configs, err := searchAnnotationSecret("myapp", secrets)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(configs))
	assert.Equal(t, "myapp-db-prod.rds.example.com", configs[0].Config.Hostname)
	assert.Equal(t, 5432, configs[0].Config.Port)
	assert.Equal(t, "admin", configs[0].Config.Username)
	assert.Equal(t, "secret", configs[0].Config.Password)
	assert.Equal(t, "myappdb", configs[0].Config.Name)
	assert.Equal(t, "myapp-db-prod", configs[0].Ref.Name)
	assert.Equal(t, "test-ns", configs[0].Ref.Namespace)
}

func TestSearchAnnotationSecret_NoMatch(t *testing.T) {
	secrets := []core.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other-db",
				Namespace: "test-ns",
				Annotations: map[string]string{
					DatabaseAnnotationKey: "other-app",
				},
			},
			Data: map[string][]byte{
				"db.host":     []byte("other-db.rds.example.com"),
				"db.port":     []byte("5432"),
				"db.user":     []byte("user"),
				"db.password": []byte("pass"),
				"db.name":     []byte("otherdb"),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "no-annotation-db",
				Namespace: "test-ns",
			},
			Data: map[string][]byte{
				"db.host":     []byte("no-annotation.rds.example.com"),
				"db.port":     []byte("5432"),
				"db.user":     []byte("user"),
				"db.password": []byte("pass"),
				"db.name":     []byte("nodb"),
			},
		},
	}

	configs, err := searchAnnotationSecret("myapp", secrets)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(configs))
}

func TestSearchAnnotationSecret_MultipleSecrets(t *testing.T) {
	secrets := []core.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myapp-db-arestore",
				Namespace: "test-ns",
			},
			Data: map[string][]byte{
				"db.host":     []byte("myapp-db-arestore.rds.example.com"),
				"db.port":     []byte("5432"),
				"db.user":     []byte("restore-user"),
				"db.password": []byte("restore-pass"),
				"db.name":     []byte("myappdb"),
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "myapp-db-prod",
				Namespace: "test-ns",
				Annotations: map[string]string{
					DatabaseAnnotationKey: "myapp",
				},
			},
			Data: map[string][]byte{
				"db.host":     []byte("myapp-db-prod.rds.example.com"),
				"db.port":     []byte("5432"),
				"db.user":     []byte("prod-user"),
				"db.password": []byte("prod-pass"),
				"db.name":     []byte("myappdb"),
			},
		},
	}

	configs, err := searchAnnotationSecret("myapp", secrets)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(configs))
	assert.Equal(t, "myapp-db-prod.rds.example.com", configs[0].Config.Hostname)
	assert.Equal(t, "prod-user", configs[0].Config.Username)
	assert.Equal(t, "myapp-db-prod", configs[0].Ref.Name)
}

func TestGenDbConfigs_ValidSecret(t *testing.T) {
	secrets := []core.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-db",
				Namespace: "test-ns",
			},
			Data: map[string][]byte{
				"db.host":     []byte("test-db-prod.rds.example.com"),
				"db.port":     []byte("5432"),
				"db.user":     []byte("user"),
				"db.password": []byte("password"),
				"db.name":     []byte("testdb"),
			},
		},
	}

	configs, err := genDbConfigs(secrets, false)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(configs))
	assert.Equal(t, "test-db-prod.rds.example.com", configs[0].Config.Hostname)
	assert.Equal(t, 5432, configs[0].Config.Port)
	assert.Equal(t, "user", configs[0].Config.Username)
	assert.Equal(t, "password", configs[0].Config.Password)
	assert.Equal(t, "testdb", configs[0].Config.Name)
	assert.Equal(t, "verify-full", configs[0].Config.SslMode)
}

func TestGenDbConfigs_MissingKeys(t *testing.T) {
	secrets := []core.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "incomplete-db",
				Namespace: "test-ns",
			},
			Data: map[string][]byte{
				"db.host": []byte("incomplete.rds.example.com"),
				"db.port": []byte("5432"),
			},
		},
	}

	configs, err := genDbConfigs(secrets, false)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(configs))
}
