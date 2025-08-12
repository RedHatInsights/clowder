package sizing

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"

	conf "github.com/RedHatInsights/clowder/controllers/cloud.redhat.com/providers/sizing/sizingconfig"
)

func TestGetLimitSizeForRequestSize(t *testing.T) {
	assert.Equal(t, conf.LimitSizeToRequestSize["small"], "medium")
	assert.Equal(t, conf.LimitSizeToRequestSize["medium"], "large")
	assert.Equal(t, conf.LimitSizeToRequestSize["large"], "x-large")
}

func TestGetVolSizeToCapacityMap(t *testing.T) {
	s := conf.VolSizeToCapacity
	assert.Equal(t, s["x-small"], "1Gi")
	assert.Equal(t, s["small"], "2Gi")
	assert.Equal(t, s["medium"], "3Gi")
	assert.Equal(t, s["large"], "5Gi")
}

func TestGetCPUSizeToCapacityMap(t *testing.T) {
	c := conf.CPUSizeToCapacity
	assert.Equal(t, c["small"], "600m")
	assert.Equal(t, c["medium"], "1200m")
	assert.Equal(t, c["large"], "1800m")
	assert.Equal(t, c["x-large"], "2400m")
}

func TestGetRAMSizeToCapacityMap(t *testing.T) {
	r := conf.RAMSizeToCapacity
	assert.Equal(t, r["small"], "1Gi")
	assert.Equal(t, r["medium"], "2Gi")
	assert.Equal(t, r["large"], "3Gi")
	assert.Equal(t, r["x-large"], "4Gi")
}

func TestGetDefaultResourceRequirements(t *testing.T) {
	reqs := GetDefaultResourceRequirements()

	ramSmall := conf.RAMSizeToCapacity["x-small"]
	cpuSmall := conf.CPUSizeToCapacity["x-small"]
	ramMed := conf.RAMSizeToCapacity["small"]
	cpuMed := conf.CPUSizeToCapacity["small"]

	assert.Equal(t, reqs.Limits["memory"], resource.MustParse(ramMed))
	assert.Equal(t, reqs.Limits["cpu"], resource.MustParse(cpuMed))
	assert.Equal(t, reqs.Requests["memory"], resource.MustParse(ramSmall))
	assert.Equal(t, reqs.Requests["cpu"], resource.MustParse(cpuSmall))
}

func TestDBDResourceRequirements(t *testing.T) {
	reqs := GetResourceRequirementsForSize("medium")

	ramLarge := conf.RAMSizeToCapacity["large"]
	cpuLarge := conf.CPUSizeToCapacity["large"]
	ramMed := conf.RAMSizeToCapacity["medium"]
	cpuMed := conf.CPUSizeToCapacity["medium"]

	assert.Equal(t, reqs.Limits["memory"], resource.MustParse(ramLarge))
	assert.Equal(t, reqs.Limits["cpu"], resource.MustParse(cpuLarge))
	assert.Equal(t, reqs.Requests["memory"], resource.MustParse(ramMed))
	assert.Equal(t, reqs.Requests["cpu"], resource.MustParse(cpuMed))
}

func TestGetDefaultVolCapacity(t *testing.T) {
	d := GetDefaultVolCapacity()
	dd := conf.VolSizeToCapacity[GetDefaultSizeVol()]
	assert.Equal(t, d, dd)
}
