package makers

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
)

const webPort = 8000

func TestSingleDependency(t *testing.T) {

	var app crd.ClowdApp
	var apps crd.ClowdAppList

	objMeta := metav1.ObjectMeta{
		Name:      "reqapp",
		Namespace: "default",
		Labels: map[string]string{
			"app": "test",
		},
	}

	app = crd.ClowdApp{
		ObjectMeta: objMeta,
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{
				"bopper",
			},
			Pods: []crd.PodSpec{{
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
				Pods: []crd.PodSpec{{
					Web:  true,
					Name: "bopper",
				}}},
		},
		},
	}

	config, missing := makeDepConfig(webPort, &app, &apps)

	if len(missing) > 0 {
		t.Errorf("We got a missing dep when there shouldn't have been one")
	}

	if config[0].Hostname != "bopper.bopperspace.svc" {
		t.Errorf("We didn't get the right service hostname")
	}
	if config[0].Port != 8000 {
		t.Errorf("We didn't get the right service port")
	}
	if config[0].Name != "bopper" {
		t.Errorf("We didn't get the right service name")
	}
}

func TestMissingDependency(t *testing.T) {

	var app crd.ClowdApp
	var apps crd.ClowdAppList

	objMeta := metav1.ObjectMeta{
		Name:      "reqapp",
		Namespace: "default",
		Labels: map[string]string{
			"app": "test",
		},
	}

	app = crd.ClowdApp{
		ObjectMeta: objMeta,
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{
				"bopper",
			},
			Pods: []crd.PodSpec{{
				Name: "reqapp",
			}},
		},
	}

	nobjMeta := objMeta
	nobjMeta.Name = "bopper"
	nobjMeta.Namespace = "bopperspace"
	apps = crd.ClowdAppList{}

	deps, missing := makeDepConfig(webPort, &app, &apps)

	if len(deps) > 0 {
		t.Errorf("We got deps when we shouldn't have")
	}

	if missing[0] != "bopper" {
		t.Errorf("Didn't get the right missing dep")
	}
}

func TestMultiDependency(t *testing.T) {

	var app crd.ClowdApp
	var apps crd.ClowdAppList

	objMeta := metav1.ObjectMeta{
		Name:      "reqapp",
		Namespace: "default",
		Labels: map[string]string{
			"app": "test",
		},
	}

	app = crd.ClowdApp{
		ObjectMeta: objMeta,
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{
				"bopper",
				"snapper",
			},
			Pods: []crd.PodSpec{{
				Name: "reqapp",
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
					Pods: []crd.PodSpec{{
						Web:  true,
						Name: "whopper",
					}}},
			},
			{
				ObjectMeta: nobjMeta,
				Spec: crd.ClowdAppSpec{
					Pods: []crd.PodSpec{
						{
							Web:  true,
							Name: "chopper",
						},
						{
							Web:  true,
							Name: "bopper",
						},
					},
				},
			},
		},
	}

	config, missing := makeDepConfig(webPort, &app, &apps)
	if len(missing) > 0 {
		t.Errorf("We got a missing dep error")
	}

	if config[0].Hostname != "chopper.bopperspace.svc" {
		t.Errorf("We didn't get the right service hostname")
	}
	if config[0].Port != 8000 {
		t.Errorf("We didn't get the right service port")
	}
	if config[0].Name != "chopper" {
		t.Errorf("We didn't get the right service name")
	}
	if config[0].App != "bopper" {
		t.Errorf("We didn't get the right app name")
	}

	if config[1].Hostname != "bopper.bopperspace.svc" {
		t.Errorf("We didn't get the right service hostname")
	}
	if config[1].Port != 8000 {
		t.Errorf("We didn't get the right service port")
	}
	if config[1].Name != "bopper" {
		t.Errorf("We didn't get the right service name")
	}
	if config[1].App != "bopper" {
		t.Errorf("We didn't get the right app name")
	}

	if config[2].Hostname != "whopper.snapperspace.svc" {
		t.Errorf("We didn't get the right service hostname")
	}
	if config[2].Port != 8000 {
		t.Errorf("We didn't get the right service port")
	}
	if config[2].Name != "whopper" {
		t.Errorf("We didn't get the right service name")
	}
	if config[2].App != "snapper" {
		t.Errorf("We didn't get the right app name")
	}

	if len(config) != 3 {
		t.Errorf("Wrong number of dep services")
	}
}
