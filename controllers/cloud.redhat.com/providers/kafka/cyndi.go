package kafka

import (
	"context"

	crd "cloud.redhat.com/clowder/v2/apis/cloud.redhat.com/v1alpha1"
	cyndi "cloud.redhat.com/clowder/v2/apis/cyndi-operator/v1alpha1"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/errors"
	"cloud.redhat.com/clowder/v2/controllers/cloud.redhat.com/utils"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ensures that a CyndiPipeline resource exists
func validateCyndiPipeline(
	ctx context.Context, cl client.Client, app *crd.ClowdApp, connectClusterNamespace string) error {
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
	ctx context.Context, cl client.Client, app *crd.ClowdApp, connectClusterNamespace string,
	connectClusterName string) error {
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

	update, err := utils.UpdateOrErr(cl.Get(ctx, nn, &pipeline))
	if err != nil {
		return err
	}

	pipeline.SetNamespace(connectClusterNamespace)
	pipeline.SetName(appName)
	pipeline.Spec.AppName = appName
	pipeline.Spec.ConnectCluster = &connectClusterName
	pipeline.Spec.InsightsOnly = app.Spec.Cyndi.InsightsOnly

	err = update.Apply(ctx, cl, &pipeline)
	if err != nil {
		return err
	}

	return nil
}
