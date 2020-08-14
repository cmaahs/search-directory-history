package cmd

import (
	"testing"
)
/*
To run unit tests run `./go test ./cmd` from the working directory
*/

func TestAddLeadingSlash(t *testing.T) {
	tables := []struct {
		input string
		expected string
	}{
		{"my/normal/path", "/my/normal/path"},
		{"/already/has/a/leading/slash", "/already/has/a/leading/slash"},
		{"/", "/"},
	}
	for _, table := range tables {
		output := addLeadingSlash(table.input)
		if output != table.expected {
			t.Errorf("addLeadingSlash(%s) failed. Expected: %s, Actual: %s", table.input, output, table.expected)
		}
	}
}

func TestLeadingInt(t *testing.T) {
	tables := []struct {
		input string
		expectedX int
		expectedRem string
		expectedErr string
	}{
		{"123456789test", 123456789, "test", ""},
		{"123hi45", 123, "hi45", ""},
		{"999999999999999999999999999999999999999999999999999999999999", 0, "", "time: bad [0-9]*"},
	}
	for _, table := range tables {
		x, rem, err := leadingInt(table.input)

		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}

		if !(x == table.expectedX && rem == table.expectedRem && errMsg == table.expectedErr) {
			t.Errorf("leadingInt(%s) failed.\n\tExpected: %d, %s, %s\n\t  Actual: %d, %s, %s", table.input,
				table.expectedX, table.expectedRem, table.expectedErr, x, rem, errMsg)
		}
	}
}
