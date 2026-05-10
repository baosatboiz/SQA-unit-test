import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// Mock tất cả dependencies của AuthController trước khi import
vi.mock('../src/config/redis.js', () => ({
  redisClient: {
    get: vi.fn(),
    setEx: vi.fn(),
    del: vi.fn(),
    ttl: vi.fn(),
    keys: vi.fn(),
  },
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

import { redisClient } from '../src/config/redis.js'
import AuthController from '../src/controllers/authController.js'

afterEach(() => {
  vi.clearAllMocks()
})

// ═════════════════════════════════════════════════════════════════════════════
// AuthController.verifyOtpSchema - Validation schema OTP
// AUTH-UT-005 - AUTH-UT-007
// Sử dụng schema thật từ AuthController (không redefine)
// ═════════════════════════════════════════════════════════════════════════════

describe('AuthController.verifyOtpSchema - Xác thực OTP', () => {
  beforeEach(() => { vi.clearAllMocks() })

  // AUTH-UT-005: OTP hợp lệ 6 chữ số
  it('AUTH-UT-005 - ✓ Email hợp lệ và OTP đúng 6 chữ số → pass validation', async () => {
    const result = await AuthController.verifyOtpSchema.validateAsync({
      email: 'user@gmail.com',
      otp: '123456',
    })

    expect(result.otp).toBe('123456')
  })

  // AUTH-UT-006: OTP thiếu chữ số (< 6)
  it('AUTH-UT-006 - ✗ OTP < 6 chữ số → reject (400)', async () => {
    await expect(
      AuthController.verifyOtpSchema.validateAsync({ email: 'user@gmail.com', otp: '123' }),
    ).rejects.toThrow()
  })

  // AUTH-UT-007: OTP quá dài (> 6)
  it('AUTH-UT-007 - ✗ OTP > 6 chữ số → reject (400)', async () => {
    await expect(
      AuthController.verifyOtpSchema.validateAsync({ email: 'user@gmail.com', otp: '1234567' }),
    ).rejects.toThrow()
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// AuthController.verifyOTPFromCache() - Logic xác thực OTP từ Redis cache
// AUTH-UT-008 - AUTH-UT-011
// Gọi trực tiếp hàm service thật, mock redisClient
// ═════════════════════════════════════════════════════════════════════════════

describe('AuthController.verifyOTPFromCache() - Xác thực OTP từ cache', () => {
  beforeEach(() => { vi.clearAllMocks() })

  const email = 'pending@gmail.com'

  // AUTH-UT-008: OTP đúng → verified=true, xóa OTP khỏi Redis
  it('AUTH-UT-008 - ✓ OTP đúng → success=true, Redis.del được gọi để xóa OTP', async () => {
    const cachedOtpData = { otp: '123456', count: 0, created_at: new Date().toISOString() }
    vi.mocked(redisClient.get).mockResolvedValue(JSON.stringify(cachedOtpData))
    vi.mocked(redisClient.del).mockResolvedValue(1 as any)

    const result = await AuthController.verifyOTPFromCache(email, '123456')

    expect(result.success).toBe(true)
    expect(redisClient.del).toHaveBeenCalledWith(`otp:${email}`)
  })

  // AUTH-UT-009: OTP sai → tăng attempt count, không xóa OTP
  it('AUTH-UT-009 - ✗ OTP sai → success=false, attempt count tăng lên 1, Redis.setEx ghi lại count mới, Redis.del không gọi', async () => {
    console.log('[AUTH-UT-009] OTP sai → kiểm tra count tăng và setEx cập nhật đúng dữ liệu')
    const cachedOtpData = { otp: '123456', count: 0, created_at: new Date().toISOString() }
    vi.mocked(redisClient.get).mockResolvedValue(JSON.stringify(cachedOtpData))
    vi.mocked(redisClient.ttl).mockResolvedValue(250 as any)
    vi.mocked(redisClient.setEx).mockResolvedValue('OK' as any)

    const result = await AuthController.verifyOTPFromCache(email, '999999')

    expect(result.success).toBe(false)
    expect(result.attempts).toBe(1)
    expect(redisClient.del).not.toHaveBeenCalled()

    // Kiểm tra setEx được gọi với count đã tăng lên 1 (không phải count=0 cũ)
    expect(redisClient.setEx).toHaveBeenCalledWith(
      `otp:${email}`,
      250, // TTL còn lại từ redisClient.ttl
      expect.stringContaining('"count":1'),
    )
  })

  // AUTH-UT-010: OTP hết hạn → Redis trả null → OTP_EXPIRED
  it('AUTH-UT-010 - ✗ OTP hết hạn (Redis trả null) → success=false, message OTP_EXPIRED', async () => {
    vi.mocked(redisClient.get).mockResolvedValue(null)

    const result = await AuthController.verifyOTPFromCache(email, '123456')

    expect(result.success).toBe(false)
    expect(result.message).toMatch(/hết hạn/i)
  })

  // AUTH-UT-011: Đã thử >= 5 lần → OTP_MAX_ATTEMPTS
  it('AUTH-UT-011 - ✗ Đã thử OTP 5 lần → success=false, message OTP_MAX_ATTEMPTS (không kiểm tra OTP nữa)', async () => {
    const cachedOtpData = { otp: '123456', count: 5, created_at: new Date().toISOString() }
    vi.mocked(redisClient.get).mockResolvedValue(JSON.stringify(cachedOtpData))

    const result = await AuthController.verifyOTPFromCache(email, 'anything')

    expect(result.success).toBe(false)
    expect(result.message).toMatch(/quá nhiều lần/i)
    expect(result.attempts).toBe(5)
  })
})
