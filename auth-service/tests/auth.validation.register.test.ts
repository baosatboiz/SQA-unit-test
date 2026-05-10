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
// AuthController.registerSchema - Validation đăng ký
// AUTH-UT-012 - AUTH-UT-017
// Sử dụng schema thật từ AuthController (không redefine)
// ═════════════════════════════════════════════════════════════════════════════

describe('AuthController.registerSchema - Đăng ký tài khoản', () => {
  // AUTH-UT-012: Đầy đủ thông tin hợp lệ
  it('AUTH-UT-012 - ✓ Đầy đủ thông tin hợp lệ → pass validation', async () => {
    const result = await AuthController.registerSchema.validateAsync({
      email: 'newuser@gmail.com',
      password: 'Cinema@123',
      confirmPassword: 'Cinema@123',
      firstName: 'Nguyen',
      lastName: 'Van A',
      address: '123 Main St',
    })

    expect(result.email).toBe('newuser@gmail.com')
  })

  // AUTH-UT-013: Email sai định dạng
  it('AUTH-UT-013 - ✗ Email sai định dạng → reject (400)', async () => {
    await expect(
      AuthController.registerSchema.validateAsync({
        email: 'notAnEmail',
        password: 'Cinema@123',
        confirmPassword: 'Cinema@123',
        firstName: 'A',
        lastName: 'B',
      }),
    ).rejects.toThrow()
  })

  // AUTH-UT-014: Password < 6 ký tự
  it('AUTH-UT-014 - ✗ Password < 6 ký tự → reject (400)', async () => {
    await expect(
      AuthController.registerSchema.validateAsync({
        email: 'user@gmail.com',
        password: '12345',
        confirmPassword: '12345',
        firstName: 'A',
        lastName: 'B',
      }),
    ).rejects.toThrow()
  })

  // AUTH-UT-015: confirmPassword không khớp
  it('AUTH-UT-015 - ✗ confirmPassword không khớp password → reject (400)', async () => {
    await expect(
      AuthController.registerSchema.validateAsync({
        email: 'user@gmail.com',
        password: 'Cinema@123',
        confirmPassword: 'Cinema@456',
        firstName: 'A',
        lastName: 'B',
      }),
    ).rejects.toThrow()
  })

  it('AUTH-UT-016 - ✓ Address để trống (optional) → pass', async () => {
    const result = await AuthController.registerSchema.validateAsync({
      email: 'user@gmail.com',
      password: 'Cinema@123',
      confirmPassword: 'Cinema@123',
      firstName: 'Nguyen',
      lastName: 'Van A',
      address: '',
    })

    expect(result.address).toBe('')
  })

  it('AUTH-UT-017 - ✗ Thiếu firstName → reject', async () => {
    await expect(
      AuthController.registerSchema.validateAsync({
        email: 'user@gmail.com',
        password: 'Cinema@123',
        confirmPassword: 'Cinema@123',
        lastName: 'B',
      }),
    ).rejects.toThrow()
  })
})
