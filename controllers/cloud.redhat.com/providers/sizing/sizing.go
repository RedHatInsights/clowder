// Package sizing provides resource sizing and scaling functionality for Clowder applications
package sizing

// Note on package:
// I didn't really want to pull this out into its own package
// I wanted this in database or providers but I ran into dependency cycle problems
// no matter what I did. So easiest and cleanest solution was just to pull it out
// that said maybe in the future we can extend sizing out to other stuff in which case
// a sizing package will be helpful

// Note on naming:
// Naming is hard. In the context of this API "sizes" are
// the t shirt sizes (small, medium, etc) whereas "capacities"
// are the values k8s uses like Gi, M, m, etc. This distinction is
// important because most of what we're doing here is
// converting sizes to capacities, so I enforce the distinction
// strictly. The method names are long, but more importantly, accurate

import (
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	conf "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/sizing/sizingconfig"
)

// Public methods

// Get default resource requirement requests and limits
func GetDefaultResourceRequirements() core.ResourceRequirements {
	return GetResourceRequirementsForSize(GetDefaultSizeCPURAM())
}

// Gets the default size for CPU and RAM
func GetDefaultSizeCPURAM() string {
	return conf.DefaultSizeCPURAM
}

// Gets the default vol size
func GetDefaultSizeVol() string {
	return conf.DefaultSizeVol
}

// Get the default volume capacity
func GetDefaultVolCapacity() string {
	return conf.VolSizeToCapacity[GetDefaultSizeVol()]
}

// Get resource requirements - request and limits - for a given size
func GetResourceRequirementsForSize(tShirtSize string) core.ResourceRequirements {
	requestSize := useDefaultIfEmptySize(tShirtSize, GetDefaultSizeCPURAM())
	limitSize := conf.LimitSizeToRequestSize[requestSize]
	return core.ResourceRequirements{
		Limits: core.ResourceList{
			"memory": resource.MustParse(conf.RAMSizeToCapacity[limitSize]),
			"cpu":    resource.MustParse(conf.CPUSizeToCapacity[limitSize]),
		},
		Requests: core.ResourceList{
			"memory": resource.MustParse(conf.RAMSizeToCapacity[requestSize]),
			"cpu":    resource.MustParse(conf.CPUSizeToCapacity[requestSize]),
		},
	}
}

// For a givin vol size get the capacity. Providing "" gets the default.
func GetVolCapacityForSize(size string) string {
	requestSize := useDefaultIfEmptySize(size, GetDefaultSizeVol())
	return conf.VolSizeToCapacity[requestSize]
}

// Accepts 2 sizes. Returns true if first size is larger than second
func IsSizeLarger(capacityA string, capacityB string) bool {
	return conf.SizeIndex[capacityA] > conf.SizeIndex[capacityB]
}

// Private methods

// Often we have to sanitize a size such that "" == whatever the default is
func useDefaultIfEmptySize(size string, def string) string {
	if size == "" {
		return def
	}
	return size
}
