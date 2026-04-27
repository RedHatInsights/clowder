package iqe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"
)

func configMapKeyRef(name, key string) *core.EnvVarSource {
	return &core.EnvVarSource{
		ConfigMapKeyRef: &core.ConfigMapKeySelector{
			LocalObjectReference: core.LocalObjectReference{Name: name},
			Key:                  key,
		},
	}
}

func secretKeyRef(name, key string) *core.EnvVarSource {
	return &core.EnvVarSource{
		SecretKeyRef: &core.SecretKeySelector{
			LocalObjectReference: core.LocalObjectReference{Name: name},
			Key:                  key,
		},
	}
}

func TestUpdateEnvVars_ValueReplacesValue(t *testing.T) {
	existing := []core.EnvVar{
		{Name: "FOO", Value: "old"},
	}
	updated := updateEnvVars(existing, []core.EnvVar{
		{Name: "FOO", Value: "new"},
	})

	assert.Len(t, updated, 1)
	assert.Equal(t, "new", updated[0].Value)
	assert.Nil(t, updated[0].ValueFrom)
}

func TestUpdateEnvVars_ValueFromReplacesValue(t *testing.T) {
	existing := []core.EnvVar{
		{Name: "FOO", Value: "old"},
	}
	ref := configMapKeyRef("my-config", "MY_KEY")
	updated := updateEnvVars(existing, []core.EnvVar{
		{Name: "FOO", ValueFrom: ref},
	})

	assert.Len(t, updated, 1)
	assert.Equal(t, "", updated[0].Value)
	assert.Equal(t, ref, updated[0].ValueFrom)
}

func TestUpdateEnvVars_ValueReplacesValueFrom(t *testing.T) {
	existing := []core.EnvVar{
		{Name: "FOO", ValueFrom: secretKeyRef("my-secret", "MY_KEY")},
	}
	updated := updateEnvVars(existing, []core.EnvVar{
		{Name: "FOO", Value: "literal"},
	})

	assert.Len(t, updated, 1)
	assert.Equal(t, "literal", updated[0].Value)
	assert.Nil(t, updated[0].ValueFrom)
}

func TestUpdateEnvVars_ValueFromReplacesValueFrom(t *testing.T) {
	existing := []core.EnvVar{
		{Name: "FOO", ValueFrom: secretKeyRef("old-secret", "OLD_KEY")},
	}
	newRef := configMapKeyRef("new-config", "NEW_KEY")
	updated := updateEnvVars(existing, []core.EnvVar{
		{Name: "FOO", ValueFrom: newRef},
	})

	assert.Len(t, updated, 1)
	assert.Equal(t, "", updated[0].Value)
	assert.Equal(t, newRef, updated[0].ValueFrom)
}

func TestUpdateEnvVars_EmptyValueAndNilValueFromSkipped(t *testing.T) {
	existing := []core.EnvVar{
		{Name: "FOO", Value: "keep"},
	}
	updated := updateEnvVars(existing, []core.EnvVar{
		{Name: "FOO", Value: ""},
	})

	assert.Len(t, updated, 1)
	assert.Equal(t, "keep", updated[0].Value)
}

func TestUpdateEnvVars_NewValueAppended(t *testing.T) {
	existing := []core.EnvVar{
		{Name: "FOO", Value: "bar"},
	}
	updated := updateEnvVars(existing, []core.EnvVar{
		{Name: "BAZ", Value: "qux"},
	})

	assert.Len(t, updated, 2)
	assert.Equal(t, "FOO", updated[0].Name)
	assert.Equal(t, "BAZ", updated[1].Name)
	assert.Equal(t, "qux", updated[1].Value)
}

func TestUpdateEnvVars_NewValueFromAppended(t *testing.T) {
	existing := []core.EnvVar{
		{Name: "FOO", Value: "bar"},
	}
	ref := secretKeyRef("ibutsu-token", "IBUTSU_TOKEN")
	updated := updateEnvVars(existing, []core.EnvVar{
		{Name: "IBUTSU_TOKEN", ValueFrom: ref},
	})

	assert.Len(t, updated, 2)
	assert.Equal(t, "FOO", updated[0].Name)
	assert.Equal(t, "IBUTSU_TOKEN", updated[1].Name)
	assert.Equal(t, ref, updated[1].ValueFrom)
	assert.Equal(t, "", updated[1].Value)
}

func TestUpdateEnvVars_MultipleUpdatesAndAppends(t *testing.T) {
	existing := []core.EnvVar{
		{Name: "A", Value: "1"},
		{Name: "B", Value: "2"},
	}
	cmRef := configMapKeyRef("ibutsu-config", "IBUTSU_MODE")
	updated := updateEnvVars(existing, []core.EnvVar{
		{Name: "A", Value: "updated"},
		{Name: "B", ValueFrom: cmRef},
		{Name: "C", Value: "new"},
	})

	assert.Len(t, updated, 3)

	assert.Equal(t, "updated", updated[0].Value)
	assert.Nil(t, updated[0].ValueFrom)

	assert.Equal(t, "", updated[1].Value)
	assert.Equal(t, cmRef, updated[1].ValueFrom)

	assert.Equal(t, "C", updated[2].Name)
	assert.Equal(t, "new", updated[2].Value)
}

func TestUpdateEnvVars_EmptyExisting(t *testing.T) {
	ref := secretKeyRef("my-secret", "KEY")
	updated := updateEnvVars([]core.EnvVar{}, []core.EnvVar{
		{Name: "NEW_VAR", ValueFrom: ref},
	})

	assert.Len(t, updated, 1)
	assert.Equal(t, "NEW_VAR", updated[0].Name)
	assert.Equal(t, ref, updated[0].ValueFrom)
}

func TestUpdateEnvVars_EmptyNew(t *testing.T) {
	existing := []core.EnvVar{
		{Name: "FOO", Value: "bar"},
	}
	updated := updateEnvVars(existing, []core.EnvVar{})

	assert.Len(t, updated, 1)
	assert.Equal(t, "bar", updated[0].Value)
}
