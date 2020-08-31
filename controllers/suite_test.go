/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"path/filepath"
	"testing"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	cloudredhatcomv1alpha1 "cloud.redhat.com/whippoorwill/v2/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestApi(t *testing.T) {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("bootstrapping test environment")

	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	cfg, err := testEnv.Start()

	if err != nil {
		t.Error(err)
		return
	}

	if cfg == nil {
		t.Error("env config was returned nil")
		return
	}

	err = cloudredhatcomv1alpha1.AddToScheme(scheme.Scheme)

	if err != nil {
		t.Error(err)
		return
	}

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})

	if err != nil {
		t.Error(err)
		return
	}

	if k8sClient == nil {
		t.Error("k8sClient was returned nil")
		return
	}

	err = testEnv.Stop()

	if err != nil {
		t.Error(err)
		return
	}
}
