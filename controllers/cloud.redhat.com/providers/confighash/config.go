package confighash

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (ch *confighashProvider) updateHashCache(dList *apps.DeploymentList, app *crd.ClowdApp) error {
	for _, deployment := range dList.Items {
		for _, cont := range deployment.Spec.Template.Spec.Containers {
			for _, env := range cont.Env {
				if env.ValueFrom != nil && env.ValueFrom.ConfigMapKeyRef != nil {
					nn := types.NamespacedName{
						Name:      env.ValueFrom.ConfigMapKeyRef.Name,
						Namespace: app.Namespace,
					}
					if nn.Name == app.Name {
						continue
					}
					cf := &core.ConfigMap{}
					if err := ch.Client.Get(ch.Ctx, nn, cf); err != nil {
						if env.ValueFrom.ConfigMapKeyRef.Optional != nil && *env.ValueFrom.ConfigMapKeyRef.Optional && k8serr.IsNotFound(err) {
							continue
						}
						return fmt.Errorf("could not get env configmap: %w", err)
					}
					if err := ch.HashCache.AddClowdObjectToObject(app, cf); err != nil {
						return err
					}
				}
				if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil {
					nn := types.NamespacedName{
						Name:      env.ValueFrom.SecretKeyRef.Name,
						Namespace: app.Namespace,
					}
					if nn.Name == app.Name {
						continue
					}
					sec := &core.Secret{}
					if err := ch.Client.Get(ch.Ctx, nn, sec); err != nil {
						if env.ValueFrom.SecretKeyRef.Optional != nil && *env.ValueFrom.SecretKeyRef.Optional && k8serr.IsNotFound(err) {
							continue
						}
						return fmt.Errorf("could not get env secret: %w", err)
					}
					if err := ch.HashCache.AddClowdObjectToObject(app, sec); err != nil {
						return err
					}
				}
			}
		}
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			if volume.ConfigMap != nil {
				nn := types.NamespacedName{
					Name:      volume.ConfigMap.Name,
					Namespace: app.Namespace,
				}
				if nn.Name == app.Name {
					continue
				}
				cf := &core.ConfigMap{}
				if err := ch.Client.Get(ch.Ctx, nn, cf); err != nil {
					if volume.ConfigMap.Optional != nil && *volume.ConfigMap.Optional && k8serr.IsNotFound(err) {
						continue
					}
					return fmt.Errorf("could not get vol configmap: %w", err)
				}
				if err := ch.HashCache.AddClowdObjectToObject(app, cf); err != nil {
					return err
				}
			}
			if volume.Secret != nil {
				nn := types.NamespacedName{
					Name:      volume.Secret.SecretName,
					Namespace: app.Namespace,
				}
				if nn.Name == app.Name {
					continue
				}
				sec := &core.Secret{}
				if err := ch.Client.Get(ch.Ctx, nn, sec); err != nil {
					if volume.Secret.Optional != nil && *volume.Secret.Optional && k8serr.IsNotFound(err) {
						continue
					}
					return fmt.Errorf("could not get vol secret: %w", err)
				}
				if err := ch.HashCache.AddClowdObjectToObject(app, sec); err != nil {
					return err
				}
			}
		}
	}
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

	if err := ch.updateHashCache(&dList, app); err != nil {
		return "", err
	}

	ch.Config.HashCache = ch.HashCache.GetSuperHashForClowdObject(app)
	ch.Config.HashCache += ch.HashCache.GetSuperHashForClowdObject(ch.Env)

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
