package confighash

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (ch *confighashProvider) updateHashCacheConfigMap(nn types.NamespacedName, c *config.AppConfig) error {

	cfgmap := &core.ConfigMap{}

	if err := ch.Client.Get(ch.Ctx, nn, cfgmap); err != nil {
		ch.Log.Info("configmap not present, skipping inclusion")
		return nil
		//return "", errors.Wrap(fmt.Sprintf("%v - %v", nn, volume), err)
	}

	if cfgmap.GetLabels()["watch"] != "me" {
		return nil
	}

	jsonData, err := json.Marshal(cfgmap.Data)
	if err != nil {
		return errors.Wrap("failed to marshal configmap JSON", err)
	}

	h := sha256.New()
	h.Write([]byte(jsonData))
	hash := fmt.Sprintf("%x", h.Sum(nil))
	c.HashCache = append(c.HashCache, fmt.Sprintf("cm-%s-%s", nn.Name, hash))
	return nil
}

func (ch *confighashProvider) updateHashCacheSecret(nn types.NamespacedName, c *config.AppConfig) error {

	secret := &core.Secret{}

	if err := ch.Client.Get(ch.Ctx, nn, secret); err != nil {
		ch.Log.Info("secret not present, skipping inclusion")
		return nil
		//return "", errors.Wrap(fmt.Sprintf("%v - %v", nn, volume), err)
	}

	if secret.GetLabels()["watch"] != "me" {
		return nil
	}

	jsonData, err := json.Marshal(secret.Data)
	if err != nil {
		return errors.Wrap("failed to marshal secret JSON", err)
	}

	h := sha256.New()
	h.Write([]byte(jsonData))
	hash := fmt.Sprintf("%x", h.Sum(nil))
	c.HashCache = append(c.HashCache, fmt.Sprintf("sc-%s-%s", nn.Name, hash))
	return nil
}

func (ch *confighashProvider) persistConfig(app *crd.ClowdApp) (string, error) {

	// In any case, we want to overwrite the secret, so this just
	// tests to see if the secret exists
	secret := &core.Secret{}
	err := ch.Cache.Create(CoreConfigSecret, app.GetNamespacedName("%s"), secret)

	if err != nil {
		return "", err
	}

	dList := apps.DeploymentList{}
	if err := ch.Cache.List(deployProvider.CoreDeployment, &dList); err != nil {
		return "", err
	}

	ch.Config.HashCache = []string{}

	for _, deployment := range dList.Items {
		for _, cont := range deployment.Spec.Template.Spec.Containers {
			for _, env := range cont.Env {
				if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
					nn := types.NamespacedName{
						Name:      env.ValueFrom.ConfigMapKeyRef.Name,
						Namespace: app.Namespace,
					}
					ch.updateHashCacheConfigMap(nn, ch.Config)
				}
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
					nn := types.NamespacedName{
						Name:      env.ValueFrom.SecretKeyRef.Name,
						Namespace: app.Namespace,
					}
					ch.updateHashCacheSecret(nn, ch.Config)
				}
			}
		}
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			if volume.ConfigMap != nil {
				nn := types.NamespacedName{
					Name:      volume.ConfigMap.Name,
					Namespace: app.Namespace,
				}
				ch.updateHashCacheConfigMap(nn, ch.Config)
			}
			if volume.Secret != nil {
				nn := types.NamespacedName{
					Name:      volume.Secret.SecretName,
					Namespace: app.Namespace,
				}
				ch.updateHashCacheSecret(nn, ch.Config)
			}
		}
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
