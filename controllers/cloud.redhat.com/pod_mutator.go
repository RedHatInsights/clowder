package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	core "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type mutantPod struct {
	Client   client.Client
	Recorder record.EventRecorder
	decoder  *admission.Decoder
}

func (p *mutantPod) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &core.Pod{}

	err := p.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if v, ok := pod.GetAnnotations()["authsidecar"]; ok && v == "enabled" {
		ridx := -1
		for idx, container := range pod.Spec.Containers {
			if container.Name == "crcauth" {
				ridx = idx
				break
			}
		}

		port, ok := pod.GetAnnotations()["authsidecar/port"]
		if !ok {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("pod does not specify service port"))
		}

		container := core.Container{
			Name:  "crcauth",
			Image: "127.0.0.1:5000/crccaddy:2",
			Env: []core.EnvVar{{
				Name:  "CADDY_PORT",
				Value: port,
			}},
		}

		if ridx == -1 {
			pod.Spec.Containers = append(pod.Spec.Containers, container)
		} else {
			pod.Spec.Containers[ridx] = container
		}
	}

	marshaledObj, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledObj)
}

func (a *mutantPod) InjectDecoder(d *admission.Decoder) error {
	a.decoder = d
	return nil
}
