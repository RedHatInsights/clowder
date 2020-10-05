package config

import (
	"context"
	"strconv"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/types"
)

func (dbc *DatabaseConfig) Populate(data *map[string]string) {
	port, _ := strconv.Atoi((*data)["port"])
	dbc.Hostname = (*data)["hostname"]
	dbc.Name = (*data)["name"]
	dbc.Password = (*data)["password"]
	dbc.PgPass = (*data)["pgPass"]
	dbc.Port = port
	dbc.Username = (*data)["username"]
}

func (ojs *ObjectStoreConfig) Populate(data *map[string]string) {
	port, _ := strconv.Atoi((*data)["port"])
	ojs.Hostname = (*data)["hostname"]
	ojs.AccessKey = (*data)["accessKey"]
	ojs.SecretKey = (*data)["secretKey"]
	ojs.Port = port
}

func MakeOrGetSecret(ctx context.Context, env *crd.ClowdEnvironment, client client.Client, nn types.NamespacedName, dataInit func() map[string]string) (*map[string]string, error) {
	secret := &core.Secret{}
	secretUpdate, err := utils.UpdateOrErr(client.Get(ctx, nn, secret))

	if err != nil {
		return nil, err
	}

	data := make(map[string]string)

	if len(secret.Data) == 0 {
		data = dataInit()
		secret.StringData = data

		secret.Name = nn.Name
		secret.Namespace = nn.Namespace
		secret.ObjectMeta.OwnerReferences = []metav1.OwnerReference{env.MakeOwnerReference()}
		secret.Type = core.SecretTypeOpaque

		if err = secretUpdate.Apply(ctx, client, secret); err != nil {
			return nil, err
		}
	} else {
		err = client.Get(ctx, nn, secret)
		if err != nil {
			return nil, err
		}
		for k, v := range secret.Data {
			(data)[k] = string(v)
		}
	}
	return &data, nil
}
