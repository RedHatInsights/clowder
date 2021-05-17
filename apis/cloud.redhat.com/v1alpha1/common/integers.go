package common

/*
MIT License

Copyright (c) 2020 Hiroki Suezawa

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

import (
	"errors"
	"strconv"
)

// Helpers to handle int32 conversions safely, from github.com/rung/go-safecast

// Int32 converts int to int32 in a safe way.
// You get error when the value is out of the 32-bit range.
func Int32(i int) (int32, error) {
	if i > 2147483647 || i < -2147483648 {
		return 0, errors.New("int32 out of range")
	}
	return int32(i), nil
}

// Atoi32 converts string to int32 in a safe way.
// You get error when the value is out of the 32-bit range.
//
// This is a wrapper function of strconv.ParseInt.
func Atoi32(s string) (int32, error) {
	i, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(i), nil
}
