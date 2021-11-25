package providers

import (
	"context"
	"encoding/json"
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/clowderconfig"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/utils"
	"github.com/RedHatInsights/go-difflib/difflib"
	"github.com/go-logr/logr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	core "k8s.io/api/core/v1"

	cyndi "github.com/RedHatInsights/cyndi-operator/api/v1alpha1"
	strimzi "github.com/RedHatInsights/strimzi-client-go/apis/kafka.strimzi.io/v1beta2"
	keda "github.com/kedacore/keda/v2/api/v1alpha1"
	prom "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
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
	GetType() client.Object
	GetWriteNow() bool
}

type ResourceOptions struct {
	WriteNow bool
}

// ResourceIdent is a simple struct declaring a providers identifier and the type of resource to be
// put into the cache. It functions as an identifier allowing multiple objects to be returned if
// they all come from the same provider and have the same purpose. Think a list of Jobs created by
// a Job creator.
type ResourceIdentSingle struct {
	Provider string
	Purpose  string
	Type     client.Object
	WriteNow bool
}

func (r ResourceIdentSingle) GetProvider() string {
	return r.Provider
}

func (r ResourceIdentSingle) GetPurpose() string {
	return r.Purpose
}

func (r ResourceIdentSingle) GetType() client.Object {
	return r.Type
}

func (r ResourceIdentSingle) GetWriteNow() bool {
	return r.WriteNow
}

// ResourceIdent is a simple struct declaring a providers identifier and the type of resource to be
// put into the cache. It functions as an identifier allowing multiple objects to be returned if
// they all come from the same provider and have the same purpose. Think a list of Jobs created by
// a Job creator.
type ResourceIdentMulti struct {
	Provider string
	Purpose  string
	Type     client.Object
	WriteNow bool
}

func (r ResourceIdentMulti) GetProvider() string {
	return r.Provider
}

func (r ResourceIdentMulti) GetPurpose() string {
	return r.Purpose
}

func (r ResourceIdentMulti) GetType() client.Object {
	return r.Type
}

func (r ResourceIdentMulti) GetWriteNow() bool {
	return r.WriteNow
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
	utilruntime.Must(prom.AddToScheme(scheme))
	utilruntime.Must(keda.AddToScheme(scheme))

	gvk, _ := utils.GetKindFromObj(scheme, &strimzi.KafkaTopic{})
	protectedGVKs[gvk] = true

	if !clowderconfig.LoadedConfig.Features.KedaResources {
		gvk, _ := utils.GetKindFromObj(scheme, &keda.ScaledObject{})
		protectedGVKs[gvk] = true
	}

	secretCompare, _ = utils.GetKindFromObj(scheme, &core.Secret{})
}

func registerGVK(obj client.Object) {
	gvk, _ := utils.GetKindFromObj(scheme, obj)
	if _, ok := protectedGVKs[gvk]; !ok {
		if _, ok := possibleGVKs[gvk]; !ok {
			possibleGVKs[gvk] = true
			fmt.Println("Registered type: ", gvk.Group, gvk.Kind, gvk.Version)
		}
	}
}

// NewResourceIdent is a helper function that returns a ResourceIdent object.
func NewSingleResourceIdent(provider string, purpose string, object client.Object, opts ...ResourceOptions) ResourceIdentSingle {
	registerGVK(object)
	writeNow := false
	for _, opt := range opts {
		writeNow = opt.WriteNow
	}

	return ResourceIdentSingle{
		Provider: provider,
		Purpose:  purpose,
		Type:     object,
		WriteNow: writeNow,
	}
}

// NewResourceIdent is a helper function that returns a ResourceIdent object.
func NewMultiResourceIdent(provider string, purpose string, object client.Object, opts ...ResourceOptions) ResourceIdentMulti {
	registerGVK(object)
	writeNow := false
	for _, opt := range opts {
		writeNow = opt.WriteNow
	}

	return ResourceIdentMulti{
		Provider: provider,
		Purpose:  purpose,
		Type:     object,
		WriteNow: writeNow,
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
	Object   client.Object
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
// blank object is stored in the cache it is imperative that the user of this function call Create
// before modifying the obejct they wish to be placed in the cache.
func (o *ObjectCache) Create(resourceIdent ResourceIdent, nn types.NamespacedName, object client.Object) error {

	update, err := utils.UpdateOrErr(o.client.Get(o.ctx, nn, object))

	if err != nil {
		return err
	}

	if _, ok := o.data[resourceIdent][nn]; ok {
		return fmt.Errorf("cannot create: ident store [%s] already has item named [%s]", resourceIdent, nn)
	}

	var gvk, obGVK schema.GroupVersionKind
	if gvk, err = utils.GetKindFromObj(o.scheme, resourceIdent.GetType()); err != nil {
		return err
	}

	if obGVK, err = utils.GetKindFromObj(o.scheme, object); err != nil {
		return err
	}

	if gvk != obGVK {
		return fmt.Errorf("create: resourceIdent type does not match runtime object [%s] [%s] [%s]", nn, gvk, obGVK)
	}

	if _, ok := o.resourceTracker[gvk]; !ok {
		o.resourceTracker[gvk] = map[types.NamespacedName]bool{nn: true}
	}

	o.resourceTracker[gvk][nn] = true

	if _, ok := o.data[resourceIdent]; !ok {
		o.data[resourceIdent] = make(map[types.NamespacedName]*k8sResource)
	}

	var jsonData []byte
	if clowderconfig.LoadedConfig.DebugOptions.Cache.Create || clowderconfig.LoadedConfig.DebugOptions.Cache.Apply {
		jsonData, _ = json.MarshalIndent(object, "", "  ")
	}

	o.data[resourceIdent][nn] = &k8sResource{
		Object:   object.DeepCopyObject().(client.Object),
		Update:   update,
		Status:   false,
		jsonData: string(jsonData),
	}

	if clowderconfig.LoadedConfig.DebugOptions.Cache.Create {
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
func (o *ObjectCache) Update(resourceIdent ResourceIdent, object client.Object) error {
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

	var gvk, obGVK schema.GroupVersionKind
	if gvk, err = utils.GetKindFromObj(o.scheme, resourceIdent.GetType()); err != nil {
		return err
	}

	if obGVK, err = utils.GetKindFromObj(o.scheme, object); err != nil {
		return err
	}

	if gvk != obGVK {
		return fmt.Errorf("create: resourceIdent type does not match runtime object [%s] [%s] [%s]", nn, gvk, obGVK)
	}

	o.data[resourceIdent][nn].Object = object.DeepCopyObject().(client.Object)

	if clowderconfig.LoadedConfig.DebugOptions.Cache.Update {
		var jsonData []byte
		jsonData, _ = json.MarshalIndent(o.data[resourceIdent][nn].Object, "", "  ")
		if object.GetObjectKind().GroupVersionKind() == secretCompare {
			o.log.Info("UPDATE resource ", "namespace", nn.Namespace, "name", nn.Name, "provider", resourceIdent.GetProvider(), "purpose", resourceIdent.GetPurpose(), "kind", object.GetObjectKind().GroupVersionKind().Kind, "diff", "hidden")
		} else {
			o.log.Info("UPDATE resource ", "namespace", nn.Namespace, "name", nn.Name, "provider", resourceIdent.GetProvider(), "purpose", resourceIdent.GetPurpose(), "kind", object.GetObjectKind().GroupVersionKind().Kind, "diff", string(jsonData))
		}
	}

	if resourceIdent.GetWriteNow() {
		o.log.Info("INSTANT APPLY resource ", "namespace", nn.Namespace, "name", nn.Name, "provider", resourceIdent.GetProvider(), "purpose", resourceIdent.GetPurpose(), "kind", object.GetObjectKind().GroupVersionKind().Kind, "update", o.data[resourceIdent][nn].Update)

		i := o.data[resourceIdent][nn]
		if err := i.Update.Apply(o.ctx, o.client, i.Object); err != nil {
			return err
		}
		if i.Status {
			if err := o.client.Status().Update(o.ctx, i.Object); err != nil {
				return err
			}
		}
	}

	return nil
}

// Get pulls the item from the cache and populates the given empty object. An error is returned if
// the items are of different types and also if the item is not in the cache. A get should be used
// by a downstream provider. If modifications are made to the object, it should be updated using the
// Update call.
func (o *ObjectCache) Get(resourceIdent ResourceIdent, object client.Object, nn ...types.NamespacedName) error {
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
func (o *ObjectCache) Status(resourceIdent ResourceIdent, object client.Object) error {
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
		if k.GetWriteNow() {
			continue
		}
		for n, i := range v {
			o.log.Info("APPLY resource ", "namespace", n.Namespace, "name", n.Name, "provider", k.GetProvider(), "purpose", k.GetPurpose(), "kind", i.Object.GetObjectKind().GroupVersionKind().Kind, "update", i.Update)
			if clowderconfig.LoadedConfig.DebugOptions.Cache.Apply {
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

func getNamespacedNameFromRuntime(object client.Object) (types.NamespacedName, error) {
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
