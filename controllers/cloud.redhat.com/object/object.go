package object

import (
	"context"
	"fmt"
	"sync"

	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClowdObject defines functions shared by ClowdEnv, ClowdApp, and ClowdJobInvocation
type ClowdObject interface {
	MakeOwnerReference() metav1.OwnerReference
	GetLabels() map[string]string
	GetClowdNamespace() string
	GetClowdName() string
	GetUID() types.UID
	GetClowdSAName() string
	GetPrimaryLabel() string
	GroupVersionKind() schema.GroupVersionKind
	GetNamespacesInEnv(context.Context, client.Client) ([]string, error)
	client.Object
}

// LabeledClowdObject is used to be able to treat ClowdEnv and ClowdApp as the same type
type LabeledClowdObject interface {
	MakeOwnerReference() metav1.OwnerReference
	GetLabels() map[string]string
}

type IPCCache struct {
	configs map[string]*configCache
}

type configCache struct {
	newConfig *config.AppConfig
	mutex     sync.RWMutex
}

func NewIPCCache() *IPCCache {
	return &IPCCache{
		configs: make(map[string]*configCache),
	}
}

func (ipccache *IPCCache) GetWriteableConfig(key string) *config.AppConfig {
	var ok bool
	if _, ok = ipccache.configs[key]; !ok {
		ipccache.configs[key] = &configCache{}
		ipccache.configs[key].newConfig = &config.AppConfig{Internal: &config.Internal{}}
		ipccache.configs[key].mutex = sync.RWMutex{}
	}
	return ipccache.configs[key].newConfig
}

func (ipccache *IPCCache) GetReadableConfig(key string) (*config.AppConfig, error) {
	var ok bool
	if _, ok = ipccache.configs[key]; !ok {
		return nil, fmt.Errorf("cache does not hold env [%s]", key)
	}
	return ipccache.configs[key].newConfig, nil
}

func (ipccache *IPCCache) UnlockConfig(key string) {
	if _, ok := ipccache.configs[key]; ok {
		ipccache.configs[key].mutex.Unlock()
	}
}

func (ipccache *IPCCache) LockConfig(key string) {
	if _, ok := ipccache.configs[key]; ok {
		ipccache.configs[key].mutex.Lock()
	}
}

func (ipccache *IPCCache) PersistConfig(key string) {
	// ipccache.configs[key].newConfig = ipccache.configs[key].newConfig
}
