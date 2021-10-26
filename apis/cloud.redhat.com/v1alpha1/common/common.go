package common

// Int32Ptr returns a pointer to an int32 version of n
func Int32Ptr(n int) *int32 {
	t, err := Int32(n)
	if err != nil {
		panic(err)
	}
	return &t
}

// Int64Ptr returns a pointer to an int64
func Int64Ptr(n int64) *int64 {
	return &n
}

// TruePtr returns a pointer to True
func TruePtr() *bool {
	t := true
	return &t
}

// FalsePtr returns a pointer to True
func FalsePtr() *bool {
	f := false
	return &f
}

// StringPtr returns a pointer to True
func StringPtr(str string) *string {
	s := str
	return &s
}
