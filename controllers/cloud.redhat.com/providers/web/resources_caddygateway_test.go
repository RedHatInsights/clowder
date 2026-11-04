package web

import (
	"testing"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildUpstreamAndWhiteLists(t *testing.T) {
	bopHostname := "test-bop-hostname"
	appList := &crd.ClowdAppList{
		Items: []crd.ClowdApp{
			{
				Spec: crd.ClowdAppSpec{
					Deployments: []crd.Deployment{
						{
							Name: "test-deployment-1",
							WebServices: crd.WebServices{
								Public: crd.PublicWebService{
									Enabled:        true,
									APIPath:        "test-api",
									WhitelistPaths: []string{"/whitelist-path-1", "/whitelist-path-2"},
								},
							},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-1",
					Namespace: "test-namespace-1",
				},
			},
			{
				Spec: crd.ClowdAppSpec{
					Deployments: []crd.Deployment{
						{
							Name: "test-deployment-2",
							WebServices: crd.WebServices{
								Public: crd.PublicWebService{
									Enabled: true,
									APIPaths: []crd.APIPath{
										"/api-path-1",
										"/api-path-2",
									},
								},
							},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app-2",
					Namespace: "test-namespace-2",
				},
			},
		},
	}

	// Call the function under test
	upstreamList, whitelistStrings := buildUpstreamAndWhiteLists(bopHostname, appList)

	expectedUpstreamList := []ProxyRoute{
		{Upstream: bopHostname, Path: "/v1/registrations*"},
		{Upstream: bopHostname, Path: "/v1/check_registration*"},
		{Upstream: "test-app-1-test-deployment-1.test-namespace-1.svc:8000", Path: "/api/test-api/*"},
		{Upstream: "test-app-2-test-deployment-2.test-namespace-2.svc:8000", Path: "/api-path-1"},
		{Upstream: "test-app-2-test-deployment-2.test-namespace-2.svc:8000", Path: "/api-path-2"},
	}

	expectedWhitelistStrings := []string{
		"/whitelist-path-1",
		"/whitelist-path-2",
	}

	assert.ElementsMatch(t, expectedUpstreamList, upstreamList, "Upstream list does not match expected")
	assert.ElementsMatch(t, expectedWhitelistStrings, whitelistStrings, "Whitelist strings do not match expected")
}
