// Package business — unit tests for UUID-extraction helpers.
//
// These helpers (extractUUIDNoHyphens, extractUUIDFromText, isValidUUIDNoHyphens)
// are pure, deterministic functions: no DB, no network, no mocks required.
// Each test therefore targets a single input/output contract derived from the
// Unit Testing Report (PAY-TC-024 → PAY-TC-032).

package business

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// -----------------------------------------------------------------------------
// (*paymentBiz).extractUUIDNoHyphens  —  priority: content first, description fallback
// -----------------------------------------------------------------------------

// TC-024 — extractUUIDNoHyphens prefers UUID found in `content` over the one
// in `description` when BOTH fields contain a valid 32-char hex UUID.
func TestExtractUUIDNoHyphens_PrefersContentOverDescription(t *testing.T) {
	biz := &paymentBiz{} // helper methods don't touch any field, zero-value is enough
	contentUUID := "FFBEF88798BE46D9917B5D41747F0DC1"     // 32 hex chars
	descriptionUUID := "AABBCCDDEEFF11223344556677889900" // also valid

	got := biz.extractUUIDNoHyphens(contentUUID, descriptionUUID)

	assert.Equal(t, contentUUID, got,
		"when both fields are valid, content must win (priority rule)")
}

// TC-025 — extractUUIDNoHyphens falls back to `description` when `content`
// contains no UUID. This is the documented fallback path.
func TestExtractUUIDNoHyphens_FallsBackToDescription(t *testing.T) {
	biz := &paymentBiz{}
	contentWithoutUUID := "thanh toan vnpay" // no 32-char hex substring
	descriptionUUID := "AABBCCDDEEFF11223344556677889900"

	got := biz.extractUUIDNoHyphens(contentWithoutUUID, descriptionUUID)

	assert.Equal(t, descriptionUUID, got,
		"should fall back to description when content has no UUID")
}

// TC-026 — extractUUIDNoHyphens returns the empty string when neither field
// contains a valid 32-char hex UUID. Empty string is the "not found" sentinel.
func TestExtractUUIDNoHyphens_ReturnsEmptyWhenMissing(t *testing.T) {
	biz := &paymentBiz{}
	got := biz.extractUUIDNoHyphens("chuyen khoan", "noi dung khong co uuid")
	assert.Equal(t, "", got,
		"no UUID in either field must yield empty string")
}

// -----------------------------------------------------------------------------
// extractUUIDFromText  —  strips "QH" prefix, then scans substring
// -----------------------------------------------------------------------------

// TC-027 — when the text starts with the canonical "QH" prefix followed by
// 32 hex chars, extractUUIDFromText must strip "QH" and return only the
// UUID portion.
func TestExtractUUIDFromText_StripsQHPrefix(t *testing.T) {
	input := "QHFFBEF88798BE46D9917B5D41747F0DC1"
	expected := "FFBEF88798BE46D9917B5D41747F0DC1"

	got := extractUUIDFromText(input)

	assert.Equal(t, expected, got, "QH prefix must be removed")
}

// TC-028 — the extractor must also find a 32-char hex UUID embedded anywhere
// inside the text (substring search), not only at the beginning.
func TestExtractUUIDFromText_FindsUUIDAnywhereInText(t *testing.T) {
	embeddedUUID := "FFBEF88798BE46D9917B5D41747F0DC1"
	input := "Payment " + embeddedUUID + " done"

	got := extractUUIDFromText(input)

	assert.Equal(t, embeddedUUID, got,
		"substring search must locate a valid UUID anywhere in the text")
}

// -----------------------------------------------------------------------------
// isValidUUIDNoHyphens  —  validator for a 32-char hex string
// -----------------------------------------------------------------------------

// TC-029 — a 32-character numeric string (all digits are valid hex) must be
// accepted as a valid no-hyphen UUID.
func TestIsValidUUIDNoHyphens_AcceptsThirtyTwoHexDigits(t *testing.T) {
	input := "12345678901234567890123456789012" // exactly 32 chars, all hex

	assert.True(t, isValidUUIDNoHyphens(input),
		"32 hex digits must be recognized as a valid no-hyphen UUID")
}

// TC-030 — any string whose length is not exactly 32 must be rejected
// regardless of its character content.
func TestIsValidUUIDNoHyphens_RejectsWrongLength(t *testing.T) {
	input := "123456789" // length 9, far from 32

	assert.False(t, isValidUUIDNoHyphens(input),
		"strings shorter/longer than 32 chars must be rejected")
}

// TC-031 — a 32-char string that contains a non-hex character (here 'G' which
// is past 'F') must be rejected.
func TestIsValidUUIDNoHyphens_RejectsNonHexCharacter(t *testing.T) {
	input := "1234567890123456789012345678901G" // last char 'G' is not hex

	assert.False(t, isValidUUIDNoHyphens(input),
		"non-hex character must invalidate the UUID")
}

// TC-032 — the validator must accept lowercase hex characters (a-f) just as
// it accepts uppercase. This is important because webhook payloads can arrive
// in either case.
func TestIsValidUUIDNoHyphens_AcceptsLowercaseHex(t *testing.T) {
	input := "abcdef12345678901234567890123456" // lowercase a-f + digits = valid

	assert.True(t, isValidUUIDNoHyphens(input),
		"lowercase hex chars (a-f) must be accepted")
}
