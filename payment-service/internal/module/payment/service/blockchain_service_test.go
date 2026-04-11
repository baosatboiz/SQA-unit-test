package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test Case ID: PAY-TC-048
func TestIsValidTxHash_ValidFormat(t *testing.T) {
	assert.True(t, isValidTxHash("0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
}

// Test Case ID: PAY-TC-049
func TestIsValidTxHash_InvalidLength(t *testing.T) {
	assert.False(t, isValidTxHash("0x1234"))
}

// Test Case ID: PAY-TC-050
func TestIsValidTxHash_InvalidCharacters(t *testing.T) {
	assert.False(t, isValidTxHash("0x0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdeg"))
}

// Test Case ID: PAY-TC-051
func TestIsValidEthAddress_ValidFormat(t *testing.T) {
	assert.True(t, isValidEthAddress("0x0123456789abcdef0123456789abcdef01234567"))
}

// Test Case ID: PAY-TC-052
func TestIsValidEthAddress_InvalidPrefixOrLength(t *testing.T) {
	assert.False(t, isValidEthAddress("0123456789abcdef0123456789abcdef01234567"))
}

// Test Case ID: PAY-TC-053
func TestIsValidEthAddress_InvalidCharacters(t *testing.T) {
	assert.False(t, isValidEthAddress("0x0123456789abcdef0123456789abcdef0123456g"))
}