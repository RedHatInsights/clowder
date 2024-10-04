package sizingconfig

const (
	// The nature of the beast in this code is we got a lot of magic strings
	SizeXSmall string = "x-small"
	SizeSmall  string = "small"
	SizeMedium string = "medium"
	SizeLarge  string = "large"
	SizeXLarge string = "x-large"
	// We need to define default sizes because if a ClowdApp doesn't provide
	// volume or ram/cpu capacities we just get an empty string, so we need
	// defaults to plug in there
	DefaultSizeVol    = SizeXSmall
	DefaultSizeCPURAM = SizeXSmall
)

// Get a map of CPU T-Shirt sizes to capacities
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

// For any given size get the next size up
// Allows for size to limit mapping without conditionality
var LimitSizeToRequestSize = map[string]string{
	SizeXSmall: SizeSmall,
	SizeSmall:  SizeMedium,
	SizeMedium: SizeLarge,
	SizeLarge:  SizeXLarge,
}

// Get a map of RAM T-Shirt sizes to capacities
var RAMSizeToCapacity = map[string]string{
	SizeXSmall: "512Mi",
	SizeSmall:  "1Gi",
	SizeMedium: "2Gi",
	SizeLarge:  "3Gi",
	SizeXLarge: "4Gi",
}

// Get a map of volume T-Shirt size to capacities
var VolSizeToCapacity = map[string]string{
	// x-small is because volume t shirt sizes pre-exist this implementation and there
	// we shipped a default smaller than small. I'm just leaving that pattern intact
	// In real life no one requests x-small, they request "" and get x-small
	SizeXSmall: "1Gi",
	SizeSmall:  "2Gi",
	SizeMedium: "3Gi",
	SizeLarge:  "5Gi",
}

// This is used for performing size comparisons
var SizeIndex = map[string]int{
	SizeXSmall: 0,
	SizeSmall:  1,
	SizeMedium: 2,
	SizeLarge:  3,
	SizeXLarge: 4,
}
