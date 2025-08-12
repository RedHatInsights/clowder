package kafka

import (
	"context"
	"fmt"

	cyndi "github.com/RedHatInsights/cyndi-operator/api/v1alpha1"

	crd "github.com/RedHatInsights/clowder/apis/cloud.redhat.com/v1alpha1"
	"github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/errors"
	prov "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers"
	db "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/database"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	core "k8s.io/api/core/v1"

	rc "github.com/RedHatInsights/rhc-osdk-utils/resourceCache"
)

// ensures that a CyndiPipeline resource exists
func validateCyndiPipeline(
	ctx context.Context, cl client.Client, app *crd.ClowdApp, connectClusterNamespace string,
) error {
	if cl == nil {
		// skip if within test suite
		return nil
	}

	appName := app.Spec.Cyndi.AppName
	if appName == "" {
		appName = app.Name
	}

	nn := types.NamespacedName{
		Namespace: connectClusterNamespace,
		Name:      appName,
	}

	pipeline := cyndi.CyndiPipeline{}
	err := cl.Get(ctx, nn, &pipeline)

	if err != nil {
		missingDeps := errors.MakeMissingDependencies(errors.MissingDependency{
			Source:  "cyndi",
			Details: fmt.Sprintf("CyndiPipeline named '%s' not found in namespace '%s'", nn.Name, nn.Namespace),
		})
		return &missingDeps
	}

	if pipeline.Status.ActiveTableName == "" {
		missingDeps := errors.MakeMissingDependencies(errors.MissingDependency{
			Source:  "cyndi",
			Details: fmt.Sprintf("CyndiPipeline '%s' in namespace '%s' has no 'active table' in status", nn.Name, nn.Namespace),
		})
		return &missingDeps
	}

	return nil
}

// create a CyndiPipeline resource
func createCyndiPipeline(
	s prov.RootProvider,
	app *crd.ClowdApp,
	connectClusterNamespace string,
	connectClusterName string,
) error {
	if s.GetClient() == nil {
		// skip if within test suite
		return nil
	}

	appName := app.Spec.Cyndi.AppName
	if appName == "" {
		appName = app.Name
	}

	inventoryDbSecret, err := createCyndiInventoryDbSecret(s, app, connectClusterNamespace)
	if err != nil {
		return err
	}

	appDbSecret, err := createCyndiAppDbSecret(s, app, connectClusterNamespace)
	if err != nil {
		return err
	}

	nn := types.NamespacedName{
		Namespace: connectClusterNamespace,
		Name:      appName,
	}

	pipeline := &cyndi.CyndiPipeline{}

	if err := s.GetCache().Create(CyndiPipeline, nn, pipeline); err != nil {
		return err
	}

	pipeline.SetNamespace(connectClusterNamespace)
	pipeline.SetName(appName)
	pipeline.Spec.AppName = appName
	pipeline.Spec.InventoryDbSecret = &inventoryDbSecret
	pipeline.Spec.DbSecret = &appDbSecret
	pipeline.Spec.ConnectCluster = &connectClusterName
	pipeline.Spec.InsightsOnly = app.Spec.Cyndi.InsightsOnly
	pipeline.Spec.AdditionalFilters = app.Spec.Cyndi.AdditionalFilters

	// it would be best for the ClowdApp to own this, but since cross-namespace OwnerReferences
	// are not permitted, make this owned by the ClowdEnvironment
	pipeline.SetOwnerReferences([]metav1.OwnerReference{s.GetEnv().MakeOwnerReference()})

	return s.GetCache().Update(CyndiPipeline, pipeline)
}

func getDbSecretInSameEnv(s prov.RootProvider, app *crd.ClowdApp, name string) (*core.Secret, error) {
	// locate the clowdapp named 'name' in the same env as 'app' and return its DB secret
	appList := &crd.ClowdAppList{}

	err := crd.GetAppInSameEnv(s.GetCtx(), s.GetClient(), app, appList)
	if err != nil {
		return nil, errors.Wrap("unable to list ClowdApps in environment", err)
	}

	foundMatchingApps := []crd.ClowdApp{}
	for _, foundApp := range appList.Items {
		if foundApp.Name == name {
			foundMatchingApps = append(foundMatchingApps, foundApp)
		}
	}

	if len(foundMatchingApps) == 0 {
		return nil, errors.NewClowderError(fmt.Sprintf("unable to locate '%s' ClowdApp in environment", name))
	}

	if len(foundMatchingApps) > 1 {
		// this shouldn't happen, but check just in case...
		return nil, errors.NewClowderError(fmt.Sprintf("found more than one '%s' ClowdApp in environment", name))
	}

	refApp := foundMatchingApps[0]

	// get the db secret out of the clowdapp's namespace
	dbSecret := &core.Secret{}
	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-db", name),
		Namespace: refApp.Namespace,
	}

	if name == "host-inventory" {
		err = s.GetClient().Get(s.GetCtx(), nn, dbSecret)

		if err != nil {
			return nil, errors.Wrap(fmt.Sprintf("couldn't get '%s' secret", nn.Name), err)
		}
	} else {
		if app.Spec.Database.SharedDBAppName != "" {
			return nil, errors.NewClowderError("Shared DB app cannot use Cyndi")
		}
		switch s.GetEnv().Spec.Providers.Database.Mode {
		case "local":
			if err = s.GetCache().Get(db.LocalDBSecret, dbSecret); err != nil {
				return nil, errors.Wrap(fmt.Sprintf("couldn't get '%s' secret", nn.Name), err)
			}
		case "shared":
			if err = s.GetCache().Get(db.SharedDBAppSecret, dbSecret); err != nil {
				return nil, errors.Wrap(fmt.Sprintf("couldn't get '%s' secret", nn.Name), err)
			}
		case "app-interface":
			if app.Spec.Database.Name != "" {
				rdsCaBundleURL := s.GetEnv().Spec.Providers.Database.CaBundleURL
				dbConfig, err := db.GetDbConfig(s.GetCtx(), s.GetClient(), app.Namespace, app.Name, app.Spec.Database, rdsCaBundleURL)

				if err != nil {
					return nil, errors.Wrap("could not get database config", err)
				}

				err = s.GetClient().Get(s.GetCtx(), dbConfig.Ref, dbSecret)

				if err != nil {
					return nil, errors.Wrap(fmt.Sprintf("couldn't get '%s' secret", nn.Name), err)
				}
			}
		}
	}

	return dbSecret, nil
}

func applySecretToConnectNamespace(
	s prov.RootProvider,
	secretName string,
	connectClusterNamespace string,
	secretData map[string]string,
	resourceIdent rc.ResourceIdent,
) error {
	outNN := types.NamespacedName{
		Name:      secretName,
		Namespace: connectClusterNamespace,
	}

	secret := &core.Secret{}

	if err := s.GetCache().Create(resourceIdent, outNN, secret); err != nil {
		return err
	}

	// stringData allows specifying non-binary secret data in string form.
	// It is provided as a write-only convenience method.
	secret.StringData = secretData
	secret.Type = core.SecretTypeOpaque
	secret.SetName(outNN.Name)
	secret.SetNamespace(outNN.Namespace)
	// it would be best for the ClowdApp to own this, but since cross-namespace OwnerReferences
	// are not permitted, make this owned by the ClowdEnvironment
	secret.SetOwnerReferences([]metav1.OwnerReference{s.GetEnv().MakeOwnerReference()})

	return s.GetCache().Update(resourceIdent, secret)
}

// create a secret that tells the cyndi operator how to connect to an app's db
func createCyndiAppDbSecret(
	s prov.RootProvider,
	app *crd.ClowdApp,
	connectClusterNamespace string,
) (string, error) {
	dbSecret, err := getDbSecretInSameEnv(s, app, app.Name)
	if err != nil {
		return "", errors.Wrap(fmt.Sprintf("unable to get '%s' db secret", app.Name), err)
	}

	// create a secret in the kafka-connect cluster namespace to tell cyndi how to connect to this
	// application's DB
	secretName := fmt.Sprintf("%s-%s-db-cyndi", app.Spec.EnvName, app.Name)

	secretData := map[string]string{
		"db.host": string(dbSecret.Data["hostname"]),
		"db.port": string(dbSecret.Data["port"]),
		"db.name": string(dbSecret.Data["name"]),
		// these creds are hard-coded into the local postgres DB image we use for cyndi
		// (only in test envs). TODO: look into using env vars on the container and having
		// Clowder randomly generate these creds?
		"db.user":     "cyndi",
		"db.password": "cyndi",
	}

	err = applySecretToConnectNamespace(s, secretName, connectClusterNamespace, secretData, CyndiAppSecret)
	if err != nil {
		return "", errors.Wrap("couldn't apply cyndi db secret for app", err)
	}

	return secretName, nil
}

// create a secret that tells the cyndi operator how to connecto to host-inventory's db
func createCyndiInventoryDbSecret(
	s prov.RootProvider,
	app *crd.ClowdApp,
	connectClusterNamespace string,
) (string, error) {
	dbSecret, err := getDbSecretInSameEnv(s, app, "host-inventory")
	if err != nil {
		return "", errors.Wrap("unable to get host-inventory db secret", err)
	}

	// apply the same credentials into a secret residing in the kafka connect namespace
	secretName := fmt.Sprintf("%s-host-inventory-db-cyndi", app.Spec.EnvName)
	secretData := map[string]string{
		"db.host":     string(dbSecret.Data["hostname"]),
		"db.port":     string(dbSecret.Data["port"]),
		"db.name":     string(dbSecret.Data["name"]),
		"db.user":     string(dbSecret.Data["username"]),
		"db.password": string(dbSecret.Data["password"]),
	}
	err = applySecretToConnectNamespace(s, secretName, connectClusterNamespace, secretData, CyndiHostInventoryAppSecret)
	if err != nil {
		return "", errors.Wrap("couldn't apply cyndi db secret for host-inventory", err)
	}

	return secretName, nil
}
