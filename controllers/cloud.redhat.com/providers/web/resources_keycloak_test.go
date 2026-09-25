package web

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExternalRealmProvidesGraphQLScopeToCloudServices(t *testing.T) {
	data, err := os.ReadFile("../../../../jsons/redhat-external-realm.json")
	require.NoError(t, err)

	var realm struct {
		Clients []struct {
			ClientID            string   `json:"clientId"`
			DefaultClientScopes []string `json:"defaultClientScopes"`
		} `json:"clients"`
		ClientScopes []struct {
			Name       string            `json:"name"`
			Protocol   string            `json:"protocol"`
			Attributes map[string]string `json:"attributes"`
		} `json:"clientScopes"`
	}
	require.NoError(t, json.Unmarshal(data, &realm))

	var graphqlScopeFound bool
	for _, clientScope := range realm.ClientScopes {
		if clientScope.Name == "api.graphql" {
			graphqlScopeFound = true
			assert.Equal(t, "openid-connect", clientScope.Protocol)
			assert.Equal(t, "true", clientScope.Attributes["include.in.token.scope"])
		}
	}
	require.True(t, graphqlScopeFound, "realm should define api.graphql as a client scope")

	for _, client := range realm.Clients {
		if client.ClientID == "cloud-services" {
			require.Contains(t, client.DefaultClientScopes, "api.graphql")
			return
		}
	}
	t.Fatal("realm should define the cloud-services client")
}
