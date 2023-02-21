package controllers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
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

type ident struct {
	NN   types.NamespacedName
	Type string
}

type hashObject struct {
	Hash      string
	ClowdApps map[types.NamespacedName]bool
	ClowdEnvs map[types.NamespacedName]bool
}

type hashCache struct {
	data map[ident]*hashObject
	lock sync.RWMutex
}

func NewHashCache() hashCache {
	return hashCache{
		data: map[ident]*hashObject{},
		lock: sync.RWMutex{},
	}
}

func NewHashObject(hash string) hashObject {
	return hashObject{
		Hash:      hash,
		ClowdApps: map[types.NamespacedName]bool{},
		ClowdEnvs: map[types.NamespacedName]bool{},
	}
}

type ItemNotFoundError struct {
	item string
}

func (a ItemNotFoundError) Error() string {
	return fmt.Sprintf("item [%s] not found", a.item)
}

func (hc *hashCache) Read(obj client.Object) (*hashObject, error) {
	var oType string
	switch obj.(type) {
	case *core.ConfigMap:
		oType = "ConfigMap"
	case *core.Secret:
		oType = "Secret"
	}

	id := ident{NN: types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, Type: oType}
	hc.lock.RLock()
	defer hc.lock.RUnlock()
	v, ok := hc.data[id]
	if !ok {
		return nil, ItemNotFoundError{item: fmt.Sprintf("%s/%s", id.NN.Name, id.NN.Namespace)}
	}
	return v, nil
}

func (hc *hashCache) RemoveClowdObjectFromObjects(obj client.Object) {

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

func (hc *hashCache) CreateOrUpdateObject(obj client.Object) (bool, error) {
	// NEED TO PASS IN Q HERE TO GET IT TO ALSO TRIGGER
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

	id := ident{NN: types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, Type: oType}

	hashObject, ok := hc.data[id]

	if !ok {
		hashObj := NewHashObject(hash)
		hc.data[id] = &hashObj
		return true, nil
	} else {
		oldHash := hashObject.Hash
		hashObject.Hash = hash
		return oldHash != hash, nil
	}
}

func (hc *hashCache) AddClowdObjectToObject(clowdObj object.ClowdObject, obj client.Object) error {
	var oType string

	switch obj.(type) {
	case *core.ConfigMap:
		oType = "ConfigMap"
	case *core.Secret:
		oType = "Secret"
	}

	id := ident{NN: types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, Type: oType}

	hashObject, ok := hc.data[id]

	if !ok {
		return ItemNotFoundError{item: fmt.Sprintf("%s/%s", id.NN.Name, id.NN.Namespace)}
	} else {
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
	}
	return nil
}

func (hc *hashCache) Delete(obj client.Object) {
	var oType string

	switch obj.(type) {
	case *core.ConfigMap:
		oType = "ConfigMap"
	case *core.Secret:
		oType = "Secret"
	}

	id := ident{NN: types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, Type: oType}

	hc.lock.Lock()
	defer hc.lock.Unlock()
	delete(hc.data, id)
}

var HashCache = NewHashCache()
