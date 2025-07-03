package config

import (
	"strconv"

	"k8s.io/apimachinery/pkg/types"
)

// Populate sets the database configuration on the object from the passed in map.
func (dbc *DatabaseConfig) Populate(data *map[string]string) error {
	port, err := strconv.Atoi((*data)["port"])
	if err != nil {
		return err
	}
	dbc.Hostname = (*data)["hostname"]
	dbc.Name = (*data)["name"]
	dbc.Password = (*data)["password"]
	dbc.AdminPassword = (*data)["pgPass"]
	dbc.Port = int(port)
	dbc.Username = (*data)["username"]
	return nil
}

func (r *InMemoryDBConfig) Populate(data *map[string]string) error {
	port, err := strconv.Atoi((*data)["port"])
	if err != nil {
		return err
	}
	r.Port = int(port)

	username, exists := (*data)["username"]
	if exists && username != "" {
		r.Username = &username
	}

	password, exists := (*data)["password"]
	if exists && password != "" {
		r.Password = &password
	}

	sslMode, err := strconv.ParseBool((*data)["sslmode"])
	if err != nil {
		return err
	}
	r.SslMode = &sslMode

	r.Hostname = (*data)["hostname"]

	return nil
}

type DatabaseConfigContainer struct {
	Config DatabaseConfig       `json:"config"`
	Ref    types.NamespacedName `json:"ref"`
}
