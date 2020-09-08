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

	"fmt"

	"cloud.redhat.com/whippoorwill/v2/controllers/cloud.redhat.com/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

//ObjectStoreMaker makes the StorageConfig object
type ObjectStoreMaker struct {
	*Maker
	config config.ObjectStoreConfig
}

//Make function for the StorageMaker
func (obs *ObjectStoreMaker) Make() error {
	obs.config = config.ObjectStoreConfig{}

	var f func() error

	switch obs.Base.Spec.ObjectStore.Provider {
	case "minio":
		f = obs.minio
	case "app-interface":
		f = obs.appInterface
	}

	if f != nil {
		return f()
	}

	return nil
}

//ApplyConfig for the StorageMaker
func (obs *ObjectStoreMaker) ApplyConfig(c *config.AppConfig) {
	c.ObjectStore = obs.config
}

func (obs *ObjectStoreMaker) appInterface() error {
	return nil
}

func (obs *ObjectStoreMaker) minio() error {
	if obs.App.Spec.ObjectStore != nil {
		endpoint := obs.Base.GetAnnotations()["endpoint"]
		accessKeyID := obs.Base.GetAnnotations()["accessKey"]
		secretAccessKey := obs.Base.GetAnnotations()["secretKey"]
		obs.Log.Info(endpoint)
		obs.Log.Info(accessKeyID)
		obs.Log.Info(secretAccessKey)
		// Initialize minio client object.
		minioClient, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Secure: false,
		})
		if err != nil {
			return err
		}

		obs.Log.Info(fmt.Sprintf("%v", minioClient)) // minioClient is now setup

		for _, bucket := range obs.App.Spec.ObjectStore {
			found, err := minioClient.BucketExists(obs.Ctx, bucket)
			if err != nil {
				return err
			}
			if !found {
				err := minioClient.MakeBucket(obs.Ctx, bucket, minio.MakeBucketOptions{})
				if err != nil {
					return err
				}
			}
		}

		obs.config.AccessKey = accessKeyID
		obs.config.SecretKey = secretAccessKey
		obs.config.Endpoint = endpoint
	}

	return nil
}
