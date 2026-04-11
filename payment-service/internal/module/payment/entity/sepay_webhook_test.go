package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Unit Tests cho file: sepay_webhook.go
// Hàm: ToPayload()
// =============================================================================

// Test Case ID: PAY-TC-077
// TestSePayWebhook_ToPayload_Success
// Mô tả: Kiểm tra serialize đầy đủ các fields của SePayWebhook sang JSON string thành công.
func TestSePayWebhook_ToPayload_Success(t *testing.T) {
	// --- Setup ---
	// Khởi tạo SePayWebhook với đầy đủ dữ liệu, bao gồm cả optional fields (Code, SubAccount)
	code := "TXN001"
	subAccount := "SUB_ACC_01"
	webhook := &SePayWebhook{
		Id:              12345,
		Gateway:         "MBBank",
		TransactionDate: "2026-04-11 08:00:00",
		AccountNumber:   "0123456789",
		Code:            &code,
		Content:         "QHFFBEF88798BE46D9917B5D41747F0DC1",
		TransferType:    "in",
		TransferAmount:  150000.00,
		Accumulated:     500000.00,
		SubAccount:      &subAccount,
		ReferenceCode:   "REF_001",
		Description:     "Thanh toan ve xem phim",
	}

	// --- Execute ---
	// Gọi ToPayload() để serialize struct sang JSON string
	payload, err := webhook.ToPayload()

	// --- Validation ---
	// Không được có lỗi khi serialize
	assert.NoError(t, err, "ToPayload() không được trả lỗi khi struct hợp lệ")
	// Payload không được rỗng
	assert.NotEmpty(t, payload, "Payload JSON string không được rỗng")
	// Kiểm tra payload chứa các field quan trọng
	assert.Contains(t, payload, `"id":12345`, "Payload phải chứa field id")
	assert.Contains(t, payload, `"gateway":"MBBank"`, "Payload phải chứa field gateway")
	assert.Contains(t, payload, `"transferAmount":150000`, "Payload phải chứa field transferAmount")
	assert.Contains(t, payload, `"content":"QHFFBEF88798BE46D9917B5D41747F0DC1"`, "Payload phải chứa field content")
}

// Test Case ID: PAY-TC-078
// TestSePayWebhook_ToPayload_WithNilOptionalFields
// Mô tả: Kiểm tra serialize khi các optional fields (Code, SubAccount) là nil.
func TestSePayWebhook_ToPayload_WithNilOptionalFields(t *testing.T) {
	// --- Setup ---
	// Khởi tạo SePayWebhook với Code và SubAccount là nil
	webhook := &SePayWebhook{
		Id:              99999,
		Gateway:         "VietcomBank",
		TransactionDate: "2026-04-11 09:30:00",
		AccountNumber:   "9876543210",
		Code:            nil, // optional field - nil
		Content:         "Test payment content",
		TransferType:    "in",
		TransferAmount:  200000.00,
		Accumulated:     1000000.00,
		SubAccount:      nil, // optional field - nil
		ReferenceCode:   "REF_002",
		Description:     "Payment description",
	}

	// --- Execute ---
	// Gọi ToPayload() để serialize struct
	payload, err := webhook.ToPayload()

	// --- Validation ---
	// Serialize phải thành công dù có nil fields
	assert.NoError(t, err, "ToPayload() không được trả lỗi khi optional fields là nil")
	assert.NotEmpty(t, payload, "Payload không được rỗng")
	// Kiểm tra nil fields được serialize thành null trong JSON
	assert.Contains(t, payload, `"code":null`, "Field code phải là null khi giá trị nil")
	assert.Contains(t, payload, `"subAccount":null`, "Field subAccount phải là null khi giá trị nil")
}

