package v1alpha1

func (instance *CyndiPipeline) GetUIDString() string {
	return string(instance.GetUID())
}
