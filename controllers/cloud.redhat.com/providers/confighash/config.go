package confighash

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	core "k8s.io/api/core/v1"
)

func (ch *confighashProvider) persistConfig(app *crd.ClowdApp, c *config.AppConfig) (string, error) {

	// In any case, we want to overwrite the secret, so this just
	// tests to see if the secret exists
	secret := &core.Secret{}
	err := ch.Cache.Create(CoreConfigSecret, app.GetNamespacedName("%s"), secret)

	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(c)
	if err != nil {
		return "", errors.Wrap("Failed to marshal config JSON", err)
	}

	h := sha256.New()
	h.Write([]byte(jsonData))
	hash := fmt.Sprintf("%x", h.Sum(nil))

	secret.StringData = map[string]string{
		"cdappconfig.json": string(jsonData),
	}

	app.SetObjectMeta(secret)

	err = ch.Cache.Update(CoreConfigSecret, secret)

	if err != nil {
		return "", err
	}

	return hash, err
}
