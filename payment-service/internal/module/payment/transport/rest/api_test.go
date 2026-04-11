package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"payment-service/internal/module/payment/entity"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Mock PaymentBiz
// Implement interface business.PaymentBiz để giả lập business layer trong test.
// Mỗi field function cho phép customize hành vi trả về cho từng test case.
// =============================================================================

type mockPaymentBiz struct {
	CreatePaymentFunc         func(ctx context.Context, bookingId string, amount float64) (*entity.Payment, error)
	GetPaymentByBookingIdFunc func(ctx context.Context, bookingId string) (*entity.Payment, error)
	ProcessSePayWebhookFunc   func(ctx context.Context, webhook *entity.SePayWebhook) error
	VerifyCryptoPaymentFunc   func(ctx context.Context, req *entity.CryptoVerificationRequest) error
	ConfirmPaymentFunc        func(ctx context.Context, paymentId string, paymentMethod entity.PaymentMethod) error
}

func (m *mockPaymentBiz) CreatePayment(ctx context.Context, bookingId string, amount float64) (*entity.Payment, error) {
	if m.CreatePaymentFunc != nil {
		return m.CreatePaymentFunc(ctx, bookingId, amount)
	}
	return nil, nil
}

func (m *mockPaymentBiz) GetPaymentByBookingId(ctx context.Context, bookingId string) (*entity.Payment, error) {
	if m.GetPaymentByBookingIdFunc != nil {
		return m.GetPaymentByBookingIdFunc(ctx, bookingId)
	}
	return nil, nil
}

func (m *mockPaymentBiz) ProcessSePayWebhook(ctx context.Context, webhook *entity.SePayWebhook) error {
	if m.ProcessSePayWebhookFunc != nil {
		return m.ProcessSePayWebhookFunc(ctx, webhook)
	}
	return nil
}

func (m *mockPaymentBiz) VerifyCryptoPayment(ctx context.Context, req *entity.CryptoVerificationRequest) error {
	if m.VerifyCryptoPaymentFunc != nil {
		return m.VerifyCryptoPaymentFunc(ctx, req)
	}
	return nil
}

func (m *mockPaymentBiz) ConfirmPayment(ctx context.Context, paymentId string, paymentMethod entity.PaymentMethod) error {
	if m.ConfirmPaymentFunc != nil {
		return m.ConfirmPaymentFunc(ctx, paymentId, paymentMethod)
	}
	return nil
}

// =============================================================================
// Helper: Tạo gin engine với handler đã inject mock
// =============================================================================

func setupRouter(mock *mockPaymentBiz) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := &handler{paymentBiz: mock}

	r.POST("/payments", h.CreatePayment)
	r.GET("/payments/booking/:bookingId", h.GetPaymentByBookingId)
	r.POST("/payments/webhook/sepay", h.SePayWebhook)
	r.POST("/payments/verify-crypto", h.VerifyCryptoPayment)
	r.POST("/payments/:paymentId/confirm", h.ConfirmPayment)

	return r
}

// =============================================================================
// Tests cho CreatePayment
// =============================================================================

// Test Case ID: PAY-TC-061
// TestAPI_CreatePayment_Success
// Mô tả: Gửi request tạo payment hợp lệ → expect HTTP 200 + success=true + data payment.
func TestAPI_CreatePayment_Success(t *testing.T) {
	// --- Setup ---
	// Tạo mock trả về payment thành công
	mock := &mockPaymentBiz{
		CreatePaymentFunc: func(ctx context.Context, bookingId string, amount float64) (*entity.Payment, error) {
			return &entity.Payment{
				Id:        "pay-001",
				BookingId: bookingId,
				Amount:    amount,
				Status:    entity.PaymentStatusPending,
			}, nil
		},
	}
	router := setupRouter(mock)

	// Chuẩn bị request body hợp lệ
	body, _ := json.Marshal(map[string]interface{}{
		"booking_id": "booking-123",
		"amount":     150000.00,
	})
	req, _ := http.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// --- Execute ---
	// Gửi request qua router
	router.ServeHTTP(w, req)

	// --- Validation ---
	// Kiểm tra HTTP Status Code = 200
	assert.Equal(t, http.StatusOK, w.Code, "HTTP Status phải là 200 OK")

	// Parse JSON response
	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response phải là JSON hợp lệ")

	// Kiểm tra success = true và có data
	assert.Equal(t, true, response["success"], "Response success phải là true")
	assert.NotNil(t, response["data"], "Response phải có data")
}

// Test Case ID: PAY-TC-062
// TestAPI_CreatePayment_InvalidPayload
// Mô tả: Gửi request với JSON thiếu field bắt buộc → expect HTTP 400 + message lỗi.
func TestAPI_CreatePayment_InvalidPayload(t *testing.T) {
	// --- Setup ---
	// Mock không cần config vì request sẽ bị reject trước khi gọi biz
	mock := &mockPaymentBiz{}
	router := setupRouter(mock)

	// Request body thiếu field "amount" (required)
	body, _ := json.Marshal(map[string]interface{}{
		"booking_id": "booking-123",
	})
	req, _ := http.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// --- Execute ---
	router.ServeHTTP(w, req)

	// --- Validation ---
	// Kiểm tra HTTP Status Code = 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, w.Code, "HTTP Status phải là 400 Bad Request")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err, "Response phải là JSON hợp lệ")

	assert.Equal(t, false, response["success"], "Response success phải là false")
	assert.Equal(t, "Invalid request payload", response["message"], "Message phải rõ ràng")
}

// Test Case ID: PAY-TC-063
// TestAPI_CreatePayment_BizError
// Mô tả: Business layer trả lỗi khi tạo payment → expect HTTP 500 + message lỗi.
func TestAPI_CreatePayment_BizError(t *testing.T) {
	// --- Setup ---
	// Mock trả lỗi từ business layer
	mock := &mockPaymentBiz{
		CreatePaymentFunc: func(ctx context.Context, bookingId string, amount float64) (*entity.Payment, error) {
			return nil, fmt.Errorf("database connection failed")
		},
	}
	router := setupRouter(mock)

	body, _ := json.Marshal(map[string]interface{}{
		"booking_id": "booking-123",
		"amount":     150000.00,
	})
	req, _ := http.NewRequest(http.MethodPost, "/payments", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// --- Execute ---
	router.ServeHTTP(w, req)

	// --- Validation ---
	assert.Equal(t, http.StatusInternalServerError, w.Code, "HTTP Status phải là 500")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, false, response["success"], "Response success phải là false")
	assert.Equal(t, "Failed to create payment", response["message"])
}

// =============================================================================
// Tests cho GetPaymentByBookingId
// =============================================================================

// Test Case ID: PAY-TC-064
// TestAPI_GetPaymentByBookingId_Success
// Mô tả: Gửi request lấy payment theo bookingId hợp lệ → expect HTTP 200 + payment data.
func TestAPI_GetPaymentByBookingId_Success(t *testing.T) {
	// --- Setup ---
	mock := &mockPaymentBiz{
		GetPaymentByBookingIdFunc: func(ctx context.Context, bookingId string) (*entity.Payment, error) {
			return &entity.Payment{
				Id:        "pay-001",
				BookingId: bookingId,
				Amount:    250000.00,
				Status:    entity.PaymentStatusCompleted,
			}, nil
		},
	}
	router := setupRouter(mock)

	req, _ := http.NewRequest(http.MethodGet, "/payments/booking/booking-456", nil)
	w := httptest.NewRecorder()

	// --- Execute ---
	router.ServeHTTP(w, req)

	// --- Validation ---
	assert.Equal(t, http.StatusOK, w.Code, "HTTP Status phải là 200 OK")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, true, response["success"])
	assert.NotNil(t, response["data"], "Response phải có data chứa payment")
}

// Test Case ID: PAY-TC-065
// TestAPI_GetPaymentByBookingId_NotFound
// Mô tả: Business layer trả lỗi not found → expect HTTP 404 + message "Payment not found".
func TestAPI_GetPaymentByBookingId_NotFound(t *testing.T) {
	// --- Setup ---
	mock := &mockPaymentBiz{
		GetPaymentByBookingIdFunc: func(ctx context.Context, bookingId string) (*entity.Payment, error) {
			return nil, fmt.Errorf("payment not found for booking %s", bookingId)
		},
	}
	router := setupRouter(mock)

	req, _ := http.NewRequest(http.MethodGet, "/payments/booking/non-existent-booking", nil)
	w := httptest.NewRecorder()

	// --- Execute ---
	router.ServeHTTP(w, req)

	// --- Validation ---
	assert.Equal(t, http.StatusNotFound, w.Code, "HTTP Status phải là 404 Not Found")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Payment not found", response["message"])
}

// =============================================================================
// Tests cho SePayWebhook
// =============================================================================

// Test Case ID: PAY-TC-066
// TestAPI_SePayWebhook_Success
// Mô tả: Gửi webhook payload hợp lệ → expect HTTP 200 + status "success".
func TestAPI_SePayWebhook_Success(t *testing.T) {
	// --- Setup ---
	mock := &mockPaymentBiz{
		ProcessSePayWebhookFunc: func(ctx context.Context, webhook *entity.SePayWebhook) error {
			return nil
		},
	}
	router := setupRouter(mock)

	body, _ := json.Marshal(map[string]interface{}{
		"id":             12345,
		"gateway":        "MBBank",
		"transferAmount": 150000.00,
		"content":        "QHFFBEF88798BE46D9917B5D41747F0DC1",
		"transferType":   "in",
		"accountNumber":  "0123456789",
		"referenceCode":  "REF001",
		"description":    "Thanh toan",
	})
	req, _ := http.NewRequest(http.MethodPost, "/payments/webhook/sepay", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// --- Execute ---
	router.ServeHTTP(w, req)

	// --- Validation ---
	assert.Equal(t, http.StatusOK, w.Code, "HTTP Status phải là 200 OK")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "success", response["status"], "Response status phải là 'success'")
}

// Test Case ID: PAY-TC-067
// TestAPI_SePayWebhook_InvalidPayload
// Mô tả: Gửi JSON sai format (không phải JSON) → expect HTTP 400 + error message.
func TestAPI_SePayWebhook_InvalidPayload(t *testing.T) {
	// --- Setup ---
	mock := &mockPaymentBiz{}
	router := setupRouter(mock)

	// Body không phải JSON hợp lệ
	req, _ := http.NewRequest(http.MethodPost, "/payments/webhook/sepay", bytes.NewBufferString("invalid-json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// --- Execute ---
	router.ServeHTTP(w, req)

	// --- Validation ---
	assert.Equal(t, http.StatusBadRequest, w.Code, "HTTP Status phải là 400 Bad Request")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "Invalid webhook payload", response["error"])
}

// Test Case ID: PAY-TC-068
// TestAPI_SePayWebhook_MissingRequiredFields
// Mô tả: Gửi webhook thiếu các required fields (Id=0, Gateway="", TransferAmount=0) → expect HTTP 400.
func TestAPI_SePayWebhook_MissingRequiredFields(t *testing.T) {
	// --- Setup ---
	mock := &mockPaymentBiz{}
	router := setupRouter(mock)

	// JSON hợp lệ nhưng thiếu required fields (id=0, gateway="", transferAmount=0)
	body, _ := json.Marshal(map[string]interface{}{
		"id":             0,
		"gateway":        "",
		"transferAmount": 0,
		"content":        "some content",
	})
	req, _ := http.NewRequest(http.MethodPost, "/payments/webhook/sepay", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// --- Execute ---
	router.ServeHTTP(w, req)

	// --- Validation ---
	assert.Equal(t, http.StatusBadRequest, w.Code, "HTTP Status phải là 400 khi thiếu required fields")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "Missing required fields", response["error"])
}

// Test Case ID: PAY-TC-069
// TestAPI_SePayWebhook_BizError
// Mô tả: Business layer trả lỗi khi xử lý webhook → expect HTTP 500 + error message.
func TestAPI_SePayWebhook_BizError(t *testing.T) {
	// --- Setup ---
	mock := &mockPaymentBiz{
		ProcessSePayWebhookFunc: func(ctx context.Context, webhook *entity.SePayWebhook) error {
			return fmt.Errorf("payment not found with UUID")
		},
	}
	router := setupRouter(mock)

	body, _ := json.Marshal(map[string]interface{}{
		"id":             12345,
		"gateway":        "MBBank",
		"transferAmount": 150000.00,
		"content":        "QHFFBEF88798BE46D9917B5D41747F0DC1",
	})
	req, _ := http.NewRequest(http.MethodPost, "/payments/webhook/sepay", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// --- Execute ---
	router.ServeHTTP(w, req)

	// --- Validation ---
	assert.Equal(t, http.StatusInternalServerError, w.Code, "HTTP Status phải là 500")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "Failed to process webhook", response["error"])
}

// =============================================================================
// Tests cho VerifyCryptoPayment
// =============================================================================

// Test Case ID: PAY-TC-070
// TestAPI_VerifyCryptoPayment_Success
// Mô tả: Gửi request verify crypto payment hợp lệ → expect HTTP 200 + success message.
func TestAPI_VerifyCryptoPayment_Success(t *testing.T) {
	// --- Setup ---
	mock := &mockPaymentBiz{
		VerifyCryptoPaymentFunc: func(ctx context.Context, req *entity.CryptoVerificationRequest) error {
			return nil
		},
	}
	router := setupRouter(mock)

	body, _ := json.Marshal(map[string]interface{}{
		"booking_id":  "booking-789",
		"tx_hash":     "0xabc123def456789",
		"from_address": "0xSender123",
		"to_address":   "0xReceiver456",
		"amount_eth":   "0.05",
		"amount_vnd":   1500000.00,
		"network":      "sepolia",
	})
	req, _ := http.NewRequest(http.MethodPost, "/payments/verify-crypto", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// --- Execute ---
	router.ServeHTTP(w, req)

	// --- Validation ---
	assert.Equal(t, http.StatusOK, w.Code, "HTTP Status phải là 200 OK")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, true, response["success"])
	assert.Equal(t, "Crypto payment verified successfully", response["message"])
}

// Test Case ID: PAY-TC-071
// TestAPI_VerifyCryptoPayment_InvalidPayload
// Mô tả: Gửi request thiếu required fields → expect HTTP 400.
func TestAPI_VerifyCryptoPayment_InvalidPayload(t *testing.T) {
	// --- Setup ---
	mock := &mockPaymentBiz{}
	router := setupRouter(mock)

	// Thiếu tx_hash, from_address, to_address, amount_eth, network (all required)
	body, _ := json.Marshal(map[string]interface{}{
		"booking_id": "booking-789",
	})
	req, _ := http.NewRequest(http.MethodPost, "/payments/verify-crypto", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// --- Execute ---
	router.ServeHTTP(w, req)

	// --- Validation ---
	assert.Equal(t, http.StatusBadRequest, w.Code, "HTTP Status phải là 400")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Invalid request payload", response["message"])
}

// Test Case ID: PAY-TC-072
// TestAPI_VerifyCryptoPayment_BizError
// Mô tả: Business layer verify thất bại → expect HTTP 500 + error details.
func TestAPI_VerifyCryptoPayment_BizError(t *testing.T) {
	// --- Setup ---
	mock := &mockPaymentBiz{
		VerifyCryptoPaymentFunc: func(ctx context.Context, req *entity.CryptoVerificationRequest) error {
			return fmt.Errorf("blockchain verification failed: transaction not confirmed")
		},
	}
	router := setupRouter(mock)

	body, _ := json.Marshal(map[string]interface{}{
		"booking_id":  "booking-789",
		"tx_hash":     "0xabc123",
		"from_address": "0xSender",
		"to_address":   "0xReceiver",
		"amount_eth":   "0.05",
		"amount_vnd":   1500000.00,
		"network":      "sepolia",
	})
	req, _ := http.NewRequest(http.MethodPost, "/payments/verify-crypto", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// --- Execute ---
	router.ServeHTTP(w, req)

	// --- Validation ---
	assert.Equal(t, http.StatusInternalServerError, w.Code, "HTTP Status phải là 500")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Failed to verify crypto payment", response["message"])
	assert.NotEmpty(t, response["error"], "Response phải có error detail")
}

// =============================================================================
// Tests cho ConfirmPayment
// =============================================================================

// Test Case ID: PAY-TC-073
// TestAPI_ConfirmPayment_Success
// Mô tả: Gửi confirm payment với paymentId và payment_method hợp lệ → expect HTTP 200.
func TestAPI_ConfirmPayment_Success(t *testing.T) {
	// --- Setup ---
	mock := &mockPaymentBiz{
		ConfirmPaymentFunc: func(ctx context.Context, paymentId string, paymentMethod entity.PaymentMethod) error {
			return nil
		},
	}
	router := setupRouter(mock)

	body, _ := json.Marshal(map[string]interface{}{
		"payment_method": "CASH",
	})
	req, _ := http.NewRequest(http.MethodPost, "/payments/pay-001/confirm", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// --- Execute ---
	router.ServeHTTP(w, req)

	// --- Validation ---
	assert.Equal(t, http.StatusOK, w.Code, "HTTP Status phải là 200 OK")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, true, response["success"])
	assert.Equal(t, "Payment confirmed successfully", response["message"])
}

// Test Case ID: PAY-TC-074
// TestAPI_ConfirmPayment_MissingPaymentId
// Mô tả: Gửi request không có paymentId trong URL param → expect HTTP 400.
// Lưu ý: Trong Gin, nếu route là /:paymentId/confirm và gửi tới route không match,
// sẽ trả 404. Ta test bằng cách đặt route riêng để paymentId thành rỗng.
func TestAPI_ConfirmPayment_MissingPaymentId(t *testing.T) {
	// --- Setup ---
	// Tạo router đặc biệt với route cho phép paymentId rỗng
	gin.SetMode(gin.TestMode)
	mock := &mockPaymentBiz{}
	r := gin.New()
	h := &handler{paymentBiz: mock}
	// Route đặc biệt: khi paymentId rỗng trong logic (không phải param)
	// Gin sẽ match /:paymentId/confirm với bất kỳ segment nào.
	// Tuy nhiên, ta không thể gửi paymentId rỗng qua URL param.
	// Workaround: test trường hợp paymentId=" " (whitespace) - sẽ vẫn pass vì Gin nhận nó.
	// Thực tế, test này đảm bảo handler đúng khi paymentId empty.
	r.POST("/payments/confirm", func(c *gin.Context) {
		// Simulate empty paymentId bằng cách set param rỗng
		c.Params = gin.Params{{Key: "paymentId", Value: ""}}
		h.ConfirmPayment(c)
	})

	body, _ := json.Marshal(map[string]interface{}{
		"payment_method": "CASH",
	})
	req, _ := http.NewRequest(http.MethodPost, "/payments/confirm", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// --- Execute ---
	r.ServeHTTP(w, req)

	// --- Validation ---
	assert.Equal(t, http.StatusBadRequest, w.Code, "HTTP Status phải là 400 khi paymentId rỗng")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Payment ID is required", response["message"])
}

// Test Case ID: PAY-TC-075
// TestAPI_ConfirmPayment_InvalidPayload
// Mô tả: Gửi request thiếu payment_method trong body → expect HTTP 400.
func TestAPI_ConfirmPayment_InvalidPayload(t *testing.T) {
	// --- Setup ---
	mock := &mockPaymentBiz{}
	router := setupRouter(mock)

	// Body rỗng - thiếu payment_method (required)
	body, _ := json.Marshal(map[string]interface{}{})
	req, _ := http.NewRequest(http.MethodPost, "/payments/pay-001/confirm", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// --- Execute ---
	router.ServeHTTP(w, req)

	// --- Validation ---
	assert.Equal(t, http.StatusBadRequest, w.Code, "HTTP Status phải là 400")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Invalid request payload", response["message"])
}

// Test Case ID: PAY-TC-076
// TestAPI_ConfirmPayment_InvalidMethod
// Mô tả: Gửi request với payment_method không hợp lệ → expect HTTP 400.
func TestAPI_ConfirmPayment_InvalidMethod(t *testing.T) {
	// --- Setup ---
	mock := &mockPaymentBiz{}
	router := setupRouter(mock)

	// Payment method không nằm trong enum cho phép (CASH, BANK_TRANSFER, CRYPTOCURRENCY)
	body, _ := json.Marshal(map[string]interface{}{
		"payment_method": "CREDIT_CARD",
	})
	req, _ := http.NewRequest(http.MethodPost, "/payments/pay-001/confirm", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// --- Execute ---
	router.ServeHTTP(w, req)

	// --- Validation ---
	assert.Equal(t, http.StatusBadRequest, w.Code, "HTTP Status phải là 400 khi method không hợp lệ")

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, false, response["success"])
	assert.Equal(t, "Invalid payment method", response["message"])
}

