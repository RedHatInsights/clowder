package iqe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	deployProvider "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers/deployment"

	"k8s.io/apimachinery/pkg/types"
)

var IqeSecret = providers.NewSingleResourceIdent("cji", "iqe_secret", &core.Secret{})

func CreateIqeJobResource(cache *providers.ObjectCache, cji *crd.ClowdJobInvocation, env *crd.ClowdEnvironment, app *crd.ClowdApp, nn types.NamespacedName, ctx context.Context, j *batchv1.Job, logger logr.Logger, client client.Client) error {
	labels := cji.GetLabels()
	cji.SetObjectMeta(j, crd.Name(nn.Name), crd.Labels(labels))

	j.ObjectMeta.Labels = labels
	j.Spec.Template.ObjectMeta.Labels = labels

	j.Spec.Template.Spec.RestartPolicy = core.RestartPolicyNever

	j.Spec.Template.Spec.ImagePullSecrets = []core.LocalObjectReference{
		{Name: "quay-cloudservices-pull"},
	}

	pod := crd.PodSpec{
		Resources: env.Spec.Providers.Testing.Iqe.Resources,
	}

	envvar := []core.EnvVar{
		{Name: "ACG_CONFIG", Value: "/cdapp/cdappconfig.json"},
		{Name: "ENV_FOR_DYNACONF", Value: cji.Spec.Testing.Iqe.DynaconfEnvName},
		{Name: "NAMESPACE", Value: nn.Namespace},
		{Name: "CLOWDER_ENABLED", Value: "true"},
	}

	tag := ""
	if cji.Spec.Testing.Iqe.ImageTag != "" {
		tag = cji.Spec.Testing.Iqe.ImageTag
	} else {
		tag = app.Spec.Testing.IqePlugin
	}
	plugin := app.Spec.Testing.IqePlugin
	iqeImage := env.Spec.Providers.Testing.Iqe.ImageBase

	constructedIqeCommand, err := ConstructIqeCommand(cji, plugin)
	if err != nil {
		return err
	}

	j.Spec.Template.Spec.ServiceAccountName = fmt.Sprintf("iqe-%s", app.Spec.EnvName)

	c := core.Container{
		Name:         nn.Name,
		Image:        fmt.Sprintf("%s:%s", iqeImage, tag),
		Command:      constructedIqeCommand,
		Env:          envvar,
		Resources:    deployProvider.ProcessResources(&pod, env),
		VolumeMounts: []core.VolumeMount{},
		// Because the tags on iqe plugins are not commit based, we need to pull everytime we run.
		// A leftover tag from a previous run is never guaranteed to be up to date
		ImagePullPolicy: core.PullAlways,
	}

	if cji.Spec.Debug {
		c.Name = fmt.Sprintf("%s-debug", nn.Name)
		c.Command = []string{"/bin/sh", "-c", "while true; do sleep 1; done"}
		if len(cji.Spec.DebugCommand) > 0 {
			c.Command = cji.Spec.DebugCommand
		}
		c.Args = []string{}
	}

	j.Spec.Template.Spec.Volumes = []core.Volume{}
	configAccess := env.Spec.Providers.Testing.ConfigAccess

	switch configAccess {
	// Build cdenvconfig.json and mount it
	case "environment":
		if secretErr := createAndApplyIqeSecret(cache, ctx, cji, app, env.Name, logger, client); secretErr != nil {
			logger.Error(secretErr, "Cannot apply iqe secret")
			return secretErr
		}
		c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
			Name:      "cdenvconfig",
			MountPath: "/cdenv",
		})

		j.Spec.Template.Spec.Volumes = append(j.Spec.Template.Spec.Volumes, core.Volume{
			Name: "cdenvconfig",
			VolumeSource: core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					SecretName: nn.Name,
				},
			},
		})
		// if we have env access, we also want app access as well, so also run the next case
		fallthrough

	// mount cdappconfig
	case "app":
		c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
			Name:      "config-secret",
			MountPath: "/cdapp",
		})

		j.Spec.Template.Spec.Volumes = append(j.Spec.Template.Spec.Volumes, core.Volume{
			Name: "config-secret",
			VolumeSource: core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					SecretName: cji.Spec.AppName,
				},
			},
		})

	default:
		logger.Info("No config mounted to the iqe pod")
	}

	j.Spec.Template.Spec.Containers = []core.Container{c}

	return nil
}

func ConstructIqeCommand(cji *crd.ClowdJobInvocation, plugin string) ([]string, error) {
	if plugin == "" {
		return []string{}, errors.New("iqe-plugin is missing from ClowdApp")
	}

	command := []string{
		"iqe", "tests", "plugin",
		fmt.Sprintf("%v", strings.ReplaceAll(plugin, "-", "_")),
	}
	if cji.Spec.Testing.Iqe.Marker != "" {
		command = append(command, "-m", cji.Spec.Testing.Iqe.Marker)
	}
	if cji.Spec.Testing.Iqe.Filter != "" {
		// Note: go can append multiple values to a slice
		command = append(command, "-k", cji.Spec.Testing.Iqe.Filter)
	}
	return command, nil
}

func createAndApplyIqeSecret(cache *providers.ObjectCache, ctx context.Context, cji *crd.ClowdJobInvocation, app *crd.ClowdApp, envName string, logger logr.Logger, client client.Client) error {
	iqeSecret := &core.Secret{}

	appList := crd.ClowdAppList{}
	if err := crd.GetAppInSameEnv(ctx, client, app, &appList); err != nil {
		return err
	}

	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-iqe", cji.Name),
		Namespace: cji.Namespace,
	}

	if err := cache.Create(IqeSecret, nn, iqeSecret); err != nil {
		logger.Error(err, "Failed to check for iqe secret")
		return err
	}
	iqeSecret.SetName(nn.Name)
	iqeSecret.SetNamespace(nn.Namespace)

	// This should maybe be owned by the job
	iqeSecret.SetOwnerReferences([]metav1.OwnerReference{cji.MakeOwnerReference()})

	// loop through secrets and get their appConfig
	envConfig := make(map[string]interface{})
	// because we want a list of appConfigs, we need to nest this under the envConfig
	appConfigs := make(map[string]config.AppConfig)
	for _, app := range appList.Items {
		appConfig, err := fetchConfig(types.NamespacedName{
			Name:      app.Name,
			Namespace: app.Namespace,
		}, ctx, logger, client)
		if err != nil {
			// r.Recorder.Eventf(&app, "Warning", "AppConfigMissing", "app config [%s] missing", app.Name)
			logger.Error(err, "Failed to fetch app config for app")
			return err
		}
		appConfigs[app.Name] = appConfig
	}
	envConfig["cdappconfigs"] = appConfigs

	// Marshall the data into the top level "cdenvconfig.json" to be mounted as a single secret
	// with the appconfigs list embedded
	envData, err := json.Marshal(envConfig)
	if err != nil {
		logger.Error(err, "Failed to marshal iqe secret")
		return err
	}

	// and finally cast all these configs and create the secret
	cdEnv := make(map[string][]byte)
	cdEnv["cdenvconfig.json"] = envData
	iqeSecret.Data = cdEnv
	if err := cache.Update(IqeSecret, iqeSecret); err != nil {
		logger.Error(err, "Failed to check for iqe secret")
		return err
	}

	return nil
}

func fetchConfig(name types.NamespacedName, ctx context.Context, logger logr.Logger, client client.Client) (config.AppConfig, error) {
	secretConfig := core.Secret{}
	cfg := config.AppConfig{}

	if err := client.Get(ctx, name, &secretConfig); err != nil {
		logger.Error(err, "Failed to get app secret")
		// r.Recorder.Eventf(&secretConfig, "Warning", "SecretMissing", "secret [%s] missing", name)
		return cfg, err
	}

	if err := json.Unmarshal(secretConfig.Data["cdappconfig.json"], &cfg); err != nil {
		logger.Error(err, "Could not unmarshall json for cdappconfig")
		// r.Recorder.Eventf(&secretConfig, "Warning", "UnmarshallError", "app config [%s] not unmarshalled", name)
		return cfg, err
	}

	return cfg, nil
}
