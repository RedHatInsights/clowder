package providers

import (
	"context"
	"encoding/json"
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/clowder_config"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/object"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	"github.com/RedHatInsights/go-difflib/difflib"
	"github.com/go-logr/logr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	core "k8s.io/api/core/v1"

	cyndi "cloud.redhat.com/clowder/v2/apis/cyndi-operator/v1alpha1"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ResourceIdent interface {
	GetProvider() string
	GetPurpose() string
	GetType() runtime.Object
}

// ResourceIdent is a simple struct declaring a providers identifier and the type of resource to be
// put into the cache. It functions as an identifier allowing multiple objects to be returned if
// they all come from the same provider and have the same purpose. Think a list of Jobs created by
// a Job creator.
type ResourceIdentSingle struct {
	Provider string
	Purpose  string
	Type     runtime.Object
}

func (r ResourceIdentSingle) GetProvider() string {
	return r.Provider
}

func (r ResourceIdentSingle) GetPurpose() string {
	return r.Purpose
}

func (r ResourceIdentSingle) GetType() runtime.Object {
	return r.Type
}

// ResourceIdent is a simple struct declaring a providers identifier and the type of resource to be
// put into the cache. It functions as an identifier allowing multiple objects to be returned if
// they all come from the same provider and have the same purpose. Think a list of Jobs created by
// a Job creator.
type ResourceIdentMulti struct {
	Provider string
	Purpose  string
	Type     runtime.Object
}

func (r ResourceIdentMulti) GetProvider() string {
	return r.Provider
}

func (r ResourceIdentMulti) GetPurpose() string {
	return r.Purpose
}

func (r ResourceIdentMulti) GetType() runtime.Object {
	return r.Type
}

var possibleGVKs = make(map[schema.GroupVersionKind]bool)

var scheme = runtime.NewScheme()

var protectedGVKs = make(map[schema.GroupVersionKind]bool)

var secretCompare schema.GroupVersionKind

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(crd.AddToScheme(scheme))
	utilruntime.Must(strimzi.AddToScheme(scheme))
	utilruntime.Must(cyndi.AddToScheme(scheme))

	gvk, _ := getKindFromObj(scheme, &strimzi.KafkaTopic{})
	protectedGVKs[gvk] = true

	secretCompare, _ = getKindFromObj(scheme, &core.Secret{})
}

func registerGVK(obj runtime.Object) {
	gvk, _ := getKindFromObj(scheme, obj)
	if _, ok := protectedGVKs[gvk]; !ok {
		if _, ok := possibleGVKs[gvk]; !ok {
			possibleGVKs[gvk] = true
			fmt.Println("Registered type ", gvk.Group, gvk.Kind, gvk.Version)
		}
	}
}

// NewResourceIdent is a helper function that returns a ResourceIdent object.
func NewSingleResourceIdent(provider string, purpose string, object runtime.Object) ResourceIdentSingle {
	registerGVK(object)
	return ResourceIdentSingle{
		Provider: provider,
		Purpose:  purpose,
		Type:     object,
	}
}

// NewResourceIdent is a helper function that returns a ResourceIdent object.
func NewMultiResourceIdent(provider string, purpose string, object runtime.Object) ResourceIdentMulti {
	registerGVK(object)
	return ResourceIdentMulti{
		Provider: provider,
		Purpose:  purpose,
		Type:     object,
	}
}

// ObjectCache is the main caching provider object. It holds references to some anciliary objects
// as well as a Data structure that is used to hold the K8sResources.
type ObjectCache struct {
	data            map[ResourceIdent]map[types.NamespacedName]*k8sResource
	resourceTracker map[schema.GroupVersionKind]map[types.NamespacedName]bool
	scheme          *runtime.Scheme
	client          client.Client
	ctx             context.Context
	log             logr.Logger
}

type k8sResource struct {
	Object   runtime.Object
	Update   utils.Updater
	Status   bool
	jsonData string
}

// NewObjectCache returns an instance of the ObjectCache which defers all applys until the end of
// the reconciliation process, and allows providers to pull objects out of the cache for
// modification.
func NewObjectCache(ctx context.Context, kclient client.Client, scheme *runtime.Scheme) ObjectCache {
	logCheck := ctx.Value(errors.ClowdKey("log"))
	var log logr.Logger

	if logCheck == nil {
		log = ctrllog.NullLogger{}
	} else {
		log = (*ctx.Value(errors.ClowdKey("log")).(*logr.Logger)).WithName("resource-cache-client")
	}

	return ObjectCache{
		scheme:          scheme,
		client:          kclient,
		ctx:             ctx,
		data:            make(map[ResourceIdent]map[types.NamespacedName]*k8sResource),
		resourceTracker: make(map[schema.GroupVersionKind]map[types.NamespacedName]bool),
		log:             log,
	}
}

// Create first attempts to fetch the object from k8s for initial population. If this fails, the
// blank object is stored in the cache it is imperitive that the user of this function call Create
// before modifying the obejct they wish to be placed in the cache.
func (o *ObjectCache) Create(resourceIdent ResourceIdent, nn types.NamespacedName, object runtime.Object) error {

	update, err := utils.UpdateOrErr(o.client.Get(o.ctx, nn, object))

	if err != nil {
		return err
	}

	if _, ok := o.data[resourceIdent][nn]; ok {
		return fmt.Errorf("cannot create: ident store [%s] already has item named [%s]", resourceIdent, nn)
	}

	gvk, err := getKindFromObj(o.scheme, resourceIdent.GetType())

	if err != nil {
		return err
	}

	if _, ok := o.resourceTracker[gvk]; !ok {
		o.resourceTracker[gvk] = map[types.NamespacedName]bool{nn: true}
	}

	o.resourceTracker[gvk][nn] = true

	if _, ok := o.data[resourceIdent]; !ok {
		o.data[resourceIdent] = make(map[types.NamespacedName]*k8sResource)
	}

	var jsonData []byte
	if clowder_config.LoadedConfig.DebugOptions.Cache.Create || clowder_config.LoadedConfig.DebugOptions.Cache.Apply {
		jsonData, _ = json.MarshalIndent(object, "", "  ")
	}

	o.data[resourceIdent][nn] = &k8sResource{
		Object:   object.DeepCopyObject(),
		Update:   update,
		Status:   false,
		jsonData: string(jsonData),
	}

	if clowder_config.LoadedConfig.DebugOptions.Cache.Create {
		diffVal := "hidden"

		if object.GetObjectKind().GroupVersionKind() != secretCompare {
			diffVal = string(jsonData)
		}

		o.log.Info("CREATE resource ",
			"namespace", nn.Namespace,
			"name", nn.Name,
			"provider", resourceIdent.GetProvider(),
			"purpose", resourceIdent.GetPurpose(),
			"kind", object.GetObjectKind().GroupVersionKind().Kind,
			"diff", diffVal,
		)
	}

	return nil
}

// Update takes the item and tries to update the version in the cache. This will fail if the item is
// not in the cache. A previous provider should have "created" the item before it can be updated.
func (o *ObjectCache) Update(resourceIdent ResourceIdent, object runtime.Object) error {
	if _, ok := o.data[resourceIdent]; !ok {
		return fmt.Errorf("object cache not found, cannot update")
	}

	nn, err := getNamespacedNameFromRuntime(object)

	if err != nil {
		return err
	}

	if _, ok := o.data[resourceIdent][nn]; !ok {
		return fmt.Errorf("object not found in cache, cannot update")
	}

	o.data[resourceIdent][nn].Object = object.DeepCopyObject()

	if clowder_config.LoadedConfig.DebugOptions.Cache.Update {
		var jsonData []byte
		jsonData, _ = json.MarshalIndent(o.data[resourceIdent][nn].Object, "", "  ")
		if object.GetObjectKind().GroupVersionKind() == secretCompare {
			o.log.Info("UPDATE resource ", "namespace", nn.Namespace, "name", nn.Name, "provider", resourceIdent.GetProvider(), "purpose", resourceIdent.GetPurpose(), "kind", object.GetObjectKind().GroupVersionKind().Kind, "diff", "hidden")
		} else {
			o.log.Info("UPDATE resource ", "namespace", nn.Namespace, "name", nn.Name, "provider", resourceIdent.GetProvider(), "purpose", resourceIdent.GetPurpose(), "kind", object.GetObjectKind().GroupVersionKind().Kind, "diff", string(jsonData))
		}
	}

	return nil
}

// Get pulls the item from the cache and populates the given empty object. An error is returned if
// the items are of different types and also if the item is not in the cache. A get should be used
// by a downstream provider. If modifications are made to the object, it should be updated using the
// Update call.
func (o *ObjectCache) Get(resourceIdent ResourceIdent, object runtime.Object, nn ...types.NamespacedName) error {
	if _, ok := o.data[resourceIdent]; !ok {
		return fmt.Errorf("object cache not found, cannot get")
	}

	if len(nn) > 1 {
		return fmt.Errorf("cannot request more than one named item with get, use list")
	}

	if _, ok := resourceIdent.(ResourceIdentSingle); ok {
		oMap := o.data[resourceIdent]
		for _, v := range oMap {
			if err := o.scheme.Convert(v.Object, object, o.ctx); err != nil {
				return err
			}
			object.GetObjectKind().SetGroupVersionKind(v.Object.GetObjectKind().GroupVersionKind())
		}
	} else {
		v, ok := o.data[resourceIdent][nn[0]]
		if !ok {
			return fmt.Errorf("object not found")
		}
		if err := o.scheme.Convert(v.Object, object, o.ctx); err != nil {
			return err
		}
		object.GetObjectKind().SetGroupVersionKind(v.Object.GetObjectKind().GroupVersionKind())
	}
	return nil
}

// List returns a list of objects stored in the cache for the given ResourceIdent. This list
// behanves like a standard k8s List object although the revision cannot be relied upon. It is
// simply to return something that is familiar to users of k8s client-go. It internally converts the
// objects in the list to JSON and adds them to a JSON string, before converting the entire object
// into a List object of the type passed in by the user.
func (o *ObjectCache) List(resourceIdent ResourceIdentMulti, object runtime.Object) error {
	oMap := o.data[resourceIdent]

	gvCopy := core.SchemeGroupVersion
	negotiator := runtime.NewClientNegotiator(clientgoscheme.Codecs.WithoutConversion(), gvCopy)
	encoder, err := negotiator.Encoder("application/json", nil)

	if err != nil {
		return err
	}

	decoder, err := negotiator.Decoder("application/json", nil)

	if err != nil {
		return err
	}

	data := []byte("{\"metadata\":{\"resourceVersion\":\"1\"},\"items\":[")

	for _, v := range oMap {

		bt, err := runtime.Encode(encoder, v.Object)

		if err != nil {
			return err
		}

		data = append(data, bt...)
		data = append(data, []byte(",")...)

	}

	endstring := []byte("]}")

	if len(oMap) > 0 {
		data = data[0 : len(data)-1]
	}

	data = append(data, endstring...)

	err = runtime.DecodeInto(decoder, data, object)

	if err != nil {
		return err
	}
	return nil
}

// Status marks the object for having a status update
func (o *ObjectCache) Status(resourceIdent ResourceIdent, object runtime.Object) error {
	if _, ok := o.data[resourceIdent]; !ok {
		return fmt.Errorf("object cache not found, cannot update")
	}

	nn, err := getNamespacedNameFromRuntime(object)

	if err != nil {
		return err
	}

	o.data[resourceIdent][nn].Status = true

	return nil
}

// ApplyAll takes all the items in the cache and tries to apply them, given the boolean by the
// update field on the internal resource. If the update is true, then the object will by applied, if
// it is false, then the object will be created.
func (o *ObjectCache) ApplyAll() error {
	for k, v := range o.data {
		for n, i := range v {
			o.log.Info("APPLY resource ", "namespace", n.Namespace, "name", n.Name, "provider", k.GetProvider(), "purpose", k.GetPurpose(), "kind", i.Object.GetObjectKind().GroupVersionKind().Kind, "update", i.Update)
			if clowder_config.LoadedConfig.DebugOptions.Cache.Apply {
				jsonData, _ := json.MarshalIndent(i.Object, "", "  ")
				diff := difflib.UnifiedDiff{
					A:        difflib.SplitLines(string(jsonData)),
					B:        difflib.SplitLines(i.jsonData),
					FromFile: "old",
					ToFile:   "new",
					Context:  3,
				}
				text, _ := difflib.GetUnifiedDiffString(diff)
				if i.Object.GetObjectKind().GroupVersionKind() == secretCompare {
					o.log.Info("Update diff", "diff", "hidden", "type", "update", "resType", i.Object.GetObjectKind().GroupVersionKind().Kind, "name", n.Name, "namespace", n.Namespace)
				} else {
					o.log.Info("Update diff", "diff", text, "type", "update", "resType", i.Object.GetObjectKind().GroupVersionKind().Kind, "name", n.Name, "namespace", n.Namespace)
				}
			}
			if err := i.Update.Apply(o.ctx, o.client, i.Object); err != nil {
				return err
			}
			if i.Status {
				if err := o.client.Status().Update(o.ctx, i.Object); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// Debug prints out the contents of the cache.
func (o *ObjectCache) Debug() {
	for iden, v := range o.data {
		fmt.Printf("\n%v-%v", iden.GetProvider(), iden.GetPurpose())
		for pi, i := range v {
			nn, err := getNamespacedNameFromRuntime(i.Object)
			if err != nil {
				fmt.Print(err.Error())
			}
			gvks, _, _ := o.scheme.ObjectKinds(i.Object)
			gvk := gvks[0]
			fmt.Printf("\nObject %v - %v - %v - %v\n", nn, i.Update, gvk, pi)
		}
	}
}

// Reconcile performs the delete on objects that are no longer required
func (o *ObjectCache) Reconcile(clowdObj object.ClowdObject) error {
	clowdApp, isClowdApp := clowdObj.(*crd.ClowdApp)

	//fmt.Print("-----------------" + clowdObj.GetPrimaryLabel())
	for gvk := range possibleGVKs {
		v, ok := o.resourceTracker[gvk]

		if !ok {
			v = make(map[types.NamespacedName]bool)
		}

		nobjList := unstructured.UnstructuredList{}
		nobjList.SetGroupVersionKind(gvk)

		var opts []client.ListOption

		opts = []client.ListOption{
			client.MatchingLabels{clowdObj.GetPrimaryLabel(): clowdObj.GetClowdName()},
		}

		if isClowdApp {
			opts = append(opts, client.InNamespace(clowdApp.Namespace))
		}

		err := o.client.List(o.ctx, &nobjList, opts...)
		if err != nil {
			return err
		}

		//fmt.Printf("\n%v %v", gvk, len(nobjList.Items))

		for _, obj := range nobjList.Items {
			for _, ownerRef := range obj.GetOwnerReferences() {
				if ownerRef.UID == clowdObj.GetUID() {
					nn := types.NamespacedName{
						Name:      obj.GetName(),
						Namespace: obj.GetNamespace(),
					}
					if err != nil {
						return err
					}
					//fmt.Printf("\n%v\n", v)
					if _, ok := v[nn]; !ok {
						o.log.Info("DELETE resource ", "namespace", obj.GetNamespace(), "name", obj.GetName(), "kind", obj.GetObjectKind().GroupVersionKind().Kind)
						err := o.client.Delete(o.ctx, &obj)
						if err != nil {
							return err
						}
					}
				}
			}
		}
	}
	//fmt.Println("\n-----------------")
	return nil
}

func getNamespacedNameFromRuntime(object runtime.Object) (types.NamespacedName, error) {
	om, err := meta.Accessor(object)

	if err != nil {
		return types.NamespacedName{}, err
	}

	nn := types.NamespacedName{
		Namespace: om.GetNamespace(),
		Name:      om.GetName(),
	}

	return nn, nil
}

func getKindFromObj(scheme *runtime.Scheme, object runtime.Object) (schema.GroupVersionKind, error) {
	gvks, nok, err := scheme.ObjectKinds(object)

	if err != nil {
		return schema.EmptyObjectKind.GroupVersionKind(), err
	}

	if nok {
		return schema.EmptyObjectKind.GroupVersionKind(), fmt.Errorf("object type is unknown")
	}

	return gvks[0], nil
}
