package makers

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const webPort = 8000
const privatePort = 10000

func defaultMetaObject() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      "reqapp",
		Namespace: "default",
		Labels: map[string]string{
			"clowdapp": "test",
		},
	}
}

func TestSingleDependency(t *testing.T) {

	var app crd.ClowdApp
	var apps crd.ClowdAppList

	objMeta := defaultMetaObject()

	app = crd.ClowdApp{
		ObjectMeta: objMeta,
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{
				"bopper",
			},
			Deployments: []crd.Deployment{{
				Name: "reqapp",
			}},
		},
	}

	nobjMeta := objMeta
	nobjMeta.Name = "bopper"
	nobjMeta.Namespace = "bopperspace"
	apps = crd.ClowdAppList{
		Items: []crd.ClowdApp{{
			ObjectMeta: nobjMeta,
			Spec: crd.ClowdAppSpec{
				Deployments: []crd.Deployment{{
					WebServices: crd.WebServices{
						Private: crd.PrivateWebService{
							Enabled: true,
						},
						Public: crd.PublicWebService{
							Enabled: true,
						},
					},
					Name: "bopper",
				}},
			},
		}},
	}

	deps := []config.DependencyEndpoint{}
	privDeps := []config.PrivateDependencyEndpoint{}

	missing := makeDepConfig(&deps, &privDeps, webPort, privatePort, &app, &apps)

	if len(missing) > 0 {
		t.Errorf("We got a missing dep when there shouldn't have been one")
	}

	if deps[0].Hostname != "bopper-bopper.bopperspace.svc" {
		t.Errorf("We didn't get the right service hostname, got %v should be %v", deps[0].Hostname, "bopper-bopper.bopperspace.svc")
	}
	if deps[0].Port != 8000 {
		t.Errorf("We didn't get the right service port")
	}
	if deps[0].Name != "bopper" {
		t.Errorf("We didn't get the right service name, got %v should be %v", deps[0].Name, "bopper")
	}

	if privDeps[0].Hostname != "bopper-bopper.bopperspace.svc" {
		t.Errorf("We didn't get the right service hostname, got %v should be %v", privDeps[1].Hostname, "bopper-bopper.bopperspace.svc")
	}
	if privDeps[0].Port != 10000 {
		t.Errorf("We didn't get the right service port")
	}
	if privDeps[0].Name != "bopper" {
		t.Errorf("We didn't get the right service name, got %v should be %v", privDeps[1].Name, "bopper")
	}
}

func TestMissingDependency(t *testing.T) {

	var app crd.ClowdApp
	var apps crd.ClowdAppList

	objMeta := defaultMetaObject()

	app = crd.ClowdApp{
		ObjectMeta: objMeta,
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{
				"bopper",
			},
			Deployments: []crd.Deployment{{
				Name: "reqapp",
			}},
			OptionalDependencies: []string{
				"marvin",
			},
		},
	}

	nobjMeta := objMeta
	nobjMeta.Name = "bopper"
	nobjMeta.Namespace = "bopperspace"
	apps = crd.ClowdAppList{}

	deps := []config.DependencyEndpoint{}
	privDeps := []config.PrivateDependencyEndpoint{}

	missing := makeDepConfig(&deps, &privDeps, webPort, privatePort, &app, &apps)

	if len(privDeps) > 0 {
		t.Errorf("We got private deps we shouldn't have")
	}

	if len(deps) > 0 {
		t.Errorf("We got deps when we shouldn't have")
	}

	if missing[0] != "bopper" {
		t.Errorf("Didn't get the right missing dep")
	}
}

func TestOptionalDependency(t *testing.T) {

	var app crd.ClowdApp
	var apps crd.ClowdAppList

	objMeta := defaultMetaObject()

	app = crd.ClowdApp{
		ObjectMeta: objMeta,
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{
				"zaphod",
			},
			OptionalDependencies: []string{
				"marvin",
			},
			Deployments: []crd.Deployment{{
				Name: "reqapp",
			}},
		},
	}

	nobjMeta := objMeta
	nobjMeta.Name = "marvin"
	nobjMeta.Namespace = "android"
	nobjMeta2 := objMeta
	nobjMeta2.Name = "zaphod"
	nobjMeta2.Namespace = "android"
	apps = crd.ClowdAppList{
		Items: []crd.ClowdApp{{
			ObjectMeta: nobjMeta,
			Spec: crd.ClowdAppSpec{
				Deployments: []crd.Deployment{{
					Web:  true,
					Name: "deep",
				}}},
		},
			{
				ObjectMeta: nobjMeta2,
				Spec: crd.ClowdAppSpec{
					Deployments: []crd.Deployment{{
						Web:  true,
						Name: "beeble",
					}}},
			},
		},
	}

	deps := []config.DependencyEndpoint{}
	privDeps := []config.PrivateDependencyEndpoint{}

	makeDepConfig(&deps, &privDeps, webPort, privatePort, &app, &apps)

	if len(privDeps) > 0 {
		t.Errorf("We got private deps we shouldn't have")
	}

	if len(deps) != 2 {
		t.Errorf("We didn't get the dependency")
	}

	if deps[0].App != "zaphod" {
		t.Errorf("We didn't get the right dependency")
	}

	if deps[1].App != "marvin" {
		t.Errorf("We didn't get the right dependency")
	}
}

func assertAppConfig(t *testing.T, cfg config.DependencyEndpoint, hostname string, port int, name string, app string) {
	if cfg.Hostname != hostname {
		t.Errorf("We didn't get the right service hostname")
	}
	if cfg.Port != port {
		t.Errorf("We didn't get the right service port")
	}
	if cfg.Name != name {
		t.Errorf("We didn't get the right service name")
	}
	if cfg.App != app {
		t.Errorf("We didn't get the right app name")
	}
}

func TestMultiDependency(t *testing.T) {

	var app crd.ClowdApp
	var apps crd.ClowdAppList

	objMeta := defaultMetaObject()

	app = crd.ClowdApp{
		ObjectMeta: objMeta,
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{
				"bopper",
				"snapper",
			},
			Deployments: []crd.Deployment{{
				Name: "service",
			}},
		},
	}

	nobjMeta := objMeta
	nobjMeta.Name = "bopper"
	nobjMeta.Namespace = "bopperspace"
	n2objMeta := objMeta
	n2objMeta.Name = "snapper"
	n2objMeta.Namespace = "snapperspace"
	apps = crd.ClowdAppList{
		Items: []crd.ClowdApp{
			{
				ObjectMeta: n2objMeta,
				Spec: crd.ClowdAppSpec{
					Deployments: []crd.Deployment{
						{
							Name: "whopper",
							Web:  true,
						},
					},
				}},
			{
				ObjectMeta: nobjMeta,
				Spec: crd.ClowdAppSpec{
					Deployments: []crd.Deployment{
						{
							Name: "chopper",
							Web:  true,
						},
						{
							Name: "bopper",
							Web:  true,
						},
					},
				},
			},
		},
	}

	deps := []config.DependencyEndpoint{}
	privDeps := []config.PrivateDependencyEndpoint{}

	missing := makeDepConfig(&deps, &privDeps, webPort, privatePort, &app, &apps)

	if len(privDeps) > 0 {
		t.Errorf("We got private deps we shouldn't have")
	}

	if len(missing) > 0 {
		t.Errorf("We got a missing dep error")
	}

	assertAppConfig(t, deps[0], "bopper-chopper.bopperspace.svc", 8000, "chopper", "bopper")
	assertAppConfig(t, deps[1], "bopper-bopper.bopperspace.svc", 8000, "bopper", "bopper")
	assertAppConfig(t, deps[2], "snapper-whopper.snapperspace.svc", 8000, "whopper", "snapper")

	if len(deps) != 3 {
		t.Errorf("Wrong number of dep services")
	}
}

type Params map[string]map[string]string
type IDAndParams struct {
	Params Params
	ID     string
}

func NewIDAndParam(ID, limitCPU, limitMemory, requestsCPU, requestsMemory string) IDAndParams {
	return IDAndParams{
		ID: ID,
		Params: Params{
			"limits": {
				"cpu":    limitCPU,
				"memory": limitMemory,
			},
			"requests": {
				"cpu":    requestsCPU,
				"memory": requestsMemory,
			},
		},
	}
}

func createResourceRequirements(params Params) core.ResourceRequirements {
	rl := core.ResourceList{}
	limits := params["limits"]

	if limits != nil {
		if limits["cpu"] != "" {
			rl["cpu"] = resource.MustParse(limits["cpu"])
		}
		if limits["memory"] != "" {
			rl["memory"] = resource.MustParse(limits["memory"])
		}
	}

	rr := core.ResourceList{}
	requests := params["requests"]
	if requests != nil {
		if requests["cpu"] != "" {
			rr["cpu"] = resource.MustParse(requests["cpu"])
		}
		if requests["memory"] != "" {
			rr["memory"] = resource.MustParse(requests["memory"])
		}
	}

	return core.ResourceRequirements{
		Limits:   rl,
		Requests: rr,
	}
}

func setupResourcesForTest(params Params) (*apps.Deployment, *crd.ClowdEnvironment, *crd.ClowdApp) {
	var app crd.ClowdApp

	objMeta := defaultMetaObject()

	appResources := createResourceRequirements(params)

	app = crd.ClowdApp{
		ObjectMeta: objMeta,
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{
				"bopper",
				"snapper",
			},
			Deployments: []crd.Deployment{{
				Name: "reqapp",
				PodSpec: crd.PodSpec{
					Resources: appResources,
				},
			}},
		},
	}

	envResourceParams := Params{
		"limits": {
			"cpu":    "300m",
			"memory": "1024Mi",
		},
		"requests": {
			"cpu":    "30m",
			"memory": "512Mi",
		},
	}

	env := crd.ClowdEnvironment{
		Spec: crd.ClowdEnvironmentSpec{
			ResourceDefaults: createResourceRequirements(envResourceParams),
		},
	}

	d := &apps.Deployment{}

	return d, &env, &app
}

func TestResourceDefaults(t *testing.T) {

	envOptions := []IDAndParams{
		NewIDAndParam("Limit CPU", "40m", "", "", ""),
		NewIDAndParam("Limit Memory", "", "40Mi", "", ""),
		NewIDAndParam("Request CPU", "", "", "40m", ""),
		NewIDAndParam("Request Memory", "", "", "", "40Mi"),
		NewIDAndParam("Full Override", "40m", "40Mi", "40M", "40Mi"),
	}

	for _, tt := range envOptions {
		t.Run(tt.ID, func(t *testing.T) {

			d, env, app := setupResourcesForTest(tt.Params)

			appResources := app.Spec.Deployments[0].PodSpec.Resources

			nn := types.NamespacedName{
				Name:      app.Spec.Deployments[0].Name,
				Namespace: app.Namespace,
			}

			initDeployment(app, env, d, nn, app.Spec.Deployments[0], "hihi")

			var expectedLimitCPU, expectedLimitMemory, expectedRequestsCPU, expectedRequestsMemory resource.Quantity

			appLimits := tt.Params["limits"]
			appRequests := tt.Params["requests"]
			envLimits := env.Spec.ResourceDefaults.Limits
			envRequests := env.Spec.ResourceDefaults.Requests

			if appLimits["cpu"] != "" {
				expectedLimitCPU = *appResources.Limits.Cpu()
			} else {
				expectedLimitCPU = envLimits["cpu"]
			}
			if appLimits["memory"] != "" {
				expectedLimitMemory = *appResources.Limits.Memory()
			} else {
				expectedLimitMemory = envLimits["memory"]
			}

			if appRequests["cpu"] != "" {
				expectedRequestsCPU = *appResources.Requests.Cpu()
			} else {
				expectedRequestsCPU = envRequests["cpu"]
			}
			if appRequests["memory"] != "" {
				expectedRequestsMemory = *appResources.Requests.Memory()
			} else {
				expectedRequestsMemory = envRequests["memory"]
			}

			containerLimits := d.Spec.Template.Spec.Containers[0].Resources.Limits
			containerRequests := d.Spec.Template.Spec.Containers[0].Resources.Requests

			if *containerLimits.Cpu() != expectedLimitCPU {
				t.Errorf("Resource Limit for CPU was not set %v did not equal %v", containerLimits.Cpu(), expectedLimitCPU)
			}
			if *containerLimits.Memory() != expectedLimitMemory {
				t.Errorf("Resource Limit for Memory was not set %v did not equal %v", containerLimits.Memory(), expectedLimitMemory)
			}
			if *containerRequests.Cpu() != expectedRequestsCPU {
				t.Errorf("Resource Requests for CPU was not set %v did not equal %v", containerRequests.Cpu(), expectedRequestsCPU)
			}
			if *containerRequests.Memory() != expectedRequestsMemory {
				t.Errorf("Resource Requests for Memory was not set %v did not equal %v", containerRequests.Memory(), expectedRequestsMemory)
			}
		})
	}
}
