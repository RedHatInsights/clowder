package deployment

import (
	"testing"

	"k8s.io/apimachinery/pkg/types"

	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"

	"k8s.io/apimachinery/pkg/api/resource"
)

func defaultMetaObject() metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      "reqapp",
		Namespace: "default",
		Labels: map[string]string{
			"app": "test",
		},
	}
}

type Params map[string]map[string]string
type IDAndParams struct {
	Params Params
	ID     string
}

func NewIDAndParam(id, limitCPU, limitMemory, requestsCPU, requestsMemory string) IDAndParams {
	return IDAndParams{
		ID: id,
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

			deployment := app.Spec.Deployments[0]

			err := initDeployment(app, env, d, nn, &deployment)
			if err != nil {
				t.Errorf("error was not nil")
			}

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
