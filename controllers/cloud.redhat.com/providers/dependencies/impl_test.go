package dependencies

import (
	"testing"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestServesField_EmptyList_UsesLocalApp tests that when both ClowdApp and ClowdAppRef exist
// with an empty serves list, the local ClowdApp is used
func TestServesField_EmptyList_UsesLocalApp(t *testing.T) {
	app := crd.ClowdApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inventory",
			Namespace: "default",
		},
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{"rbac"},
			Deployments: []crd.Deployment{{
				Name: "api",
			}},
		},
	}

	// Local ClowdApp 'rbac'
	apps := crd.ClowdAppList{
		Items: []crd.ClowdApp{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rbac",
				Namespace: "rbac-ns",
			},
			Spec: crd.ClowdAppSpec{
				Deployments: []crd.Deployment{{
					Name: "service",
					WebServices: crd.WebServices{
						Public: crd.PublicWebService{
							Enabled: true,
						},
					},
				}},
			},
		}},
	}

	// ClowdAppRef 'rbac' with empty serves
	appRefs := &crd.ClowdAppRefList{
		Items: []crd.ClowdAppRef{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rbac",
				Namespace: "default",
			},
			Spec: crd.ClowdAppRefSpec{
				Serves: []string{}, // Empty serves list
				Deployments: []crd.ClowdAppRefDeployment{{
					Name:     "service",
					Hostname: "rbac.remote.example.com",
					WebServices: crd.WebServices{
						Public: crd.PublicWebService{
							Enabled: true,
						},
					},
				}},
			},
		}},
	}

	deps := []config.DependencyEndpoint{}
	privDeps := []config.PrivateDependencyEndpoint{}

	missing := makeDepConfig(&deps, &privDeps, makeWebCfg(), &app, &apps, appRefs)

	if len(missing) > 0 {
		t.Errorf("Expected no missing deps, got %v", missing)
	}

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	// Should use local ClowdApp hostname, not remote
	expectedHostname := "rbac-service.rbac-ns.svc"
	if deps[0].Hostname != expectedHostname {
		t.Errorf("Expected hostname %s (local ClowdApp), got %s", expectedHostname, deps[0].Hostname)
	}
}

// TestServesField_ConsumerInList_UsesAppRef tests that when the consumer is in the serves list,
// the ClowdAppRef is used instead of the local ClowdApp
func TestServesField_ConsumerInList_UsesAppRef(t *testing.T) {
	app := crd.ClowdApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inventory",
			Namespace: "default",
		},
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{"rbac"},
			Deployments: []crd.Deployment{{
				Name: "api",
			}},
		},
	}

	// Local ClowdApp 'rbac'
	apps := crd.ClowdAppList{
		Items: []crd.ClowdApp{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rbac",
				Namespace: "rbac-ns",
			},
			Spec: crd.ClowdAppSpec{
				Deployments: []crd.Deployment{{
					Name: "service",
					WebServices: crd.WebServices{
						Public: crd.PublicWebService{
							Enabled: true,
						},
					},
				}},
			},
		}},
	}

	// ClowdAppRef 'rbac' with 'inventory' in serves
	appRefs := &crd.ClowdAppRefList{
		Items: []crd.ClowdAppRef{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rbac",
				Namespace: "default",
			},
			Spec: crd.ClowdAppRefSpec{
				Serves: []string{"inventory"}, // inventory in serves list
				Deployments: []crd.ClowdAppRefDeployment{{
					Name:     "service",
					Hostname: "rbac.remote.example.com",
					WebServices: crd.WebServices{
						Public: crd.PublicWebService{
							Enabled: true,
						},
					},
				}},
			},
		}},
	}

	deps := []config.DependencyEndpoint{}
	privDeps := []config.PrivateDependencyEndpoint{}

	missing := makeDepConfig(&deps, &privDeps, makeWebCfg(), &app, &apps, appRefs)

	if len(missing) > 0 {
		t.Errorf("Expected no missing deps, got %v", missing)
	}

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	// Should use remote ClowdAppRef hostname
	expectedHostname := "rbac.remote.example.com"
	if deps[0].Hostname != expectedHostname {
		t.Errorf("Expected hostname %s (remote ClowdAppRef), got %s", expectedHostname, deps[0].Hostname)
	}
}

// TestServesField_ConsumerNotInList_UsesLocalApp tests that when the consumer is NOT in the serves list,
// the local ClowdApp is used
func TestServesField_ConsumerNotInList_UsesLocalApp(t *testing.T) {
	app := crd.ClowdApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inventory",
			Namespace: "default",
		},
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{"rbac"},
			Deployments: []crd.Deployment{{
				Name: "api",
			}},
		},
	}

	// Local ClowdApp 'rbac'
	apps := crd.ClowdAppList{
		Items: []crd.ClowdApp{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rbac",
				Namespace: "rbac-ns",
			},
			Spec: crd.ClowdAppSpec{
				Deployments: []crd.Deployment{{
					Name: "service",
					WebServices: crd.WebServices{
						Public: crd.PublicWebService{
							Enabled: true,
						},
					},
				}},
			},
		}},
	}

	// ClowdAppRef 'rbac' with 'other-app' in serves (not 'inventory')
	appRefs := &crd.ClowdAppRefList{
		Items: []crd.ClowdAppRef{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rbac",
				Namespace: "default",
			},
			Spec: crd.ClowdAppRefSpec{
				Serves: []string{"other-app"}, // Different app in serves
				Deployments: []crd.ClowdAppRefDeployment{{
					Name:     "service",
					Hostname: "rbac.remote.example.com",
					WebServices: crd.WebServices{
						Public: crd.PublicWebService{
							Enabled: true,
						},
					},
				}},
			},
		}},
	}

	deps := []config.DependencyEndpoint{}
	privDeps := []config.PrivateDependencyEndpoint{}

	missing := makeDepConfig(&deps, &privDeps, makeWebCfg(), &app, &apps, appRefs)

	if len(missing) > 0 {
		t.Errorf("Expected no missing deps, got %v", missing)
	}

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	// Should use local ClowdApp hostname
	expectedHostname := "rbac-service.rbac-ns.svc"
	if deps[0].Hostname != expectedHostname {
		t.Errorf("Expected hostname %s (local ClowdApp), got %s", expectedHostname, deps[0].Hostname)
	}
}

// TestServesField_OnlyAppRefExists_UsesAppRef tests that when only ClowdAppRef exists (no local ClowdApp),
// it is used regardless of serves field
func TestServesField_OnlyAppRefExists_UsesAppRef(t *testing.T) {
	app := crd.ClowdApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inventory",
			Namespace: "default",
		},
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{"rbac"},
			Deployments: []crd.Deployment{{
				Name: "api",
			}},
		},
	}

	// No local ClowdApp
	apps := crd.ClowdAppList{
		Items: []crd.ClowdApp{},
	}

	// Only ClowdAppRef exists with empty serves
	appRefs := &crd.ClowdAppRefList{
		Items: []crd.ClowdAppRef{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rbac",
				Namespace: "default",
			},
			Spec: crd.ClowdAppRefSpec{
				Serves: []string{}, // Empty serves
				Deployments: []crd.ClowdAppRefDeployment{{
					Name:     "service",
					Hostname: "rbac.remote.example.com",
					WebServices: crd.WebServices{
						Public: crd.PublicWebService{
							Enabled: true,
						},
					},
				}},
			},
		}},
	}

	deps := []config.DependencyEndpoint{}
	privDeps := []config.PrivateDependencyEndpoint{}

	missing := makeDepConfig(&deps, &privDeps, makeWebCfg(), &app, &apps, appRefs)

	if len(missing) > 0 {
		t.Errorf("Expected no missing deps, got %v", missing)
	}

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	// Should use ClowdAppRef since it's the only option
	expectedHostname := "rbac.remote.example.com"
	if deps[0].Hostname != expectedHostname {
		t.Errorf("Expected hostname %s (ClowdAppRef), got %s", expectedHostname, deps[0].Hostname)
	}
}

// TestServesField_OnlyLocalAppExists_UsesApp tests that when only local ClowdApp exists,
// it is used (current behavior)
func TestServesField_OnlyLocalAppExists_UsesApp(t *testing.T) {
	app := crd.ClowdApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inventory",
			Namespace: "default",
		},
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{"rbac"},
			Deployments: []crd.Deployment{{
				Name: "api",
			}},
		},
	}

	// Only local ClowdApp exists
	apps := crd.ClowdAppList{
		Items: []crd.ClowdApp{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rbac",
				Namespace: "rbac-ns",
			},
			Spec: crd.ClowdAppSpec{
				Deployments: []crd.Deployment{{
					Name: "service",
					WebServices: crd.WebServices{
						Public: crd.PublicWebService{
							Enabled: true,
						},
					},
				}},
			},
		}},
	}

	// No ClowdAppRef
	appRefs := &crd.ClowdAppRefList{
		Items: []crd.ClowdAppRef{},
	}

	deps := []config.DependencyEndpoint{}
	privDeps := []config.PrivateDependencyEndpoint{}

	missing := makeDepConfig(&deps, &privDeps, makeWebCfg(), &app, &apps, appRefs)

	if len(missing) > 0 {
		t.Errorf("Expected no missing deps, got %v", missing)
	}

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	// Should use local ClowdApp
	expectedHostname := "rbac-service.rbac-ns.svc"
	if deps[0].Hostname != expectedHostname {
		t.Errorf("Expected hostname %s (local ClowdApp), got %s", expectedHostname, deps[0].Hostname)
	}
}

// TestServesField_BothMissing_ReportsMissing tests that when neither ClowdApp nor ClowdAppRef exists,
// the dependency is reported as missing
func TestServesField_BothMissing_ReportsMissing(t *testing.T) {
	app := crd.ClowdApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inventory",
			Namespace: "default",
		},
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{"rbac"},
			Deployments: []crd.Deployment{{
				Name: "api",
			}},
		},
	}

	// No ClowdApp
	apps := crd.ClowdAppList{
		Items: []crd.ClowdApp{},
	}

	// No ClowdAppRef
	appRefs := &crd.ClowdAppRefList{
		Items: []crd.ClowdAppRef{},
	}

	deps := []config.DependencyEndpoint{}
	privDeps := []config.PrivateDependencyEndpoint{}

	missing := makeDepConfig(&deps, &privDeps, makeWebCfg(), &app, &apps, appRefs)

	if len(missing) != 1 {
		t.Fatalf("Expected 1 missing dep, got %d", len(missing))
	}

	if missing[0] != "rbac" {
		t.Errorf("Expected missing dep 'rbac', got '%s'", missing[0])
	}

	if len(deps) != 0 {
		t.Errorf("Expected no deps, got %d", len(deps))
	}
}

// TestServesField_OptionalDependency_SameLogic tests that optional dependencies follow the same
// serves logic as regular dependencies
func TestServesField_OptionalDependency_SameLogic(t *testing.T) {
	app := crd.ClowdApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inventory",
			Namespace: "default",
		},
		Spec: crd.ClowdAppSpec{
			OptionalDependencies: []string{"rbac"}, // Optional, not required
			Deployments: []crd.Deployment{{
				Name: "api",
			}},
		},
	}

	// Local ClowdApp 'rbac'
	apps := crd.ClowdAppList{
		Items: []crd.ClowdApp{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rbac",
				Namespace: "rbac-ns",
			},
			Spec: crd.ClowdAppSpec{
				Deployments: []crd.Deployment{{
					Name: "service",
					WebServices: crd.WebServices{
						Public: crd.PublicWebService{
							Enabled: true,
						},
					},
				}},
			},
		}},
	}

	// ClowdAppRef with 'inventory' in serves
	appRefs := &crd.ClowdAppRefList{
		Items: []crd.ClowdAppRef{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rbac",
				Namespace: "default",
			},
			Spec: crd.ClowdAppRefSpec{
				Serves: []string{"inventory"},
				Deployments: []crd.ClowdAppRefDeployment{{
					Name:     "service",
					Hostname: "rbac.remote.example.com",
					WebServices: crd.WebServices{
						Public: crd.PublicWebService{
							Enabled: true,
						},
					},
				}},
			},
		}},
	}

	deps := []config.DependencyEndpoint{}
	privDeps := []config.PrivateDependencyEndpoint{}

	missing := makeDepConfig(&deps, &privDeps, makeWebCfg(), &app, &apps, appRefs)

	if len(missing) > 0 {
		t.Errorf("Expected no missing deps, got %v", missing)
	}

	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}

	// Should use ClowdAppRef because inventory is in serves
	expectedHostname := "rbac.remote.example.com"
	if deps[0].Hostname != expectedHostname {
		t.Errorf("Expected hostname %s (ClowdAppRef), got %s", expectedHostname, deps[0].Hostname)
	}
}

// TestServesField_MultipleDeployments_AllFromSameSource tests that when both ClowdApp and ClowdAppRef
// have multiple deployments, all come from the same source
func TestServesField_MultipleDeployments_AllFromSameSource(t *testing.T) {
	app := crd.ClowdApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "inventory",
			Namespace: "default",
		},
		Spec: crd.ClowdAppSpec{
			Dependencies: []string{"rbac"},
			Deployments: []crd.Deployment{{
				Name: "api",
			}},
		},
	}

	// Local ClowdApp 'rbac' with 2 deployments
	apps := crd.ClowdAppList{
		Items: []crd.ClowdApp{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rbac",
				Namespace: "rbac-ns",
			},
			Spec: crd.ClowdAppSpec{
				Deployments: []crd.Deployment{
					{
						Name: "service-1",
						WebServices: crd.WebServices{
							Public: crd.PublicWebService{
								Enabled: true,
							},
						},
					},
					{
						Name: "service-2",
						WebServices: crd.WebServices{
							Public: crd.PublicWebService{
								Enabled: true,
							},
						},
					},
				},
			},
		}},
	}

	// ClowdAppRef 'rbac' with 2 deployments and 'inventory' in serves
	appRefs := &crd.ClowdAppRefList{
		Items: []crd.ClowdAppRef{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rbac",
				Namespace: "default",
			},
			Spec: crd.ClowdAppRefSpec{
				Serves: []string{"inventory"},
				Deployments: []crd.ClowdAppRefDeployment{
					{
						Name:     "service-1",
						Hostname: "rbac-1.remote.example.com",
						WebServices: crd.WebServices{
							Public: crd.PublicWebService{
								Enabled: true,
							},
						},
					},
					{
						Name:     "service-2",
						Hostname: "rbac-2.remote.example.com",
						WebServices: crd.WebServices{
							Public: crd.PublicWebService{
								Enabled: true,
							},
						},
					},
				},
			},
		}},
	}

	deps := []config.DependencyEndpoint{}
	privDeps := []config.PrivateDependencyEndpoint{}

	missing := makeDepConfig(&deps, &privDeps, makeWebCfg(), &app, &apps, appRefs)

	if len(missing) > 0 {
		t.Errorf("Expected no missing deps, got %v", missing)
	}

	if len(deps) != 2 {
		t.Fatalf("Expected 2 dependencies (both from ClowdAppRef), got %d", len(deps))
	}

	// All should be from ClowdAppRef
	for i, dep := range deps {
		if dep.Hostname != "rbac-1.remote.example.com" && dep.Hostname != "rbac-2.remote.example.com" {
			t.Errorf("Deployment %d has unexpected hostname %s (should be from ClowdAppRef)", i, dep.Hostname)
		}
	}

	// Verify no local ClowdApp hostnames are present
	for _, dep := range deps {
		if dep.Hostname == "rbac-service-1.rbac-ns.svc" || dep.Hostname == "rbac-service-2.rbac-ns.svc" {
			t.Errorf("Found local ClowdApp hostname %s - should only use ClowdAppRef", dep.Hostname)
		}
	}
}
