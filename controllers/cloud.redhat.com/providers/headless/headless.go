package headless

import (
	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/config"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	deployProvider "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/deployment"
	rc "github.com/RedHatInsights/rhc-osdk-utils/resource_cache"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var ProvName = "headless"

var CoreService = rc.NewMultiResourceIdent(ProvName, "core_service", &core.Service{})

// GetHeadlessService is the register callback
func GetHeadlessService(c *providers.Provider) (providers.ClowderProvider, error) {
	return NewHeadlessService(c)
}

//Provider init code
func init() {
	providers.ProvidersRegistration.Register(GetHeadlessService, 10, ProvName)
}

type headlessService struct {
	providers.Provider
	Port       int32
	TargetPort int32
}

// NewHeadlessService returns a new headless service provider
func NewHeadlessService(p *providers.Provider) (providers.ClowderProvider, error) {
	return &headlessService{Provider: *p}, nil
}

//Provider API interface
func (h *headlessService) Provide(app *crd.ClowdApp, c *config.AppConfig) error {

	h.Port = h.Env.Spec.Providers.HeadlessService.Port
	h.TargetPort = h.Env.Spec.Providers.HeadlessService.TargetPort

	for _, deployment := range app.Spec.Deployments {

		if err := h.makeServiceForDeployment(&deployment, app); err != nil {
			return err
		}
	}
	return nil
}

func (h *headlessService) makeServiceForDeployment(deployment *crd.Deployment, app *crd.ClowdApp) error {
	service := &core.Service{}
	service.Spec.ClusterIP = "None"
	service.Spec.Ports = []core.ServicePort{{
		Protocol:   "TCP",
		Port:       h.Port,
		TargetPort: intstr.FromInt(int(h.TargetPort)),
	},
	}

	nn := app.GetDeploymentNamespacedName(deployment)

	if err := h.Cache.Create(CoreService, nn, service); err != nil {
		return err
	}

	d := &apps.Deployment{}

	h.Cache.Get(deployProvider.CoreDeployment, d, app.GetDeploymentNamespacedName(deployment))

	if err := h.Cache.Update(CoreService, service); err != nil {
		return err
	}

	if err := h.Cache.Update(deployProvider.CoreDeployment, d); err != nil {
		return err
	}

	return nil
}
