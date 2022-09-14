package headless

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var ProvName = "headless"
var CoreService = rc.NewMultiResourceIdent(ProvName, "core_service", &core.Service{})

//init is the Provider init code
func init() {
	providers.ProvidersRegistration.Register(GetHeadlessServiceProvider, 10, ProvName)
}

// GetHeadlessService is the register callback
func GetHeadlessServiceProvider(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewHeadlessServiceProvider(c)
}

// NewHeadlessService returns a new headless service provider
func NewHeadlessServiceProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	return &headlessServiceProvider{Provider: *p}, nil
}

//This is the provider. It loops through the deployments and creates a headless service for each one
type headlessServiceProvider struct {
	providers.Provider
}

//deploymentHasHeadlessService returns true if the deployment has a headless service
func (h *headlessServiceProvider) deploymentHasHeadlessService(deployment *crd.Deployment) bool {
	return deployment.HeadlessService != nil
}

//Provide is the Provider API interface
func (h *headlessServiceProvider) Provide(app *crd.ClowdApp, c *config.AppConfig) error {
	for _, deployment := range app.Spec.Deployments {
		if !h.deploymentHasHeadlessService(&deployment) {
			continue
		}
		headlessService := makeHeadlessService(&deployment, app, h)
		if err := headlessService.MakeHeadlessService(); err != nil {
			return err
		}
	}
	return nil
}

//makeHeadlessService is the constructor for headlessService struct
func makeHeadlessService(deployment *crd.Deployment, app *crd.ClowdApp, provider *headlessServiceProvider) headlessService {
	return headlessService{
		deployment: deployment,
		app:        app,
		provider:   provider,
	}
}

//This struct represents the headless service instances
type headlessService struct {
	deployment     *crd.Deployment
	app            *crd.ClowdApp
	provider       *headlessServiceProvider
	service        *core.Service
	coreDeployment *apps.Deployment
}

//MakeHeadlessService is the main function for creating the headless service
func (h *headlessService) MakeHeadlessService() error {
	h.makeService()

	h.setServiceNN()

	h.cacheService()

	h.getCoreDeployment()

	h.setServiceLabels()

	if err := h.updateServiceCache(); err != nil {
		return err
	}

	return nil
}

//makeService makes the core service
func (h *headlessService) makeService() {
	h.service = &core.Service{}
	h.service.Spec.ClusterIP = "None"
	h.service.Spec.Ports = []core.ServicePort{{
		Protocol:   "TCP",
		Port:       h.deployment.HeadlessService.Port,
		TargetPort: intstr.FromInt(int(h.deployment.HeadlessService.TargetPort)),
	},
	}
}

//serviceName returns the name of the service
func (h *headlessService) serviceName() string {
	return fmt.Sprintf("%s-%s-headless-service", h.app.Name, h.deployment.Name)
}

//getServiceNN returns the namespaced name of the service
func (h *headlessService) getServiceNN() types.NamespacedName {
	return types.NamespacedName{
		Name:      h.serviceName(),
		Namespace: h.app.Namespace,
	}
}

//setServiceNN sets the namespaced name of the service
func (h *headlessService) setServiceNN() {
	service_nn := h.getServiceNN()
	h.service.SetName(service_nn.Name)
	h.service.SetNamespace(service_nn.Namespace)
}

//cacheService creates the cache of the service
func (h *headlessService) cacheService() error {
	return h.provider.Cache.Create(CoreService, h.getServiceNN(), h.service)
}

//getCoreDeployment gets the core deployment
func (h *headlessService) getCoreDeployment() {
	h.coreDeployment = &apps.Deployment{}
	h.provider.Cache.Get(deployProvider.CoreDeployment, h.coreDeployment, h.app.GetDeploymentNamespacedName(h.deployment))

}

//setServiceLabels sets the labels of the service to point back to the deployment
func (h *headlessService) setServiceLabels() {
	deploymentLabels := h.coreDeployment.GetObjectMeta().GetLabels()
	h.service.Spec.Selector = map[string]string{"app": deploymentLabels["app"]}
}

//updateServiceCache updates the service cache
func (h *headlessService) updateServiceCache() error {
	return h.provider.Cache.Update(CoreService, h.service)
}
