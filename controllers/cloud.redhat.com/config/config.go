package config

import (
	"context"
	"strconv"

	obj "cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
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
	dbc.AdminPassword = (*data)["pgPass"]
	dbc.Port = port
	dbc.Username = (*data)["username"]
}

func MakeOrGetSecret(ctx context.Context, obj obj.ClowdObject, client client.Client, nn types.NamespacedName, dataInit func() map[string]string) (*map[string]string, error) {
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
		secret.ObjectMeta.OwnerReferences = []metav1.OwnerReference{obj.MakeOwnerReference()}
		secret.Type = core.SecretTypeOpaque

		if err = secretUpdate.Apply(ctx, client, secret); err != nil {
			return nil, err
		}
	} else {
		client.Update(ctx, secret)
		for k, v := range secret.Data {
			(data)[k] = string(v)
		}
	}
	return &data, nil
}
