package kafka

import (
	"context"
	"fmt"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	cyndi "cloud.redhat.com/clowder/v2/apis/cyndi-operator/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	core "k8s.io/api/core/v1"
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

	// TODO: potentially do several more validation checks here to ensure the CyndiPipeline
	// is configured properly
	pipeline := cyndi.CyndiPipeline{}
	err := cl.Get(ctx, nn, &pipeline)

	if err != nil {
		return &errors.MissingDependencies{
			MissingDeps: map[string][]string{"cyndiPipeline": {nn.Name}},
		}
	}

	return nil
}

// create a CyndiPipeline resource
func createCyndiPipeline(
	ctx context.Context,
	cl client.Client,
	app *crd.ClowdApp,
	env *crd.ClowdEnvironment,
	connectClusterNamespace string,
	connectClusterName string,
) error {
	if cl == nil {
		// skip if within test suite
		return nil
	}

	appName := app.Spec.Cyndi.AppName
	if appName == "" {
		appName = app.Name
	}

	inventoryDbSecret, err := createCyndiInventoryDbSecret(ctx, cl, app, env, connectClusterNamespace)
	if err != nil {
		return err
	}

	appDbSecret, err := createCyndiAppDbSecret(ctx, cl, app, env, connectClusterNamespace)
	if err != nil {
		return err
	}

	nn := types.NamespacedName{
		Namespace: connectClusterNamespace,
		Name:      appName,
	}

	pipeline := cyndi.CyndiPipeline{}

	update, err := utils.UpdateOrErr(cl.Get(ctx, nn, &pipeline))
	if err != nil {
		return err
	}

	pipeline.SetNamespace(connectClusterNamespace)
	pipeline.SetName(appName)
	pipeline.Spec.AppName = appName
	pipeline.Spec.InventoryDbSecret = &inventoryDbSecret
	pipeline.Spec.DbSecret = &appDbSecret
	pipeline.Spec.ConnectCluster = &connectClusterName
	pipeline.Spec.InsightsOnly = app.Spec.Cyndi.InsightsOnly

	// it would be best for the ClowdApp to own this, but since cross-namespace OwnerReferences
	// are not permitted, make this owned by the ClowdEnvironment
	pipeline.SetOwnerReferences([]metav1.OwnerReference{env.MakeOwnerReference()})

	err = update.Apply(ctx, cl, &pipeline)
	if err != nil {
		return err
	}

	return nil
}

func getDbSecretInSameEnv(ctx context.Context, cl client.Client, app *crd.ClowdApp, name string) (*core.Secret, error) {
	// locate the clowdapp named 'name' in the same env as 'app' and return its DB secret
	// TODO: switch this to use cache instead of getting secret out of k8s?
	appList := &crd.ClowdAppList{}

	err := crd.GetAppInSameEnv(ctx, cl, app, appList)
	if err != nil {
		return nil, errors.Wrap("unable to list ClowdApps in environment", err)
	}

	foundMatchingApps := []crd.ClowdApp{}
	for _, foundApp := range appList.Items {
		if foundApp.ObjectMeta.Name == name {
			foundMatchingApps = append(foundMatchingApps, foundApp)
		}
	}

	if len(foundMatchingApps) == 0 {
		return nil, errors.New(fmt.Sprintf("unable to locate '%s' ClowdApp in environment", name))
	}

	if len(foundMatchingApps) > 1 {
		// this shouldn't happen, but check just in case...
		return nil, errors.New(fmt.Sprintf("found more than one '%s' ClowdApp in environment", name))
	}

	refApp := foundMatchingApps[0]

	// get the db secret out of the clowdapp's namespace
	dbSecret := &core.Secret{}
	nn := types.NamespacedName{
		Name:      fmt.Sprintf("%s-db", name),
		Namespace: refApp.Namespace,
	}

	err = cl.Get(ctx, nn, dbSecret)

	if err != nil {
		return nil, errors.Wrap(fmt.Sprintf("couldn't get '%s' secret", nn.Name), err)
	}

	return dbSecret, nil
}

func applySecretToConnectNamespace(
	ctx context.Context,
	cl client.Client,
	env *crd.ClowdEnvironment,
	secretName string,
	connectClusterNamespace string,
	secretData map[string]string,
) error {
	outNN := types.NamespacedName{
		Name:      secretName,
		Namespace: connectClusterNamespace,
	}

	secret := &core.Secret{}

	update, err := utils.UpdateOrErr(cl.Get(ctx, outNN, secret))
	if err != nil {
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
	secret.SetOwnerReferences([]metav1.OwnerReference{env.MakeOwnerReference()})

	if err := update.Apply(ctx, cl, secret); err != nil {
		return err
	}

	return nil
}

// create a secret that tells the cyndi operator how to connect to an app's db
func createCyndiAppDbSecret(
	ctx context.Context,
	cl client.Client,
	app *crd.ClowdApp,
	env *crd.ClowdEnvironment,
	connectClusterNamespace string,
) (string, error) {
	dbSecret, err := getDbSecretInSameEnv(ctx, cl, app, app.Name)
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

	err = applySecretToConnectNamespace(ctx, cl, env, secretName, connectClusterNamespace, secretData)
	if err != nil {
		return "", errors.Wrap("couldn't apply cyndi db secret for app", err)
	}

	return secretName, nil
}

// create a secret that tells the cyndi operator how to connecto to host-inventory's db
func createCyndiInventoryDbSecret(
	ctx context.Context,
	cl client.Client,
	app *crd.ClowdApp,
	env *crd.ClowdEnvironment,
	connectClusterNamespace string,
) (string, error) {
	dbSecret, err := getDbSecretInSameEnv(ctx, cl, app, "host-inventory")
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
	err = applySecretToConnectNamespace(ctx, cl, env, secretName, connectClusterNamespace, secretData)
	if err != nil {
		return "", errors.Wrap("couldn't apply cyndi db secret for host-inventory", err)
	}

	return secretName, nil
}
