package nano

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The  parseDurationTests test vector is derived from the GO time package
// https://golang.org/pkg/time/
//
// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// We added the minDur test, which the Go library doesn't properly handle.

var parseDurationTests = []struct {
	in       string
	expected Duration
}{
	// simple
	{"0s", 0},
	{"5s", 5 * Second},
	{"30s", 30 * Second},
	{"1478s", 1478 * Second},
	// minu sign
	{"-5s", -5 * Second},
	{"-0s", 0},
	//{"+0", 0},
	// decimal
	{"5.0s", 5 * Second},
	{"5.6s", 5*Second + 600*Millisecond},
	{"5.s", 5 * Second},
	{".5s", 500 * Millisecond},
	{"1.0s", 1 * Second},
	{"1.00s", 1 * Second},
	{"1.004s", 1*Second + 4*Millisecond},
	{"1.0040s", 1*Second + 4*Millisecond},
	{"100.00100s", 100*Second + 1*Millisecond},
	// different units
	{"10ns", 10 * Nanosecond},
	{"11us", 11 * Microsecond},
	{"13ms", 13 * Millisecond},
	{"14s", 14 * Second},
	{"15m", 15 * Minute},
	{"16h", 16 * Hour},
	// composite durations
	{"3h30m", 3*Hour + 30*Minute},
	{"10.5s4m", 4*Minute + 10*Second + 500*Millisecond},
	{"-2m3.4s", -(2*Minute + 3*Second + 400*Millisecond)},
	{"1h2m3s4ms5us6ns", 1*Hour + 2*Minute + 3*Second + 4*Millisecond + 5*Microsecond + 6*Nanosecond},
	{"39h9m14.425s", 39*Hour + 9*Minute + 14*Second + 425*Millisecond},
	// large value
	{"52763797000ns", 52763797000 * Nanosecond},
	// more than 9 digits after decimal point, see https://golang.org/issue/6617
	{"0.3333333333333333333h", 20 * Minute},
	// 9007199254740993 = 1<<53+1 cannot be stored precisely in a float64
	{"9007199254740993ns", (1<<53 + 1) * Nanosecond},
	// largest duration that can be represented by int64 in nanoseconds
	{"9223372036854775807ns", (1<<63 - 1) * Nanosecond},
	{"9223372036854775.807us", (1<<63 - 1) * Nanosecond},
	{"9223372036s854ms775us807ns", (1<<63 - 1) * Nanosecond},
	// large negative value
	{"-9223372036854775807ns", -1<<63 + 1*Nanosecond},
	// huge string; issue 15011.
	{"0.100000000000000000000h", 6 * Minute},
	// This value tests the first overflow check in leadingFraction.
	{"0.830103483285477580700h", 49*Minute + 48*Second + 372539827*Nanosecond},
	// Min duration
	{minDur, Duration(math.MinInt64)},
}

func TestParseDurationFromGo(t *testing.T) {
	for _, test := range parseDurationTests {
		actual, err := ParseDuration(test.in)
		require.NoError(t, err)
		assert.Exactly(t, test.expected, actual, test.in)
	}
}

var boomerangs = []string{
	"1ns",
	"2us",
	"3ms",
	"4s",
	"5m",
	"6h",
	"8y",
	"2y3d5h24m1s",
	"100us",
	"123ns",
	"123ms",
	"1.3s",
	"0s",
	"31y259d1h46m40s",
	"2h43m9.993714061s",
}

func TestParseDuration(t *testing.T) {
	d, err := ParseDuration("1ms")
	require.NoError(t, err)
	assert.Exactly(t, Duration(time.Millisecond), d)
	for _, s := range boomerangs {
		checkdur(t, s, s)
	}
	checkdur(t, "1230ms", "1.23s")
	checkdur(t, "0ns", "0s")
	checkdur(t, "1s300ms", "1.3s")
	checkdur(t, "2716us", "2.716ms")
	checkdur(t, "1230us", "1.23ms")
	checkdur(t, "11230us", "11.23ms")
	checkdur(t, "111230us", "111.23ms")
	checkdur(t, "1234ns", "1.234us")
	//checkdur(t, "", "0s")
	checkdur(t, "1w", "7d")
	checkdur(t, "1.5w", "10d12h")
	checkdur(t, "1h", "1h")
}

func checkdur(t *testing.T, in, expected string) {
	d, err := ParseDuration(in)
	require.NoError(t, err)
	actual := d.String()
	assert.Exactly(t, expected, actual)
}

func TestMarshalDuration(t *testing.T) {
	for _, s := range boomerangs {
		d, err := ParseDuration(s)
		require.NoError(t, err)
		b, err := json.Marshal(&d)
		require.NoError(t, err)
		var actual Duration
		err = json.Unmarshal(b, &actual)
		require.NoError(t, err)
		assert.Exactly(t, d, actual)
	}
}
