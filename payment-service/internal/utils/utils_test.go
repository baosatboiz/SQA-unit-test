package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Unit Tests cho file: utils.go
// Hàm: CheckUserPermission
// =============================================================================

// Test Case ID: PAY-TC-079
// TestUtils_CheckUserPermission_ReturnTrue
// Mô tả: Kiểm tra trường hợp user CÓ permission yêu cầu trong danh sách.
func TestUtils_CheckUserPermission_ReturnTrue(t *testing.T) {
	// --- Setup ---
	// Chuẩn bị danh sách permissions chứa permission cần kiểm tra
	requiredPermission := "payment:create"
	permissions := []string{"user:read", "payment:create", "booking:read"}

	// --- Execute ---
	// Gọi hàm CheckUserPermission với permission yêu cầu và danh sách permissions
	result := CheckUserPermission(requiredPermission, permissions)

	// --- Validation ---
	// Kết quả phải là true vì "payment:create" tồn tại trong danh sách
	assert.True(t, result, "Expected true khi permission '%s' tồn tại trong danh sách", requiredPermission)
}

// Test Case ID: PAY-TC-080
// TestUtils_CheckUserPermission_ReturnFalse
// Mô tả: Kiểm tra trường hợp user KHÔNG CÓ permission yêu cầu trong danh sách.
func TestUtils_CheckUserPermission_ReturnFalse(t *testing.T) {
	// --- Setup ---
	// Chuẩn bị danh sách permissions KHÔNG chứa permission cần kiểm tra
	requiredPermission := "payment:delete"
	permissions := []string{"user:read", "payment:create", "booking:read"}

	// --- Execute ---
	// Gọi hàm CheckUserPermission với permission không tồn tại
	result := CheckUserPermission(requiredPermission, permissions)

	// --- Validation ---
	// Kết quả phải là false vì "payment:delete" không tồn tại trong danh sách
	assert.False(t, result, "Expected false khi permission '%s' không tồn tại trong danh sách", requiredPermission)
}

// Test Case ID: PAY-TC-081
// TestUtils_CheckUserPermission_EmptyPermissions
// Mô tả: Kiểm tra trường hợp danh sách permissions rỗng (user không có permission nào).
func TestUtils_CheckUserPermission_EmptyPermissions(t *testing.T) {
	// --- Setup ---
	// Danh sách permissions rỗng
	requiredPermission := "payment:create"
	permissions := []string{}

	// --- Execute ---
	// Gọi hàm CheckUserPermission với danh sách rỗng
	result := CheckUserPermission(requiredPermission, permissions)

	// --- Validation ---
	// Kết quả phải là false vì danh sách rỗng, không thể tìm thấy permission nào
	assert.False(t, result, "Expected false khi danh sách permissions rỗng")
}

