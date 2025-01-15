package hashcache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func generateHashFromData(data []byte) (hash string) {
	h := sha256.New()
	h.Write([]byte(data))
	hash = fmt.Sprintf("%x", h.Sum(nil))
	return
}

type Ident struct {
	NN   types.NamespacedName
	Type string
}

type HashObject struct {
	Hash      string
	ClowdApps map[types.NamespacedName]bool
	ClowdEnvs map[types.NamespacedName]bool
	Always    bool // Secret/ConfigMap should be always updated
}

type HashCache struct {
	data map[Ident]*HashObject
	lock sync.RWMutex
}

func NewHashCache() HashCache {
	return HashCache{
		data: map[Ident]*HashObject{},
		lock: sync.RWMutex{},
	}
}

func NewHashObject(hash string, always bool) HashObject {
	return HashObject{
		Hash:      hash,
		ClowdApps: map[types.NamespacedName]bool{},
		ClowdEnvs: map[types.NamespacedName]bool{},
		Always:    always,
	}
}

type ItemNotFoundError struct {
	item string
}

func (a ItemNotFoundError) Error() string {
	return fmt.Sprintf("item [%s] not found", a.item)
}

func (hc *HashCache) Read(obj client.Object) (*HashObject, error) {
	var oType string
	switch obj.(type) {
	case *core.ConfigMap:
		oType = "ConfigMap"
	case *core.Secret:
		oType = "Secret"
	}

	id := Ident{NN: types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, Type: oType}
	hc.lock.RLock()
	defer hc.lock.RUnlock()
	v, ok := hc.data[id]
	if !ok {
		return nil, ItemNotFoundError{item: fmt.Sprintf("%s/%s", id.NN.Name, id.NN.Namespace)}
	}
	return v, nil
}

func (hc *HashCache) RemoveClowdObjectFromObjects(obj client.Object) {
	hc.lock.Lock()
	defer hc.lock.Unlock()

	for _, v := range hc.data {
		switch obj.(type) {
		case *crd.ClowdEnvironment:
			delete(v.ClowdEnvs, types.NamespacedName{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			})
		case *crd.ClowdApp:
			delete(v.ClowdApps, types.NamespacedName{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			})
		}
	}
}

// CreatesOrUpdates HashObject and adding attribute alwaysUpdate.
// This function returns a boolean indicating whether the hashCache should be updated.
func (hc *HashCache) CreateOrUpdateObject(obj client.Object, alwaysUpdate bool) (bool, error) {
	hc.lock.Lock()
	defer hc.lock.Unlock()

	var oType string
	var hash string
	switch v := obj.(type) {
	case *core.ConfigMap:
		oType = "ConfigMap"
		jsonData, err := json.Marshal(v.Data)
		if err != nil {
			return false, errors.Wrap("failed to marshal configmap JSON", err)
		}
		hash = generateHashFromData(jsonData)
	case *core.Secret:
		oType = "Secret"
		jsonData, err := json.Marshal(v.Data)
		if err != nil {
			return false, errors.Wrap("failed to marshal configmap JSON", err)
		}
		hash = generateHashFromData(jsonData)
	}

	id := Ident{NN: types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, Type: oType}

	hashObject, ok := hc.data[id]

	if !ok {
		hashObj := NewHashObject(hash, alwaysUpdate)
		hc.data[id] = &hashObj
		return true, nil
	}
	oldHash := hashObject.Hash
	hashObject.Hash = hash
	return oldHash != hash, nil
}

func (hc *HashCache) GetSuperHashForClowdObject(clowdObj object.ClowdObject) string {
	hc.lock.RLock()
	defer hc.lock.RUnlock()

	nn := types.NamespacedName{
		Name:      clowdObj.GetName(),
		Namespace: clowdObj.GetNamespace(),
	}
	keys := []Ident{}
	for k, v := range hc.data {
		switch clowdObj.(type) {
		case *crd.ClowdEnvironment:
			for env := range v.ClowdEnvs {
				if nn == env {
					keys = append(keys, k)
				}
			}
		case *crd.ClowdApp:
			for app := range v.ClowdApps {
				if nn == app {
					keys = append(keys, k)
				}
			}
		}
	}

	sort.Slice(keys, func(i, j int) bool {
		return fmt.Sprintf("%s/%s/%s", keys[i].NN.Name, keys[i].NN.Namespace, keys[i].Type) < fmt.Sprintf("%s/%s/%s", keys[j].NN.Name, keys[j].NN.Namespace, keys[i].Type)
	})

	superstring := ""
	for _, k := range keys {
		superstring += hc.data[k].Hash
	}

	return generateHashFromData([]byte(superstring))
}

func (hc *HashCache) AddClowdObjectToObject(clowdObj object.ClowdObject, obj client.Object) error {
	var oType string

	switch obj.(type) {
	case *core.ConfigMap:
		oType = "ConfigMap"
	case *core.Secret:
		oType = "Secret"
	}

	id := Ident{NN: types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, Type: oType}

	hashObject, ok := hc.data[id]

	if !ok {
		return ItemNotFoundError{item: fmt.Sprintf("%s/%s", id.NN.Name, id.NN.Namespace)}
	}
	if obj.GetAnnotations()[clowderconfig.LoadedConfig.Settings.RestarterAnnotationName] != "true" && !hc.data[id].Always {
		return nil
	}

	hc.lock.Lock()
	defer hc.lock.Unlock()

	clowdObjNamespaceName := types.NamespacedName{
		Name:      clowdObj.GetName(),
		Namespace: clowdObj.GetNamespace(),
	}
	switch clowdObj.(type) {
	case *crd.ClowdApp:
		hashObject.ClowdApps[clowdObjNamespaceName] = true
	case *crd.ClowdEnvironment:
		hashObject.ClowdEnvs[clowdObjNamespaceName] = true
	}
	return nil
}

func (hc *HashCache) Delete(obj client.Object) {
	var oType string

	switch obj.(type) {
	case *core.ConfigMap:
		oType = "ConfigMap"
	case *core.Secret:
		oType = "Secret"
	}

	id := Ident{NN: types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, Type: oType}

	hc.lock.Lock()
	defer hc.lock.Unlock()
	delete(hc.data, id)
}

var DefaultHashCache = NewHashCache()
