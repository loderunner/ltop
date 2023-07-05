package main

import (
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestCollectPropNames(t *testing.T) {
	type testCase struct {
		in     map[string]interface{}
		expect []string
	}

	testCases := []testCase{
		{in: map[string]interface{}{}, expect: []string{}},
		{
			in:     map[string]interface{}{"hello": "world"},
			expect: []string{"hello"},
		},
		{
			in:     map[string]interface{}{"hello": "world", "toto": true},
			expect: []string{"hello", "toto"},
		},
		{
			in: map[string]interface{}{
				"hello":      "world",
				"toto":       true,
				"properties": map[string]interface{}{},
			},
			expect: []string{"hello", "toto", "properties"},
		},
		{
			in: map[string]interface{}{
				"hello": "world",
				"toto":  true,
				"properties": map[string]interface{}{
					"hello": "world",
					"dead":  0xbeef,
				},
			},
			expect: []string{
				"hello",
				"toto",
				"properties.hello",
				"properties.dead",
			},
		},
	}

	for _, c := range testCases {
		got := collectPropNames(c.in)
		if !reflect.DeepEqual(got, c.expect) {
			t.Errorf("got %v, expected %v", got, c.expect)
		}
	}
}

func TestParseTime(t *testing.T) {
	type testCase struct {
		in     interface{}
		expect time.Time
		err    bool
	}
	var millis int64 = 1688596148685
	testCases := []testCase{
		{in: millis, expect: time.UnixMilli(millis).UTC()},
		{in: uint64(millis), err: true},
		{in: "2023-07-05T22:29:08.685Z", expect: time.UnixMilli(millis).UTC()},
		{in: "2023-07-06T00:29:08.685+02:00", expect: time.UnixMilli(millis).UTC()},
		{in: strconv.Itoa(int(millis)), expect: time.UnixMilli(millis).UTC()},
		{in: "Jul 10 22:29:08", expect: time.Date(0, time.July, 10, 22, 29, 8, 0, time.UTC)},
	}
	for _, c := range testCases {
		got, err := parseTime(c.in)
		if c.err {
			if err == nil {
				t.Errorf("expected error when parsing time: %#v", c.in)
			}
			continue
		}
		if !got.Equal(c.expect) {
			t.Errorf("got %v, expected %v", got, c.expect)
		}
	}
}
