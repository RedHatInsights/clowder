package status

import (
	"context"

	cond "sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatusSourceFigures struct {
	ManagedDeployments int32
	ReadyDeployments   int32
	ManagedTopics      int32
	ReadyTopics        int32
}

//Defines an interface for objects that want to participate in the status system
type StatusSource interface {
	SetStatusReady(bool)
	GetNamespaces(ctx context.Context, client client.Client) ([]string, error)
	SetDeploymentFigures(StatusSourceFigures)
	AreDeploymentsReady(StatusSourceFigures) bool
	GetObjectSpecificFigures(context.Context, client.Client) (StatusSourceFigures, string, error)
	//Why is this part of the interface and not just some function here in status or something?
	//The thinking is that each object can decide what part of the figures struct it cares
	//about. This allows us to encapsulate the implementation and get some more polymoprhism out
	//of the code
	AddDeploymentFigures(StatusSourceFigures, StatusSourceFigures) StatusSourceFigures
	cond.Setter
}
