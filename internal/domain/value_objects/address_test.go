//go:build !integration

package valueobjects

import "testing"

func TestToEIP55ChecksumKnownFixtures(t *testing.T) {
	testCases := []struct {
		canonical string
		expected  string
	}{
		{
			canonical: "0x5aaeb6053f3e94c9b9a09f33669435e7ef1beaed",
			expected:  "0x5aAeb6053F3E94C9b9A09f33669435E7Ef1BeAed",
		},
		{
			canonical: "0xfb6916095ca1df60bb79ce92ce3ea74c37c5d359",
			expected:  "0xfB6916095ca1df60bB79Ce92cE3Ea74c37c5d359",
		},
		{
			canonical: "0xdbf03b407c01e7cd3cbea99509d93f8dddc8c6fb",
			expected:  "0xdbF03B407c01E7cD3CBea99509d93f8DDDC8C6FB",
		},
	}

	for _, testCase := range testCases {
		actual, appErr := ToEIP55Checksum(testCase.canonical)
		if appErr != nil {
			t.Fatalf("expected no error for %s, got %+v", testCase.canonical, appErr)
		}
		if actual != testCase.expected {
			t.Fatalf("expected %s, got %s", testCase.expected, actual)
		}
	}
}

func TestNormalizeAddressForStorageBitcoinBech32Lowercase(t *testing.T) {
	canonical, appErr := NormalizeAddressForStorage("bitcoin", "BC1QW508D6QEJXTDG4Y5R3ZARVARY0C5XW7K7GT080")
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if canonical != "bc1qw508d6qejxtdg4y5r3zarvary0c5xw7k7gt080" {
		t.Fatalf("expected lowercase bech32 canonical, got %s", canonical)
	}
}

func TestNormalizeAddressForStorageBitcoinAcceptsTestnetBech32(t *testing.T) {
	canonical, appErr := NormalizeAddressForStorage("bitcoin", "tb1qfm5r0m9fxxv3x47clz9k2zk5f4d4f0htptf0kr")
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if canonical != "tb1qfm5r0m9fxxv3x47clz9k2zk5f4d4f0htptf0kr" {
		t.Fatalf("unexpected canonical address: %s", canonical)
	}
}

func TestNormalizeAddressForStorageBitcoinAcceptsRegtestBech32(t *testing.T) {
	canonical, appErr := NormalizeAddressForStorage("bitcoin", "bcrt1qfm5r0m9fxxv3x47clz9k2zk5f4d4f0htptf0kr")
	if appErr != nil {
		t.Fatalf("expected no error, got %+v", appErr)
	}
	if canonical != "bcrt1qfm5r0m9fxxv3x47clz9k2zk5f4d4f0htptf0kr" {
		t.Fatalf("unexpected canonical address: %s", canonical)
	}
}
