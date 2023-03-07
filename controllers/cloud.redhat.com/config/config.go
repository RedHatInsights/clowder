package config

import (
	"strconv"

	"k8s.io/apimachinery/pkg/types"
)

// Populate sets the database configuration on the object from the passed in map.
func (dbc *DatabaseConfig) Populate(data *map[string]string) error {
	port, err := strconv.ParseUint((*data)["port"], 10, 16)
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

type DatabaseConfigContainer struct {
	Config DatabaseConfig       `json:"config"`
	Ref    types.NamespacedName `json:"ref"`
}
