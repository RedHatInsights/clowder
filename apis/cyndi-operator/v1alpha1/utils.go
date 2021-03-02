package v1alpha1

// GetUIDString returns the UID from the CynciPipeline instance.
func (instance *CyndiPipeline) GetUIDString() string {
	return string(instance.GetUID())
}
