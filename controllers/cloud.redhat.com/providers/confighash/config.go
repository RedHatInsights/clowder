package confighash

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	k8serr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (ch *confighashProvider) envConfigMap(app *crd.ClowdApp, env core.EnvVar) error {
	if env.ValueFrom == nil {
		return nil
	}
	if env.ValueFrom.ConfigMapKeyRef == nil {
		return nil
	}
	nn := types.NamespacedName{
		Name:      env.ValueFrom.ConfigMapKeyRef.Name,
		Namespace: app.Namespace,
	}
	if nn.Name == app.Name {
		return nil
	}
	cf := &core.ConfigMap{}
	if err := ch.Client.Get(ch.Ctx, nn, cf); err != nil {
		if k8serr.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("could not get env configmap: %w", err)
	}
	_, err := ch.HashCache.CreateOrUpdateObject(cf, false)
	if err != nil {
		return nil
	}

	return ch.HashCache.AddClowdObjectToObject(app, cf)
}

func (ch *confighashProvider) envSecret(app *crd.ClowdApp, env core.EnvVar) error {
	if env.ValueFrom == nil {
		return nil
	}
	if env.ValueFrom.SecretKeyRef == nil {
		return nil
	}
	nn := types.NamespacedName{
		Name:      env.ValueFrom.SecretKeyRef.Name,
		Namespace: app.Namespace,
	}
	if nn.Name == app.Name {
		return nil
	}
	sec := &core.Secret{}
	if err := ch.Client.Get(ch.Ctx, nn, sec); err != nil {
		if k8serr.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("could not get env secret: %w", err)
	}
	_, err := ch.HashCache.CreateOrUpdateObject(sec, false)
	if err != nil {
		return nil
	}
	return ch.HashCache.AddClowdObjectToObject(app, sec)
}

func (ch *confighashProvider) volConfigMap(app *crd.ClowdApp, volume core.Volume) error {
	if volume.ConfigMap == nil {
		return nil
	}
	nn := types.NamespacedName{
		Name:      volume.ConfigMap.Name,
		Namespace: app.Namespace,
	}
	if nn.Name == app.Name {
		return nil
	}
	cf := &core.ConfigMap{}
	if err := ch.Client.Get(ch.Ctx, nn, cf); err != nil {
		if k8serr.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("could not get vol configmap: %w", err)
	}
	_, err := ch.HashCache.CreateOrUpdateObject(cf, false)
	if err != nil {
		return nil
	}
	return ch.HashCache.AddClowdObjectToObject(app, cf)
}

func (ch *confighashProvider) volSecret(app *crd.ClowdApp, volume core.Volume) error {
	if volume.Secret == nil {
		return nil
	}
	nn := types.NamespacedName{
		Name:      volume.Secret.SecretName,
		Namespace: app.Namespace,
	}
	if nn.Name == app.Name {
		return nil
	}
	sec := &core.Secret{}
	if err := ch.Client.Get(ch.Ctx, nn, sec); err != nil {
		if k8serr.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("could not get vol secret: %w", err)
	}
	_, err := ch.HashCache.CreateOrUpdateObject(sec, false)
	if err != nil {
		return nil
	}
	return ch.HashCache.AddClowdObjectToObject(app, sec)
}

func (ch *confighashProvider) iterateEnvVars(app *crd.ClowdApp, deployment apps.Deployment) error {
	for _, cont := range deployment.Spec.Template.Spec.Containers {
		for _, env := range cont.Env {
			if err := ch.envConfigMap(app, env); err != nil {
				return err
			}
			if err := ch.envSecret(app, env); err != nil {
				return err
			}
		}
	}

	return nil
}

func (ch *confighashProvider) iterateVolumes(app *crd.ClowdApp, deployment apps.Deployment) error {
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if err := ch.volConfigMap(app, volume); err != nil {
			return err
		}
		if err := ch.volSecret(app, volume); err != nil {
			return err
		}
	}

	return nil
}

func (ch *confighashProvider) updateHashCache(dList *apps.DeploymentList, app *crd.ClowdApp) error {
	for _, deployment := range dList.Items {
		deploy := deployment
		if err := ch.iterateEnvVars(app, deploy); err != nil {
			return err
		}
		if err := ch.iterateVolumes(app, deploy); err != nil {
			return err
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

	ch.Config.HashCache = utils.StringPtr(
		fmt.Sprintf(
			"%s%s",
			ch.HashCache.GetSuperHashForClowdObject(app),
			ch.HashCache.GetSuperHashForClowdObject(ch.Env),
		),
	)

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
