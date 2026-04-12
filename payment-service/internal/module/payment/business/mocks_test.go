// Package business — mocks for unit tests.
//
// This file provides testify-based mock implementations for the interfaces
// that paymentBiz depends on. Using mocks lets us:
//   1. Exercise business logic without touching real gRPC / blockchain / external systems
//   2. Control the return values of dependencies to reproduce edge cases
//   3. Assert that dependencies were called with the expected arguments (CheckDB-by-mock style)
//
// Note: PaymentRepository is mocked here for convenience, but the DB-touching
// tests (TC-001 → TC-023) use a REAL repository backed by a *bun.Tx so that
// the CheckDB and Rollback requirements are satisfied against a real database.
package business

import (
	"context"

	"github.com/stretchr/testify/mock"
	"github.com/uptrace/bun"

	"payment-service/internal/module/payment/entity"
)

// -----------------------------------------------------------------------------
// MockPaymentRepository — implements repository.PaymentRepository
// Used in pure-unit-test paths where we do not want a real DB.
// -----------------------------------------------------------------------------
type MockPaymentRepository struct {
	mock.Mock
}

func (m *MockPaymentRepository) FindByContent(ctx context.Context, content string) (*entity.Payment, error) {
	args := m.Called(ctx, content)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Payment), args.Error(1)
}

func (m *MockPaymentRepository) FindByBookingId(ctx context.Context, bookingId string) (*entity.Payment, error) {
	args := m.Called(ctx, bookingId)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Payment), args.Error(1)
}

func (m *MockPaymentRepository) FindByShortCode(ctx context.Context, shortCode string) (*entity.Payment, error) {
	args := m.Called(ctx, shortCode)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Payment), args.Error(1)
}

func (m *MockPaymentRepository) FindByUUIDNoHyphens(ctx context.Context, uuidNoHyphens string) (*entity.Payment, error) {
	args := m.Called(ctx, uuidNoHyphens)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Payment), args.Error(1)
}

func (m *MockPaymentRepository) UpdatePaymentFields(ctx context.Context, db bun.IDB, id string, fields map[string]interface{}) error {
	args := m.Called(ctx, db, id, fields)
	return args.Error(0)
}

func (m *MockPaymentRepository) Create(ctx context.Context, payment *entity.Payment) error {
	args := m.Called(ctx, payment)
	return args.Error(0)
}

func (m *MockPaymentRepository) GetById(ctx context.Context, id string) (*entity.Payment, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Payment), args.Error(1)
}

// -----------------------------------------------------------------------------
// MockOutboxClient — implements grpcRepo.OutboxClientInterface
// We mock this everywhere because CreateOutboxEvent calls a real gRPC service
// that is not available in unit tests.
// -----------------------------------------------------------------------------
type MockOutboxClient struct {
	mock.Mock
}

func (m *MockOutboxClient) CreateOutboxEvent(ctx context.Context, eventType string, eventData interface{}) error {
	args := m.Called(ctx, eventType, eventData)
	return args.Error(0)
}

// -----------------------------------------------------------------------------
// MockBlockchainService — implements service.BlockchainService
// Used by VerifyCryptoPayment tests so we can drive success/failure without
// hitting the Ethereum network.
// -----------------------------------------------------------------------------
type MockBlockchainService struct {
	mock.Mock
}

func (m *MockBlockchainService) VerifyTransaction(ctx context.Context, txHash, expectedFrom, expectedTo, expectedAmount string) error {
	args := m.Called(ctx, txHash, expectedFrom, expectedTo, expectedAmount)
	return args.Error(0)
}
