// Package certificateauthority provides certificate authority management functionality for Clowder applications
package certificateauthority

import (
	"fmt"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/object"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"

	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
	"github.com/RedHatInsights/rhc-osdk-utils/utils"
)

// CoreEnvCASecrets is the CA secrets resource identifier
var CoreEnvCASecrets = rc.NewMultiResourceIdent(ProvName, "core_env_ca_secrets", &core.Secret{}, rc.ResourceOptions{WriteNow: true})

type certificateAuthorityProvider struct {
	providers.Provider
}

// NewCertificateAuthorityProvider returns a new certificate authority provider
func NewCertificateAuthorityProvider(p *providers.Provider) (providers.ClowderProvider, error) {
	p.Cache.AddPossibleGVKFromIdent(
		CoreEnvCASecrets,
	)
	return &certificateAuthorityProvider{Provider: *p}, nil
}

// EnvProvide validates that CA secrets exist and creates a bundle secret containing all CAs
func (ca *certificateAuthorityProvider) EnvProvide() error {
	// Validate reserved name not used
	for _, certAuth := range ca.Env.Spec.Providers.Web.TLS.CertificateAuthorities {
		if certAuth.Name == "system-trust-store" {
			return errors.NewClowderError("certificateAuthorities cannot contain the reserved name 'system-trust-store'")
		}
	}

	// If no certificate authorities defined, nothing to do
	if len(ca.Env.Spec.Providers.Web.TLS.CertificateAuthorities) == 0 {
		return nil
	}

	// Collect all CA certificates into a bundle
	bundleData := make(map[string][]byte)

	for _, certAuth := range ca.Env.Spec.Providers.Web.TLS.CertificateAuthorities {
		// Read source secret
		sourceSecret := &core.Secret{}
		err := ca.Client.Get(ca.Ctx, types.NamespacedName{
			Name:      certAuth.Name,
			Namespace: certAuth.Namespace,
		}, sourceSecret)
		if err != nil {
			return errors.Wrap(fmt.Sprintf("certificateAuthority '%s' references secret that does not exist in namespace '%s'", certAuth.Name, certAuth.Namespace), err)
		}

		// Extract ca.crt from the secret and add to bundle with the CA name as key
		if caCert, exists := sourceSecret.Data["ca.crt"]; exists {
			bundleData[fmt.Sprintf("%s.crt", certAuth.Name)] = caCert
		} else {
			return errors.NewClowderError(fmt.Sprintf("certificateAuthority secret '%s' in namespace '%s' does not contain 'ca.crt' key", certAuth.Name, certAuth.Namespace))
		}
	}

	// Create the bundle secret in the environment's target namespace
	bundleSecretName := fmt.Sprintf("%s-ca-bundle", ca.Env.Name)
	bundleSecretNN := types.NamespacedName{
		Name:      bundleSecretName,
		Namespace: ca.Env.Spec.TargetNamespace,
	}

	bundleSecret := &core.Secret{}
	if err := ca.Cache.Get(CoreEnvCASecrets, bundleSecret, bundleSecretNN); err != nil {
		if err := ca.Cache.Create(CoreEnvCASecrets, bundleSecretNN, bundleSecret); err != nil {
			return err
		}
	}

	// Update bundle secret data
	bundleSecret.Data = bundleData
	bundleSecret.Type = core.SecretTypeOpaque

	labeler := utils.GetCustomLabeler(map[string]string{}, bundleSecretNN, ca.Env)
	labeler(bundleSecret)

	bundleSecret.Name = bundleSecretNN.Name
	bundleSecret.Namespace = bundleSecretNN.Namespace

	if err := ca.Cache.Update(CoreEnvCASecrets, bundleSecret); err != nil {
		return err
	}

	return nil
}

// Provide validates CA selection and copies bundle to app namespace if needed
func (ca *certificateAuthorityProvider) Provide(app *crd.ClowdApp) error {
	// Case 1: App uses override secret - validate it exists
	if app.Spec.TLSCertificateAuthoritySecretRef != nil {
		secret := &core.Secret{}
		err := ca.Client.Get(ca.Ctx, types.NamespacedName{
			Name:      app.Spec.TLSCertificateAuthoritySecretRef.Name,
			Namespace: app.Namespace,
		}, secret)
		if err != nil {
			return errors.Wrap(fmt.Sprintf("tlsCertificateAuthoritySecretRef '%s' does not exist in namespace '%s'", app.Spec.TLSCertificateAuthoritySecretRef.Name, app.Namespace), err)
		}
		return nil
	}

	// Case 2: No CA specified - use default (nothing to do)
	if app.Spec.TLSCertificateAuthorityName == nil {
		return nil
	}

	caName := *app.Spec.TLSCertificateAuthorityName

	// Case 3: System trust store - no bundle needed
	if caName == "system-trust-store" || caName == "" {
		return nil
	}

	// Case 4: CA name from environment bundle - validate and copy bundle if needed
	// Validate CA name exists in environment's certificateAuthorities list
	found := false
	for i := range ca.Env.Spec.Providers.Web.TLS.CertificateAuthorities {
		if ca.Env.Spec.Providers.Web.TLS.CertificateAuthorities[i].Name == caName {
			found = true
			break
		}
	}

	if !found {
		return errors.NewClowderError(fmt.Sprintf("tlsCertificateAuthorityName '%s' not found in ClowdEnvironment certificateAuthorities list", caName))
	}

	// Copy the bundle to app namespace if different from environment target namespace
	if app.Namespace != ca.Env.Spec.TargetNamespace {
		err := copyCABundle(&ca.Provider, app.Namespace, app)
		if err != nil {
			return errors.Wrap("failed to copy CA bundle", err)
		}
	}

	return nil
}

// copyCABundle copies the CA certificate bundle from environment's target namespace to app namespace
func copyCABundle(prov *providers.Provider, namespace string, obj object.ClowdObject) error {
	// Read source bundle secret from environment's target namespace
	bundleSecretName := fmt.Sprintf("%s-ca-bundle", prov.Env.Name)
	sourceSecretNN := types.NamespacedName{
		Name:      bundleSecretName,
		Namespace: prov.Env.Spec.TargetNamespace,
	}

	sourceSecret := &core.Secret{}
	err := prov.Client.Get(prov.Ctx, sourceSecretNN, sourceSecret)
	if err != nil {
		return errors.Wrap(fmt.Sprintf("failed to get CA bundle '%s' from namespace '%s'", bundleSecretName, prov.Env.Spec.TargetNamespace), err)
	}

	// Track with HashCache for ownership/garbage collection
	_, err = prov.HashCache.CreateOrUpdateObject(sourceSecret, true)
	if err != nil {
		return err
	}

	err = prov.HashCache.AddClowdObjectToObject(obj, sourceSecret)
	if err != nil {
		return err
	}

	// Create destination secret (same name as source bundle)
	destSecretNN := types.NamespacedName{
		Name:      bundleSecretName,
		Namespace: namespace,
	}

	// Create or get destination secret
	destSecret := &core.Secret{}
	if err := prov.Cache.Get(CoreEnvCASecrets, destSecret, destSecretNN); err != nil {
		if err := prov.Cache.Create(CoreEnvCASecrets, destSecretNN, destSecret); err != nil {
			return err
		}
	}

	// Copy data and metadata
	destSecret.Data = sourceSecret.Data
	destSecret.Type = sourceSecret.Type

	labeler := utils.GetCustomLabeler(map[string]string{}, destSecretNN, prov.Env)
	labeler(destSecret)

	destSecret.Name = destSecretNN.Name
	destSecret.Namespace = destSecretNN.Namespace

	// Update destination secret
	if err := prov.Cache.Update(CoreEnvCASecrets, destSecret); err != nil {
		return err
	}

	return nil
}
