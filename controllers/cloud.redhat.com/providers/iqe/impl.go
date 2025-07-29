package iqe

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	provutils "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/utils"

	"k8s.io/apimachinery/pkg/types"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

var DefaultImageIQESelenium = "quay.io/redhatqe/selenium-standalone"

var IqeSecret = rc.NewSingleResourceIdent("cji", "iqe_secret", &core.Secret{})
var VaultSecret = rc.NewSingleResourceIdent("cji", "vault_secret", &core.Secret{})
var IqeClowdJob = rc.NewSingleResourceIdent("cji", "iqe_clowdjob", &batchv1.Job{})
var ClowdJob = rc.NewMultiResourceIdent("cji", "clowdjob", &batchv1.Job{})

func joinNullableSlice(s *[]string) string {
	if s != nil {
		return strings.Join(*s, ",")
	}
	return ""
}

func updateEnvVars(existingEnvVars []core.EnvVar, newEnvVars []core.EnvVar) []core.EnvVar {
	for _, newEnvVar := range newEnvVars {
		if newEnvVar.Value == "" {
			// do not update value of an env var if the new value is empty
			continue
		}
		replaced := false
		for idx, existingEnvVar := range existingEnvVars {
			if existingEnvVar.Name == newEnvVar.Name {
				p := &existingEnvVars[idx]
				p.Value = newEnvVar.Value
				replaced = true
			}
		}
		if !replaced {
			existingEnvVars = append(existingEnvVars, newEnvVar)
		}
	}

	return existingEnvVars
}

func createIqeContainer(j *batchv1.Job, nn types.NamespacedName, cji *crd.ClowdJobInvocation, env *crd.ClowdEnvironment, app *crd.ClowdApp) *core.Container {
	// iqePlugins comes from ClowdApp spec unless overridden
	iqePlugins := app.Spec.Testing.IqePlugin
	if cji.Spec.Testing.Iqe.IqePlugins != "" {
		iqePlugins = cji.Spec.Testing.Iqe.IqePlugins
	}

	// default log level is "info" unless overridden
	logLevel := cji.Spec.Testing.Iqe.LogLevel
	if cji.Spec.Testing.Iqe.LogLevel == "" {
		logLevel = "info"
	}

	envVars := []core.EnvVar{
		{Name: "ENV_FOR_DYNACONF", Value: cji.Spec.Testing.Iqe.DynaconfEnvName},
		{Name: "NAMESPACE", Value: nn.Namespace},
		{Name: "CLOWDER_ENABLED", Value: "true"},
		{Name: "ACG_CONFIG", Value: "/cdapp/cdappconfig.json"},
		{Name: "IQE_PLUGINS", Value: iqePlugins},
		{Name: "IQE_MARKER_EXPRESSION", Value: cji.Spec.Testing.Iqe.Marker},
		{Name: "IQE_FILTER_EXPRESSION", Value: cji.Spec.Testing.Iqe.Filter},
		{Name: "IQE_LOG_LEVEL", Value: logLevel},
		{Name: "IQE_REQUIREMENTS", Value: joinNullableSlice(cji.Spec.Testing.Iqe.Requirements)},
		{Name: "IQE_REQUIREMENTS_PRIORITY", Value: joinNullableSlice(cji.Spec.Testing.Iqe.RequirementsPriority)},
		{Name: "IQE_TEST_IMPORTANCE", Value: joinNullableSlice(cji.Spec.Testing.Iqe.TestImportance)},
		{Name: "IQE_PARALLEL_ENABLED", Value: cji.Spec.Testing.Iqe.ParallelEnabled},
		{Name: "IQE_PARALLEL_WORKER_COUNT", Value: cji.Spec.Testing.Iqe.ParallelWorkerCount},
		{Name: "IQE_RP_ARGS", Value: cji.Spec.Testing.Iqe.RpArgs},
		{Name: "IQE_IBUTSU_SOURCE", Value: cji.Spec.Testing.Iqe.IbutsuSource},
	}

	if cji.Spec.Testing.Iqe.Env != nil {
		envVars = updateEnvVars(envVars, *cji.Spec.Testing.Iqe.Env)
	}

	// set image tag
	iqeImage := env.Spec.Providers.Testing.Iqe.ImageBase

	tag := "latest"
	if app.Spec.Testing.IqePlugin != "" {
		// ClowdApp has an IQE Plugin defined, use that image tag by default
		tag = app.Spec.Testing.IqePlugin
	}
	if cji.Spec.Testing.Iqe.ImageTag != "" {
		// this CJI has specified an image tag override
		tag = cji.Spec.Testing.Iqe.ImageTag
	}

	args := []string{"clowder"}
	if cji.Spec.Testing.Iqe.Debug {
		args = []string{"container-debug"}
	}

	// create pod container
	pod := crd.PodSpec{Resources: env.Spec.Providers.Testing.Iqe.Resources}

	c := core.Container{
		Name:         j.Name,
		Image:        fmt.Sprintf("%s:%s", iqeImage, tag),
		Env:          envVars,
		Resources:    deployProvider.ProcessResources(&pod, env),
		VolumeMounts: []core.VolumeMount{},
		Args:         args,
		// Because the tags on iqe plugins are not commit based, we need to pull everytime we run.
		// A leftover tag from a previous run is never guaranteed to be up to date
		ImagePullPolicy:          core.PullAlways,
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: core.TerminationMessageReadFile,
	}

	// attach sel-downloads volume to access Downloads from selenim container in case selenium is deployed
	if cji.Spec.Testing.Iqe.UI.Selenium.Deploy {
		c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
			Name:      "sel-downloads",
			MountPath: "/sel-downloads",
		})
	}

	return &c
}

func createSeleniumContainer(j *batchv1.Job, cji *crd.ClowdJobInvocation, env *crd.ClowdEnvironment) *core.Container {
	// set image tag
	image := env.Spec.Providers.Testing.Iqe.UI.Selenium.ImageBase
	if image == "" {
		image = DefaultImageIQESelenium
	}
	tag := env.Spec.Providers.Testing.Iqe.UI.Selenium.DefaultImageTag
	if tag == "" {
		tag = "ff_102.9.0esr_chrome_112.0.5615.121"
	}

	// check if this CJI has specified a selenium image tag override
	if cji.Spec.Testing.Iqe.UI.Selenium.ImageTag != "" {
		tag = cji.Spec.Testing.Iqe.UI.Selenium.ImageTag
	}

	// create pod container
	pod := crd.PodSpec{Resources: env.Spec.Providers.Testing.Iqe.UI.Selenium.Resources}

	c := core.Container{
		Name:                     fmt.Sprintf("%s-%s", j.Name, "sel"),
		Image:                    fmt.Sprintf("%s:%s", image, tag),
		Resources:                deployProvider.ProcessResources(&pod, env),
		ImagePullPolicy:          core.PullIfNotPresent,
		TerminationMessagePath:   "/dev/termination-log",
		TerminationMessagePolicy: core.TerminationMessageReadFile,
	}

	// attach /dev/shm volume
	j.Spec.Template.Spec.Volumes = append(j.Spec.Template.Spec.Volumes, core.Volume{
		Name:         "shm",
		VolumeSource: core.VolumeSource{EmptyDir: &core.EmptyDirVolumeSource{Medium: "Memory"}},
	})

	c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
		Name:      "shm",
		MountPath: "/dev/shm",
	})

	// attach sel-downloads volume to share Downloads with iqe container
	sizeLimit := resource.MustParse("64Mi")
	j.Spec.Template.Spec.Volumes = append(j.Spec.Template.Spec.Volumes, core.Volume{
		Name: "sel-downloads",
		VolumeSource: core.VolumeSource{
			EmptyDir: &core.EmptyDirVolumeSource{
				Medium:    "Memory",
				SizeLimit: &sizeLimit,
			},
		},
	})

	c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
		Name:      "sel-downloads",
		MountPath: "/home/selenium/Downloads",
	})

	return &c
}

func attachConfigVolumes(ctx context.Context, c *core.Container, cache *rc.ObjectCache, cji *crd.ClowdJobInvocation, env *crd.ClowdEnvironment, app *crd.ClowdApp, j *batchv1.Job, logger logr.Logger, client client.Client) error {
	j.Spec.Template.Spec.Volumes = []core.Volume{}

	configAccess := env.Spec.Providers.Testing.ConfigAccess

	switch configAccess {
	// Build cdenvconfig.json and mount it
	case "environment":
		if secretErr := addIqeSecretToCache(ctx, cache, cji, app, logger, client); secretErr != nil {
			logger.Error(secretErr, "cannot add IQE secret to cache")
			return secretErr
		}
		c.VolumeMounts = append(c.VolumeMounts, core.VolumeMount{
			Name:      "cdenvconfig",
			MountPath: "/cdenv",
		})

		// Because the ns name now has a suffix attached, we need to specify
		// that the secret name does not include it (and can't because the
		// env creates the secret)
		secretName := cji.GetIQEName()

		j.Spec.Template.Spec.Volumes = append(j.Spec.Template.Spec.Volumes, core.Volume{
			Name: "cdenvconfig",
			VolumeSource: core.VolumeSource{
				Secret: &core.SecretVolumeSource{
					DefaultMode: utils.Int32Ptr(420),
					SecretName:  secretName,
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
					DefaultMode: utils.Int32Ptr(420),
					SecretName:  cji.Spec.AppName,
				},
			},
		})

	default:
		logger.Info("No config mounted to the iqe pod")
	}

	return nil
}

// CreateIqeJobResource will create a Job that contains a pod spec for running IQE
func CreateIqeJobResource(ctx context.Context, cache *rc.ObjectCache, cji *crd.ClowdJobInvocation, env *crd.ClowdEnvironment, app *crd.ClowdApp, nn types.NamespacedName, j *batchv1.Job, logger logr.Logger, client client.Client) error {
	labels := cji.GetLabels()
	cji.SetObjectMeta(j, crd.Name(nn.Name), crd.Labels(labels))

	j.Labels = labels
	j.Labels["job"] = cji.GetIQEName()
	j.Name = nn.Name
	j.Spec.Template.Labels = labels

	j.Spec.Template.Spec.RestartPolicy = core.RestartPolicyNever
	j.Spec.BackoffLimit = utils.Int32Ptr(0)

	// set service account for pod
	j.Spec.Template.Spec.ServiceAccountName = fmt.Sprintf("iqe-%s", app.Spec.EnvName)

	// build IQE container config
	iqeContainer := createIqeContainer(j, nn, cji, env, app)

	// apply vault env vars to container if vaultSecretRef exists in environment
	nullSecretRef := crd.NamespacedName{}
	vaultSecretRef := env.Spec.Providers.Testing.Iqe.VaultSecretRef
	if env.Spec.Providers.Testing.Iqe.VaultSecretRef != nullSecretRef {
		// copy vault secret into destination namespace
		vaultSecret, err := addVaultSecretToCache(ctx, cache, cji, vaultSecretRef, logger, client)
		if err != nil {
			logger.Error(err, "unable to add vault secret to cache")
			return err
		}
		vaultEnvVars := buildVaultEnvVars(vaultSecret)
		iqeContainer.Env = append(iqeContainer.Env, vaultEnvVars...)
	}

	// Mount volumes to the IQE container
	if err := attachConfigVolumes(ctx, iqeContainer, cache, cji, env, app, j, logger, client); err != nil {
		return err
	}

	containers := []core.Container{*iqeContainer}

	if cji.Spec.Testing.Iqe.UI.Selenium.Deploy {
		selContainer := createSeleniumContainer(j, cji, env)
		containers = append(containers, *selContainer)
	}

	j.Spec.Template.Spec.Containers = containers

	utils.UpdateAnnotations(&j.Spec.Template, provutils.KubeLinterAnnotations)
	utils.UpdateAnnotations(j, provutils.KubeLinterAnnotations)

	return nil
}

// buildVaultEnvVars creates vault env vars for the IQE pod that will map to keys defined in the vaultSecret
func buildVaultEnvVars(vaultSecret *core.Secret) []core.EnvVar {
	vaultEnvVars := []core.EnvVar{
		{Name: "DYNACONF_IQE_VAULT_LOADER_ENABLED", Value: "true"},
		{Name: "DYNACONF_IQE_VAULT_VERIFY", Value: "true"},
		{
			Name: "DYNACONF_IQE_VAULT_URL",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{Name: vaultSecret.Name},
					Key:                  "url",
					Optional:             utils.BoolPtr(true),
				},
			},
		},
		{
			Name: "DYNACONF_IQE_VAULT_MOUNT_POINT",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{Name: vaultSecret.Name},
					Key:                  "mountPoint",
					Optional:             utils.BoolPtr(true),
				},
			},
		},
		{
			Name: "DYNACONF_IQE_VAULT_ROLE_ID",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{Name: vaultSecret.Name},
					Key:                  "roleId",
					Optional:             utils.BoolPtr(true),
				},
			},
		},
		{
			Name: "DYNACONF_IQE_VAULT_SECRET_ID",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{Name: vaultSecret.Name},
					Key:                  "secretId",
					Optional:             utils.BoolPtr(true),
				},
			},
		},
		{
			Name: "DYNACONF_IQE_VAULT_GITHUB_TOKEN",
			ValueFrom: &core.EnvVarSource{
				SecretKeyRef: &core.SecretKeySelector{
					LocalObjectReference: core.LocalObjectReference{Name: vaultSecret.Name},
					Key:                  "githubToken",
					Optional:             utils.BoolPtr(true),
				},
			},
		},
	}

	return vaultEnvVars
}

func addVaultSecretToCache(ctx context.Context, cache *rc.ObjectCache, cji *crd.ClowdJobInvocation, srcRef crd.NamespacedName, logger logr.Logger, client client.Client) (*core.Secret, error) {
	dstSecretRef := types.NamespacedName{
		Name:      fmt.Sprintf("%s-vault", cji.Name),
		Namespace: cji.Namespace,
	}

	// convert crd.NamespacedName to types.NamespacedName
	srcSecretRef := types.NamespacedName{
		Name:      srcRef.Name,
		Namespace: srcRef.Namespace,
	}

	vaultSecret, err := utils.CopySecret(ctx, client, srcSecretRef, dstSecretRef)
	if err != nil {
		logger.Error(err, "unable to copy vault secret from source namespace")
		return nil, err
	}

	if err = cache.Create(VaultSecret, dstSecretRef, vaultSecret); err != nil {
		logger.Error(err, "Failed to add vault secret to cache")
		return nil, err
	}

	return vaultSecret, err
}

func addIqeSecretToCache(ctx context.Context, cache *rc.ObjectCache, cji *crd.ClowdJobInvocation, app *crd.ClowdApp, logger logr.Logger, client client.Client) error {
	iqeSecret := &core.Secret{}
	secretName := fmt.Sprintf("%s-iqe", cji.Name)

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
	iqeSecret.SetName(secretName)
	iqeSecret.SetNamespace(nn.Namespace)

	// This should maybe be owned by the job
	iqeSecret.SetOwnerReferences([]metav1.OwnerReference{cji.MakeOwnerReference()})

	// loop through secrets and get their appConfig
	envConfig := make(map[string]interface{})
	// because we want a list of appConfigs, we need to nest this under the envConfig
	appConfigs := make(map[string]config.AppConfig)
	for _, app := range appList.Items {
		appConfig, err := fetchConfig(ctx, types.NamespacedName{
			Name:      app.Name,
			Namespace: app.Namespace,
		}, logger, client)
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

func fetchConfig(ctx context.Context, name types.NamespacedName, logger logr.Logger, client client.Client) (config.AppConfig, error) {
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
