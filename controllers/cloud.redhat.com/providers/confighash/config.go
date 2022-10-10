package confighash

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	core "k8s.io/api/core/v1"
)

func (ch *confighashProvider) persistConfig(app *crd.ClowdApp) (string, error) {

	// In any case, we want to overwrite the secret, so this just
	// tests to see if the secret exists
	secret := &core.Secret{}
	err := ch.Cache.Create(CoreConfigSecret, app.GetNamespacedName("%s"), secret)

	if err != nil {
		return "", err
	}

	jsonData, err := json.Marshal(ch.Config)
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
