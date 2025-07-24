package hashcache

import (
	"testing"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/stretchr/testify/assert"
	core "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestHashCacheAddItemAndRetrieve(t *testing.T) {
	sec := &core.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:        "test",
			Namespace:   "def",
			Annotations: map[string]string{clowderconfig.LoadedConfig.Settings.RestarterAnnotationName: "true"},
		},
	}

	hc := NewHashCache()
	update, err := hc.CreateOrUpdateObject(sec, false)
	assert.NoError(t, err)
	assert.True(t, update)
	obj, err := hc.Read(sec)
	assert.Equal(t, "74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b", obj.Hash)
	assert.NoError(t, err)
}

func TestHashCacheDeleteItem(t *testing.T) {
	sec := &core.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:        "test",
			Namespace:   "def",
			Annotations: map[string]string{clowderconfig.LoadedConfig.Settings.RestarterAnnotationName: "true"},
		},
	}

	hc := NewHashCache()
	shouldUpdate, err := hc.CreateOrUpdateObject(sec, false)
	assert.True(t, shouldUpdate)
	assert.NoError(t, err)
	obj, err := hc.Read(sec)
	assert.Equal(t, "74234e98afe7498fb5daf1f36ac2d78acc339464f950703b8c019892f982b90b", obj.Hash)
	assert.NoError(t, err)
	hc.Delete(sec)
	_, err = hc.Read(sec)
	assert.ErrorIs(t, err, ItemNotFoundError{item: "test/def"})
}

func TestHashCacheUpdateItem(t *testing.T) {
	sec := &core.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:        "test",
			Namespace:   "def",
			Annotations: map[string]string{clowderconfig.LoadedConfig.Settings.RestarterAnnotationName: "true"},
		},
		Data: map[string][]byte{
			"test": []byte("test"),
		},
	}

	hc := NewHashCache()
	_, err := hc.CreateOrUpdateObject(sec, false)
	assert.NoError(t, err)

	obj, err := hc.Read(sec)
	assert.Equal(t, "63e7360f7b4cc56da3192298bbcfeb9d85fffdd68d41d6d2723787cbf1344954", obj.Hash)
	assert.NoError(t, err)

	sec.Data = map[string][]byte{
		"test":  []byte("test"),
		"test2": []byte("test2"),
	}

	update, err := hc.CreateOrUpdateObject(sec, false)
	assert.NoError(t, err)
	assert.True(t, update)
	obj, err = hc.Read(sec)
	assert.Equal(t, "1314de0f8fa7c92419ff59ad9ca6b9c921142a494a8896bde477caefdbe92fc2", obj.Hash)
	assert.NoError(t, err)

}

func TestHashCacheRetrieveUnknownItem(t *testing.T) {
	sec := &core.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:        "test",
			Namespace:   "def",
			Annotations: map[string]string{clowderconfig.LoadedConfig.Settings.RestarterAnnotationName: "true"},
		},
	}

	hc := NewHashCache()
	_, err := hc.Read(sec)
	assert.ErrorIs(t, err, ItemNotFoundError{item: "test/def"})
}

func TestHashCacheAddClowdObj(t *testing.T) {
	sec := &core.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:        "test",
			Namespace:   "def",
			Annotations: map[string]string{clowderconfig.LoadedConfig.Settings.RestarterAnnotationName: "true"},
		},
	}

	capp := &crd.ClowdApp{
		ObjectMeta: v1.ObjectMeta{
			Name:      "testapp",
			Namespace: "def",
		},
	}

	clowdObjNamespaceName := types.NamespacedName{
		Name:      capp.GetName(),
		Namespace: capp.GetNamespace(),
	}

	hc := NewHashCache()
	_, err := hc.CreateOrUpdateObject(sec, false)
	assert.NoError(t, err)

	err = hc.AddClowdObjectToObject(capp, sec)
	assert.NoError(t, err)
	obj, err := hc.Read(sec)
	assert.NoError(t, err)
	assert.Contains(t, obj.ClowdApps, clowdObjNamespaceName)
}

func TestHashCacheDeleteClowdObj(t *testing.T) {
	sec := &core.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:        "test",
			Namespace:   "def",
			Annotations: map[string]string{clowderconfig.LoadedConfig.Settings.RestarterAnnotationName: "true"},
		},
	}

	capp := &crd.ClowdApp{
		ObjectMeta: v1.ObjectMeta{
			Name:      "testapp",
			Namespace: "def",
		},
	}

	clowdObjNamespaceName := types.NamespacedName{
		Name:      capp.GetName(),
		Namespace: capp.GetNamespace(),
	}

	hc := NewHashCache()
	_, err := hc.CreateOrUpdateObject(sec, false)
	assert.NoError(t, err)

	err = hc.AddClowdObjectToObject(capp, sec)
	assert.NoError(t, err)
	obj, err := hc.Read(sec)
	assert.NoError(t, err)
	assert.Contains(t, obj.ClowdApps, clowdObjNamespaceName)
	hc.RemoveClowdObjectFromObjects(capp)
	obj, err = hc.Read(sec)
	assert.NoError(t, err)
	assert.NotContains(t, obj.ClowdApps, clowdObjNamespaceName)
}

func TestHashCacheSuperCache(t *testing.T) {
	sec := &core.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:        "test",
			Namespace:   "def",
			Annotations: map[string]string{clowderconfig.LoadedConfig.Settings.RestarterAnnotationName: "true"},
		},
	}

	sec2 := &core.Secret{
		ObjectMeta: v1.ObjectMeta{
			Name:        "test2",
			Namespace:   "def",
			Annotations: map[string]string{clowderconfig.LoadedConfig.Settings.RestarterAnnotationName: "true"},
		},
	}

	capp := &crd.ClowdApp{
		ObjectMeta: v1.ObjectMeta{
			Name:      "testapp",
			Namespace: "def",
		},
	}

	clowdObjNamespaceName := types.NamespacedName{
		Name:      capp.GetName(),
		Namespace: capp.GetNamespace(),
	}

	hc := NewHashCache()
	_, err := hc.CreateOrUpdateObject(sec, false)
	assert.NoError(t, err)
	err = hc.AddClowdObjectToObject(capp, sec)
	assert.NoError(t, err)
	obj, err := hc.Read(sec)
	assert.NoError(t, err)
	assert.Contains(t, obj.ClowdApps, clowdObjNamespaceName)

	_, err = hc.CreateOrUpdateObject(sec2, false)
	assert.NoError(t, err)
	err = hc.AddClowdObjectToObject(capp, sec2)
	assert.NoError(t, err)
	obj, err = hc.Read(sec2)
	assert.NoError(t, err)
	assert.Contains(t, obj.ClowdApps, clowdObjNamespaceName)

	superHash := hc.GetSuperHashForClowdObject(capp)
	superHash2 := hc.GetSuperHashForClowdObject(capp)
	assert.Equal(t, superHash, superHash2)
}
