// Package sizingconfig provides configuration management for resource sizing policies
package sizingconfig

const (
	// SizeXSmall represents the x-small size designation
	SizeXSmall string = "x-small"
	SizeSmall  string = "small"
	SizeMedium string = "medium"
	SizeLarge  string = "large"
	SizeXLarge string = "x-large"
	// DefaultSizeVol is the default volume size when none is specified
	DefaultSizeVol    = SizeXSmall
	DefaultSizeCPURAM = SizeXSmall
)

// CPUSizeToCapacity maps CPU T-Shirt sizes to their capacities
var CPUSizeToCapacity = map[string]string{
	SizeXSmall: "300m",
	SizeSmall:  "600m",
	SizeMedium: "1200m",
	SizeLarge:  "1800m",
	// Why x-large? For CPU and RAM we have a request and a limit. The limit needs to be
	// larger than the request. Therefore, if large is requested we need an x-large as a
	// limit. x-large can't be requested - it isn't part of the config enum valid value set
	SizeXLarge: "2400m",
}

// LimitSizeToRequestSize maps request sizes to their corresponding limit sizes
var LimitSizeToRequestSize = map[string]string{
	SizeXSmall: SizeSmall,
	SizeSmall:  SizeMedium,
	SizeMedium: SizeLarge,
	SizeLarge:  SizeXLarge,
}

// RAMSizeToCapacity maps RAM T-Shirt sizes to their capacities
var RAMSizeToCapacity = map[string]string{
	SizeXSmall: "512Mi",
	SizeSmall:  "1Gi",
	SizeMedium: "2Gi",
	SizeLarge:  "3Gi",
	SizeXLarge: "4Gi",
}

// VolSizeToCapacity maps volume T-Shirt sizes to their capacities
var VolSizeToCapacity = map[string]string{
	// x-small is because volume t shirt sizes pre-exist this implementation and there
	// we shipped a default smaller than small. I'm just leaving that pattern intact
	// In real life no one requests x-small, they request "" and get x-small
	SizeXSmall: "1Gi",
	SizeSmall:  "2Gi",
	SizeMedium: "3Gi",
	SizeLarge:  "5Gi",
}

// SizeIndex maps sizes to their numeric indices for comparison purposes
var SizeIndex = map[string]int{
	SizeXSmall: 0,
	SizeSmall:  1,
	SizeMedium: 2,
	SizeLarge:  3,
	SizeXLarge: 4,
}
