package sizingconfig

const (
	//The nature of the beast in this code is we got a lot of magic strings
	XSMALL string = "x-small"
	SMALL  string = "small"
	MEDIUM string = "medium"
	LARGE  string = "large"
	XLARGE string = "x-large"
	//We need to define default sizes because if a ClowdApp doesn't provide
	//volume or ram/cpu capacities we just get an empty string, so we need
	//defaults to plug in there
	DEFAULT_SIZE_VOL     string = XSMALL
	DEFAULT_SIZE_CPU_RAM string = SMALL
)

//Get a map of CPU T-Shirt sizes to capacities
var CPUSizeToCapacity = map[string]string{
	SMALL:  "600m",
	MEDIUM: "1200m",
	LARGE:  "1800m",
	//Why x-large? For CPU and RAM we have a request and a limit. The limit needs to be
	//larger than the request. Therefore, if large is requested we need an x-large as a
	//limit. x-large can't be requested - it isn't part of the config enum valid value set
	XLARGE: "2400m",
}

//For any given size get the next size up
//Allows for size to limit mapping without conditionality
var LimitSizeToRequestSize = map[string]string{
	XSMALL: SMALL,
	SMALL:  MEDIUM,
	MEDIUM: LARGE,
	LARGE:  XLARGE,
}

//Get a map of RAM T-Shirt sizes to capacities
var RAMSizeToCapacity = map[string]string{
	SMALL:  "512Mi",
	MEDIUM: "1Gi",
	LARGE:  "2Gi",
	XLARGE: "3Gi",
}

//Get a map of volume T-Shirt size to capacities
var VolSizeToCapacity = map[string]string{
	//x-small is because volume t shirt sizes pre-exist this implementation and there
	//we shipped a default smaller than small. I'm just leaving that pattern intact
	//In real life no one requests x-small, they request "" and get x-small
	XSMALL: "1Gi",
	SMALL:  "2Gi",
	MEDIUM: "3Gi",
	LARGE:  "5Gi",
}

//This is used for performing size comparisons
var SizeIndex = map[string]int{
	XSMALL: 0,
	SMALL:  1,
	MEDIUM: 2,
	LARGE:  3,
	XLARGE: 4,
}
