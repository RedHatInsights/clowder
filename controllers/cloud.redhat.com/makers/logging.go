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

package makers

import (
	//config "github.com/redhatinsights/app-common-go/pkg/api/v1" - to replace the import below at a future date
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

//LoggingMaker makes the LoggingConfig obejct
type LoggingMaker struct {
	*Maker
	config config.LoggingConfig
}

//Make function for the Logging Maker
func (l *LoggingMaker) Make() (ctrl.Result, error) {
	l.config = config.LoggingConfig{}

	providerFns := []func() error{}

	for _, provider := range l.Base.Spec.Logging.Providers {
		if provider == "cloudwatch" {
			providerFns = append(providerFns, l.cloudwatch)
		} else if provider == "local" {
			providerFns = append(providerFns, l.local)
		}
	}

	for _, fn := range providerFns {
		err := fn()

		if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

//ApplyConfig for the LoggingMaker
func (l *LoggingMaker) ApplyConfig(c *config.AppConfig) {
	c.Logging = l.config
}

func (l *LoggingMaker) local() error {
	return nil
}

func (l *LoggingMaker) cloudwatch() error {

	name := types.NamespacedName{
		Name:      "cloudwatch",
		Namespace: l.App.Namespace,
	}

	secret := core.Secret{}
	err := l.Client.Get(l.Ctx, name, &secret)

	if err != nil {
		return err
	}

	cwKeys := []string{
		"aws_access_key_id",
		"aws_secret_access_key",
		"aws_region",
		"log_group_name",
	}

	decoded := make([]string, 4)

	for i := 0; i < 4; i++ {
		decoded[i], err = utils.B64Decode(&secret, cwKeys[i])

		if err != nil {
			return err
		}
	}

	l.config.Cloudwatch = &config.CloudWatchConfig{
		AccessKeyId:     decoded[0],
		SecretAccessKey: decoded[1],
		Region:          decoded[2],
		LogGroup:        decoded[3],
	}

	return nil
}
