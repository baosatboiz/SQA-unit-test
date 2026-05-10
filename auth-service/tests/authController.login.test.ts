import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('../src/config/redis.js', () => ({
  redisClient: { get: vi.fn(), setEx: vi.fn(), del: vi.fn() },
  redisPubSubClient: { publish: vi.fn() },
}))
vi.mock('../src/services/userGrpcClient.js', () => ({
  userClient: {
    GetUserByEmail: vi.fn(),
    GetUserWithRoleByEmail: vi.fn(),
    EnsurePending: vi.fn(),
    GetRoleByName: vi.fn(),
    ActivateUser: vi.fn(),
    CreateStaff: vi.fn(),
  },
}))
vi.mock('../src/services/permissionService.js', () => ({
  PermissionService: {
    getPermissionsByRoleId: vi.fn(),
    clearPermissionsCache: vi.fn(),
  },
}))
vi.mock('../src/services/tokenService.js', () => ({
  TokenService: { cacheUserInfo: vi.fn() },
}))
vi.mock('bcryptjs', () => ({ default: { hash: vi.fn(), compare: vi.fn() } }))
vi.mock('jsonwebtoken', () => ({ default: { sign: vi.fn(), verify: vi.fn() } }))

import { userClient } from '../src/services/userGrpcClient.js'
import { PermissionService } from '../src/services/permissionService.js'
import { TokenService } from '../src/services/tokenService.js'
import bcrypt from 'bcryptjs'
import jwt from 'jsonwebtoken'
import AuthController from '../src/controllers/authController.js'
import { ErrorMessages, UserStatus } from '../src/types/index.js'

function makeReqRes(body: Record<string, any>) {
  const req = { body } as any
  const res = {
    status: vi.fn().mockReturnThis(),
    json: vi.fn().mockReturnThis(),
  }
  const next = vi.fn()
  return { req, res, next }
}

// ═════════════════════════════════════════════════════════════════════════════
// AuthController.login() - Luồng đăng nhập customer
// AUTH-UT-040 đến AUTH-UT-044
// ═════════════════════════════════════════════════════════════════════════════

describe('AuthController.login() - Đăng nhập customer', () => {
  beforeEach(() => {
    vi.mocked(PermissionService.getPermissionsByRoleId).mockResolvedValue([])
    vi.mocked(TokenService.cacheUserInfo).mockResolvedValue(undefined)
    vi.mocked(jwt.sign).mockReturnValue('mock-jwt-token' as any)
  })

  // AUTH-UT-040
  it('AUTH-UT-040 - ✗ Email không tồn tại trong hệ thống → HTTP 400, INVALID_CREDENTIALS', async () => {
    console.log('[AUTH-UT-040] Email không tồn tại → kiểm tra response HTTP 400')
    vi.mocked(userClient.GetUserByEmail).mockImplementation((_: any, cb: any) =>
      cb(null, { found: false })
    )
    const { req, res, next } = makeReqRes({ email: 'notfound@gmail.com', password: 'Cinema@123' })

    await AuthController.login(req, res, next)

    expect(res.status).toHaveBeenCalledWith(400)
    expect(res.json).toHaveBeenCalledWith(
      expect.objectContaining({ message: ErrorMessages.INVALID_CREDENTIALS })
    )
    expect(next).not.toHaveBeenCalled()
  })

  // AUTH-UT-041
  it('AUTH-UT-041 - ✗ Mật khẩu sai → HTTP 400, INVALID_CREDENTIALS (không tiết lộ lý do cụ thể)', async () => {
    console.log('[AUTH-UT-041] Mật khẩu không khớp → kiểm tra response HTTP 400')
    vi.mocked(userClient.GetUserByEmail).mockImplementation((_: any, cb: any) =>
      cb(null, {
        found: true,
        user: {
          id: 'u1',
          email: 'customer@gmail.com',
          password: 'hashed_password',
          status: UserStatus.ACTIVE,
          name: 'Test User',
        },
      })
    )
    vi.mocked(bcrypt.compare).mockResolvedValue(false as any)
    const { req, res, next } = makeReqRes({ email: 'customer@gmail.com', password: 'WrongPass' })

    await AuthController.login(req, res, next)

    expect(res.status).toHaveBeenCalledWith(400)
    expect(res.json).toHaveBeenCalledWith(
      expect.objectContaining({ message: ErrorMessages.INVALID_CREDENTIALS })
    )
  })

  // AUTH-UT-042
  it('AUTH-UT-042 - ✗ Tài khoản PENDING (chưa xác thực OTP) → HTTP 400, requireVerification=true, email trả về', async () => {
    console.log('[AUTH-UT-042] Tài khoản PENDING → kiểm tra requireVerification=true')
    vi.mocked(userClient.GetUserByEmail).mockImplementation((_: any, cb: any) =>
      cb(null, {
        found: true,
        user: {
          id: 'u1',
          email: 'customer@gmail.com',
          password: 'hashed_password',
          status: UserStatus.PENDING,
          name: 'Pending User',
        },
      })
    )
    vi.mocked(bcrypt.compare).mockResolvedValue(true as any)
    const { req, res, next } = makeReqRes({ email: 'customer@gmail.com', password: 'Cinema@123' })

    await AuthController.login(req, res, next)

    expect(res.status).toHaveBeenCalledWith(400)
    expect(res.json).toHaveBeenCalledWith(
      expect.objectContaining({
        requireVerification: true,
        email: 'customer@gmail.com',
      })
    )
  })

  // AUTH-UT-043
  it('AUTH-UT-043 - ✓ Đăng nhập thành công (customer, ACTIVE) → HTTP 200, token và user info (không có password)', async () => {
    console.log('[AUTH-UT-043] Đăng nhập thành công → kiểm tra token và user info trong response')
    vi.mocked(userClient.GetUserByEmail).mockImplementation((_: any, cb: any) =>
      cb(null, {
        found: true,
        user: {
          id: 'user-uuid-001',
          email: 'customer@gmail.com',
          password: 'hashed_password',
          status: UserStatus.ACTIVE,
          name: 'Nguyen Van A',
        },
      })
    )
    vi.mocked(userClient.GetUserWithRoleByEmail).mockImplementation((_: any, cb: any) =>
      cb(null, {
        found: true,
        user: { role_name: 'customer', role_id: 'role-customer-uuid' },
      })
    )
    vi.mocked(bcrypt.compare).mockResolvedValue(true as any)
    const { req, res, next } = makeReqRes({ email: 'customer@gmail.com', password: 'Cinema@123' })

    await AuthController.login(req, res, next)

    expect(res.json).toHaveBeenCalledWith(
      expect.objectContaining({
        token: 'mock-jwt-token',
        user: expect.objectContaining({
          email: 'customer@gmail.com',
          role: 'customer',
        }),
      })
    )
    const jsonArg = vi.mocked(res.json).mock.calls[0][0]
    expect(jsonArg.user).not.toHaveProperty('password')
    expect(res.status).not.toHaveBeenCalledWith(400)
    expect(res.status).not.toHaveBeenCalledWith(403)
  })

  // AUTH-UT-044
  it('AUTH-UT-044 - ✗ Tài khoản role=admin đăng nhập qua endpoint /login (customer-only) → HTTP 403 FORBIDDEN', async () => {
    console.log('[AUTH-UT-044] Admin cố đăng nhập endpoint customer-only → kiểm tra HTTP 403')
    vi.mocked(userClient.GetUserByEmail).mockImplementation((_: any, cb: any) =>
      cb(null, {
        found: true,
        user: {
          id: 'u2',
          email: 'admin@cinema.com',
          password: 'hashed_password',
          status: UserStatus.ACTIVE,
          name: 'Admin User',
        },
      })
    )
    vi.mocked(userClient.GetUserWithRoleByEmail).mockImplementation((_: any, cb: any) =>
      cb(null, {
        found: true,
        user: { role_name: 'admin', role_id: 'role-admin-uuid' },
      })
    )
    vi.mocked(bcrypt.compare).mockResolvedValue(true as any)
    const { req, res, next } = makeReqRes({ email: 'admin@cinema.com', password: 'Admin@123' })

    await AuthController.login(req, res, next)

    expect(res.status).toHaveBeenCalledWith(403)
    expect(res.json).toHaveBeenCalled()
  })
})
