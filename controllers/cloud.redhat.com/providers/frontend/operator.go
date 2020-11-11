package frontend

import (
	"crypto/sha256"
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	obj "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	p "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/providers"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type localChrome struct {
	p.Provider
	Config config.FrontendConfig
}

func (c *localChrome) Configure(config *config.AppConfig) {
	config.Frontend = &c.Config
}

func (c *localChrome) CreateFrontend(app *crd.ClowdApp) error {
	c.Config.Frontends = map[string]string{}
	for _, spec := range app.Spec.Pods {
		if spec.Web.FrontendImage != "" {

			nn := types.NamespacedName{
				Name:      fmt.Sprintf("%v-frontend", spec.Name),
				Namespace: app.Namespace,
			}

			c.Config.Frontends[spec.Name] = fmt.Sprintf("%v.%v.svc", nn.Name, nn.Namespace)
			dd := apps.Deployment{}
			exists, err := utils.UpdateOrErr(c.Client.Get(c.Ctx, nn, &dd))

			if err != nil {
				return err
			}

			makeLocalApp(&dd, nn, app, &spec)

			if err = exists.Apply(c.Ctx, c.Client, &dd); err != nil {
				return err
			}

			s := core.Service{}
			update, err := utils.UpdateOrErr(c.Client.Get(c.Ctx, nn, &s))

			if err != nil {
				return err
			}

			makeLocalService(&s, nn, app, &spec)

			if err = update.Apply(c.Ctx, c.Client, &s); err != nil {
				return err
			}

			return nil

		}
	}
	return nil
}

func NewChromeFrontend(p *providers.Provider) (FrontendProvider, error) {
	config := config.FrontendConfig{}

	frontendProvider := localChrome{Provider: *p, Config: config}

	if err := providers.MakeComponent(p.Ctx, p.Client, p.Env, "chrome", makeLocalChrome, false); err != nil {
		return &frontendProvider, err
	}

	if err := providers.MakeComponent(p.Ctx, p.Client, p.Env, "spandx", makeLocalSpandx, false); err != nil {
		return &frontendProvider, err
	}

	return &frontendProvider, nil
}

func createConfigMap(app *crd.ClowdApp, p *providers.Provider, c *config.AppConfig) error {

	appList := crd.ClowdAppList{}
	err := p.Client.List(p.Ctx, &appList)

	data := `//.const frontendHost = process.env.PLATFORM === 'linux' ? 'localhost' : 'host.docker.internal';
const frontendHost = 'advisor-frontend';
const chromefrontendHost = 'chrome2';
const SECTION = 'insights';
const APP_ID = 'advisor';
const routes = {};

const CATALOG_APP_ID = 'catalog';
const CATALOG_FRONTEND_PORT = 8003;
`
	for _, app := range appList.Items {
		if app.Spec.EnvName != p.Env.Name {
			continue
		}
		for _, pod := range app.Spec.Pods {
			if pod.Web.FrontendImage != "" {
				data += fmt.Sprintf("routes[`/apps/%v`] = { host: `http://%v:8080` };\n", pod.Name, fmt.Sprintf("%v-frontend.%v.svc", pod.Name, app.Namespace))
				data += fmt.Sprintf("routes[`/${SECTION}/%v`] = { host: `http://%v:8080` };\n", pod.Name, fmt.Sprintf("%v-frontend.%v.svc", pod.Name, app.Namespace))
			}
		}
		//data += fmt.Sprintf("routes[`/${SECTION}/%v`] = { host: `http://%v:8002` };\n"
	}
	data += fmt.Sprintf("routes[`/apps/chrome`] = { host: `http://%v-chrome.%v.svc:8080` };\n", p.Env.Name, p.Env.Spec.TargetNamespace)
	data += "routes[`/beta/apps/${APP_ID}`] = { host: `http://${frontendHost}:8002` };\n"
	data += "routes[`/beta/${SECTION}/${APP_ID}`] = { host: `http://${frontendHost}:8002` };\n"
	data += "routes[`/${SECTION}`] = { host: `http://${frontendHost}:8002` };\n"
	data += "routes[`/apps/${CATALOG_APP_ID}`] = { host: `https://${frontendHost}:${CATALOG_FRONTEND_PORT}` };\n"

	data += "module.exports = { routes };"

	configMap := core.ConfigMap{}

	nn := providers.GetNamespacedName(p.Env, "spandx")

	err = p.Client.Get(p.Ctx, nn, &configMap)

	update, err := utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	configMap.Name = nn.Name
	configMap.Namespace = nn.Namespace
	configMap.Data = map[string]string{
		"spandx.config.js": data,
	}

	if err = update.Apply(p.Ctx, p.Client, &configMap); err != nil {
		return err
	}

	dd := apps.Deployment{}
	nn = providers.GetNamespacedName(p.Env, "spandx")

	err = p.Client.Get(p.Ctx, nn, &dd)

	if err != nil {
		return err
	}

	h := sha256.New()
	h.Write([]byte(data))
	hash := fmt.Sprintf("%x", h.Sum(nil))
	annos := dd.Spec.Template.GetAnnotations()
	if annos == nil {
		annos = map[string]string{}
	}
	annos["spandx-config"] = hash
	dd.Spec.Template.SetAnnotations(annos)

	update, err = utils.UpdateOrErr(err)
	if err != nil {
		return err
	}

	update.Apply(p.Ctx, p.Client, &dd)

	if err != nil {
		return err
	}

	return nil
}

func makeLocalChrome(o obj.ClowdObject, dd *apps.Deployment, svc *core.Service, pvc *core.PersistentVolumeClaim, usePVC bool) {
	nn := providers.GetNamespacedName(o, "chrome")

	oneReplica := int32(1)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name
	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.ObjectMeta.Labels = labels
	dd.Spec.Replicas = &oneReplica

	// probeHandler := core.Handler{
	// 	Exec: &core.ExecAction{
	// 		Command: []string{
	// 			"redis-cli",
	// 		},
	// 	},
	// }

	// livenessProbe := core.Probe{
	// 	Handler:             probeHandler,
	// 	InitialDelaySeconds: 15,
	// 	TimeoutSeconds:      2,
	// }
	// readinessProbe := core.Probe{
	// 	Handler:             probeHandler,
	// 	InitialDelaySeconds: 45,
	// 	TimeoutSeconds:      2,
	// }

	dd.Spec.Template.Spec.Containers = []core.Container{{
		Name:  nn.Name,
		Image: (o.(*crd.ClowdEnvironment)).Spec.Providers.Web.ChromeImage,
		Env:   []core.EnvVar{},
		Ports: []core.ContainerPort{{
			Name:          "chrome",
			ContainerPort: 8080,
		}},
		// LivenessProbe:  &livenessProbe,
		// ReadinessProbe: &readinessProbe,
	}}

	servicePorts := []core.ServicePort{{
		Name:     "chrome",
		Port:     8080,
		Protocol: "TCP",
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o)
}

func makeLocalSpandx(o obj.ClowdObject, dd *apps.Deployment, svc *core.Service, pvc *core.PersistentVolumeClaim, usePVC bool) {
	nn := providers.GetNamespacedName(o, "spandx")

	oneReplica := int32(1)

	labels := o.GetLabels()
	labels["env-app"] = nn.Name
	labeler := utils.MakeLabeler(nn, labels, o)

	labeler(dd)

	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.ObjectMeta.Labels = labels
	dd.Spec.Replicas = &oneReplica

	// probeHandler := core.Handler{
	// 	Exec: &core.ExecAction{
	// 		Command: []string{
	// 			"redis-cli",
	// 		},
	// 	},
	// }

	// livenessProbe := core.Probe{
	// 	Handler:             probeHandler,
	// 	InitialDelaySeconds: 15,
	// 	TimeoutSeconds:      2,
	// }
	// readinessProbe := core.Probe{
	// 	Handler:             probeHandler,
	// 	InitialDelaySeconds: 45,
	// 	TimeoutSeconds:      2,
	// }

	volumeMounts := []core.VolumeMount{{
		Name:      "config-volume",
		MountPath: "/config/",
	}}

	dd.Spec.Template.Spec.Containers = []core.Container{{
		Name:  nn.Name,
		Image: (o.(*crd.ClowdEnvironment)).Spec.Providers.Web.SpandxImage,
		Env:   []core.EnvVar{},
		Ports: []core.ContainerPort{{
			Name:          "spandx",
			ContainerPort: 1337,
		}},
		VolumeMounts: volumeMounts,

		// LivenessProbe:  &livenessProbe,
		// ReadinessProbe: &readinessProbe,
	}}

	YesMan := true
	dd.Spec.Template.Spec.Volumes = []core.Volume{}
	dd.Spec.Template.Spec.Volumes = append(dd.Spec.Template.Spec.Volumes, core.Volume{
		Name: "config-volume",
		VolumeSource: core.VolumeSource{
			ConfigMap: &core.ConfigMapVolumeSource{
				LocalObjectReference: core.LocalObjectReference{
					Name: nn.Name,
				},
				Optional: &YesMan,
			},
		},
	})

	servicePorts := []core.ServicePort{{
		Name:     "spandx",
		Port:     1337,
		Protocol: "TCP",
	}}

	utils.MakeService(svc, nn, labels, servicePorts, o)
}

func makeLocalApp(dd *apps.Deployment, nn types.NamespacedName, app *crd.ClowdApp, spec *crd.PodSpec) {

	oneReplica := int32(1)

	labels := app.GetLabels()
	labels["env-app"] = nn.Name
	labeler := utils.MakeLabeler(nn, labels, app)

	labeler(dd)

	dd.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
	dd.Spec.Template.ObjectMeta.Labels = labels
	dd.Spec.Replicas = &oneReplica

	// probeHandler := core.Handler{
	// 	Exec: &core.ExecAction{
	// 		Command: []string{
	// 			"redis-cli",
	// 		},
	// 	},
	// }

	// livenessProbe := core.Probe{
	// 	Handler:             probeHandler,
	// 	InitialDelaySeconds: 15,
	// 	TimeoutSeconds:      2,
	// }
	// readinessProbe := core.Probe{
	// 	Handler:             probeHandler,
	// 	InitialDelaySeconds: 45,
	// 	TimeoutSeconds:      2,
	// }

	dd.Spec.Template.Spec.Containers = []core.Container{{
		Name:  nn.Name,
		Image: spec.Web.FrontendImage,
		Env:   []core.EnvVar{},
		Ports: []core.ContainerPort{{
			Name:          "frontend",
			ContainerPort: 8080,
		}},
		// LivenessProbe:  &livenessProbe,
		// ReadinessProbe: &readinessProbe,
	}}
}

func makeLocalService(s *core.Service, nn types.NamespacedName, app *crd.ClowdApp, spec *crd.PodSpec) {
	servicePorts := []core.ServicePort{{
		Name:     "frontend",
		Port:     8080,
		Protocol: "TCP",
	}}
	utils.MakeService(s, nn, p.Labels{"env-app": nn.Name}, servicePorts, app)
}
