package config

import (
	"strconv"
)

// Populate sets the database configuration on the object from the passed in map.
func (dbc *DatabaseConfig) Populate(data *map[string]string) {
	port, _ := strconv.Atoi((*data)["port"])
	dbc.Hostname = (*data)["hostname"]
	dbc.Name = (*data)["name"]
	dbc.Password = (*data)["password"]
	dbc.AdminPassword = (*data)["pgPass"]
	dbc.Port = port
	dbc.Username = (*data)["username"]
}
