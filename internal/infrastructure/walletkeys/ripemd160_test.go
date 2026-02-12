//go:build !integration

package walletkeys

import (
	"encoding/hex"
	"testing"
)

func TestRIPEMD160SumKnownVectors(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: "9c1185a5c5e9fc54612808977ee8f548b2258d31"},
		{name: "abc", input: "abc", expected: "8eb208f7e05d987a9b044a8e98c6b087f15a0bfc"},
		{name: "message-digest", input: "message digest", expected: "5d0689ef49d2fae572b881b123a85ffa21595f36"},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			sum := ripemd160Sum([]byte(testCase.input))
			if got := hex.EncodeToString(sum[:]); got != testCase.expected {
				t.Fatalf("expected %s, got %s", testCase.expected, got)
			}
		})
	}
}
