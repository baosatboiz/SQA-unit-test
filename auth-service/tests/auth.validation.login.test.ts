import { describe, it, expect, vi } from 'vitest'

// Mock tất cả dependencies của AuthController trước khi import
vi.mock('../src/config/redis.js', () => ({
  redisClient: { get: vi.fn(), setEx: vi.fn(), del: vi.fn() },
  redisPubSubClient: { publish: vi.fn() },
}))
vi.mock('../src/services/userGrpcClient.js', () => ({
  userClient: {
    GetUserByEmail: vi.fn(),
    EnsurePending: vi.fn(),
    GetRoleByName: vi.fn(),
    ActivateUser: vi.fn(),
    CreateStaff: vi.fn(),
  },
}))
vi.mock('../src/services/permissionService.js', () => ({
  PermissionService: { getPermissionsByRoleId: vi.fn() },
}))
vi.mock('../src/services/tokenService.js', () => ({
  TokenService: { cacheUserInfo: vi.fn() },
}))
vi.mock('bcryptjs', () => ({ default: { hash: vi.fn(), compare: vi.fn() } }))
vi.mock('jsonwebtoken', () => ({ default: { sign: vi.fn(), verify: vi.fn() } }))

import AuthController from '../src/controllers/authController.js'

// ═════════════════════════════════════════════════════════════════════════════
// AuthController.loginSchema - Validation đăng nhập
// AUTH-UT-001 - AUTH-UT-004
// Sử dụng schema thật từ AuthController (không redefine)
// ═════════════════════════════════════════════════════════════════════════════

describe('AuthController.loginSchema - Đăng nhập', () => {
  // AUTH-UT-001: Email và password hợp lệ
  it('AUTH-UT-001 - ✓ Email và password hợp lệ → pass validation', async () => {
    const result = await AuthController.loginSchema.validateAsync({
      email: 'customer@gmail.com',
      password: 'Cinema@123',
    })

    expect(result.email).toBe('customer@gmail.com')
  })

  it('AUTH-UT-002 - ✗ Email sai định dạng → reject', async () => {
    await expect(
      AuthController.loginSchema.validateAsync({
        email: 'invalid-email',
        password: 'Cinema@123',
      }),
    ).rejects.toThrow()
  })

  it('AUTH-UT-003 - ✗ Thiếu password → reject', async () => {
    await expect(
      AuthController.loginSchema.validateAsync({
        email: 'user@gmail.com',
      }),
    ).rejects.toThrow()
  })

  it('AUTH-UT-004 - ✗ Thiếu email → reject', async () => {
    await expect(
      AuthController.loginSchema.validateAsync({
        password: 'Cinema@123',
      }),
    ).rejects.toThrow()
  })
})
