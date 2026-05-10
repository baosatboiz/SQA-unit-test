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
  PermissionService: { getPermissionsByRoleId: vi.fn() },
}))
vi.mock('../src/services/tokenService.js', () => ({
  TokenService: { cacheUserInfo: vi.fn() },
}))
vi.mock('bcryptjs', () => ({ default: { hash: vi.fn(), compare: vi.fn() } }))
vi.mock('jsonwebtoken', () => ({ default: { sign: vi.fn(), verify: vi.fn() } }))

import { userClient } from '../src/services/userGrpcClient.js'
import { redisClient, redisPubSubClient } from '../src/config/redis.js'
import bcrypt from 'bcryptjs'
import AuthController from '../src/controllers/authController.js'
import { ErrorMessages } from '../src/types/index.js'

function makeReqRes(body: Record<string, any>) {
  const req = { body } as any
  const res = {
    status: vi.fn().mockReturnThis(),
    json: vi.fn().mockReturnThis(),
  }
  const next = vi.fn()
  return { req, res, next }
}

const VALID_REGISTER_BODY = {
  email: 'newuser@gmail.com',
  password: 'Cinema@123',
  confirmPassword: 'Cinema@123',
  firstName: 'Nguyen',
  lastName: 'Van A',
  address: '123 Main St',
}

// ═════════════════════════════════════════════════════════════════════════════
// AuthController.register() - Luồng đăng ký tài khoản customer
// AUTH-UT-045 đến AUTH-UT-048
// ═════════════════════════════════════════════════════════════════════════════

describe('AuthController.register() - Đăng ký tài khoản', () => {
  beforeEach(() => {
    vi.mocked(redisClient.get).mockResolvedValue(null)
    vi.mocked(redisClient.setEx).mockResolvedValue('OK' as any)
    vi.mocked(redisPubSubClient.publish).mockResolvedValue(1 as any)
    vi.mocked(bcrypt.hash).mockResolvedValue('hashed_password' as any)
  })

  // AUTH-UT-045
  it('AUTH-UT-045 - ✗ Email đã tồn tại trong hệ thống → HTTP 400, EMAIL_EXISTS', async () => {
    console.log('[AUTH-UT-045] Email đã tồn tại → kiểm tra response HTTP 400, EMAIL_EXISTS')
    vi.mocked(userClient.GetUserByEmail).mockImplementation((_: any, cb: any) =>
      cb(null, { found: true, user: { id: 'existing-uuid', email: 'newuser@gmail.com' } })
    )
    const { req, res, next } = makeReqRes(VALID_REGISTER_BODY)

    await AuthController.register(req, res, next)

    expect(res.status).toHaveBeenCalledWith(400)
    expect(res.json).toHaveBeenCalledWith(
      expect.objectContaining({ message: ErrorMessages.EMAIL_EXISTS })
    )
    // EnsurePending không được gọi khi email đã tồn tại
    expect(userClient.EnsurePending).not.toHaveBeenCalled()
  })

  // AUTH-UT-046
  it('AUTH-UT-046 - ✗ confirmPassword không khớp password → HTTP 400 (Joi validation reject)', async () => {
    console.log('[AUTH-UT-046] confirmPassword không khớp → kiểm tra Joi validation trả HTTP 400')
    const { req, res, next } = makeReqRes({
      ...VALID_REGISTER_BODY,
      confirmPassword: 'DifferentPassword',
    })

    await AuthController.register(req, res, next)

    expect(res.status).toHaveBeenCalledWith(400)
    // Không gọi GetUserByEmail vì Joi đã reject trước
    expect(userClient.GetUserByEmail).not.toHaveBeenCalled()
  })

  // AUTH-UT-047
  it('AUTH-UT-047 - ✓ Đăng ký thành công → HTTP 201, REGISTRATION_SUCCESS', async () => {
    console.log('[AUTH-UT-047] Đăng ký thành công → kiểm tra HTTP 201 và message')
    vi.mocked(userClient.GetUserByEmail).mockImplementation((_: any, cb: any) =>
      cb(null, { found: false })
    )
    vi.mocked(userClient.GetRoleByName).mockImplementation((_: any, cb: any) =>
      cb(null, { found: true, id: 'role-customer-uuid' })
    )
    vi.mocked(userClient.EnsurePending).mockImplementation((_: any, cb: any) =>
      cb(null, { id: 'new-user-uuid' })
    )
    const { req, res, next } = makeReqRes(VALID_REGISTER_BODY)

    await AuthController.register(req, res, next)

    expect(res.status).toHaveBeenCalledWith(201)
    expect(res.json).toHaveBeenCalledWith(
      expect.objectContaining({ message: ErrorMessages.REGISTRATION_SUCCESS })
    )
    expect(next).not.toHaveBeenCalled()
  })

  // AUTH-UT-048
  it('AUTH-UT-048 - ✓ Đăng ký thành công → OTP được lưu Redis với key=otp:{email}, TTL=300 giây', async () => {
    console.log('[AUTH-UT-048] Đăng ký thành công → kiểm tra OTP được lưu Redis với TTL=300s')
    vi.mocked(userClient.GetUserByEmail).mockImplementation((_: any, cb: any) =>
      cb(null, { found: false })
    )
    vi.mocked(userClient.GetRoleByName).mockImplementation((_: any, cb: any) =>
      cb(null, { found: true, id: 'role-customer-uuid' })
    )
    vi.mocked(userClient.EnsurePending).mockImplementation((_: any, cb: any) =>
      cb(null, { id: 'new-user-uuid' })
    )
    const { req, res } = makeReqRes(VALID_REGISTER_BODY)

    await AuthController.register(req, res, vi.fn())

    // Tìm lần setEx với key có prefix "otp:"
    const allSetExCalls = vi.mocked(redisClient.setEx).mock.calls
    const otpCall = allSetExCalls.find(([key]) => key === `otp:${VALID_REGISTER_BODY.email}`)

    expect(otpCall).toBeDefined()
    expect(otpCall![1]).toBe(300) // TTL phải đúng 300 giây

    // Nội dung OTP phải có đầy đủ trường: otp, count=0, created_at
    const otpData = JSON.parse(otpCall![2])
    expect(otpData).toMatchObject({ count: 0 })
    expect(otpData.otp).toMatch(/^\d{6}$/) // 6 chữ số
    expect(otpData.created_at).toBeDefined()
  })
})
