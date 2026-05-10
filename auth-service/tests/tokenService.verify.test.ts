import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

vi.mock('../src/config/redis.js', () => ({
  redisClient: {
    get: vi.fn(),
    setEx: vi.fn(),
    del: vi.fn(),
  },
}))

vi.mock('../src/services/permissionService.js', () => ({
  PermissionService: {
    getPermissionsByRoleId: vi.fn(),
  },
}))

vi.mock('jsonwebtoken', () => ({
  default: {
    verify: vi.fn(),
    sign: vi.fn(),
  },
}))

import { redisClient } from '../src/config/redis.js'
import { PermissionService } from '../src/services/permissionService.js'
import jwt from 'jsonwebtoken'
import { TokenService } from '../src/services/tokenService.js'

afterEach(() => {
  vi.clearAllMocks()
})

const sampleToken = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.sample'

const sampleUserInfo = {
  id: 'user-uuid-001',
  email: 'customer@gmail.com',
  role: 'customer',
  roleId: 'role-customer-uuid',
  permissions: ['MOVIE_VIEW', 'BOOKING_CREATE'],
  cachedAt: '2026-05-08T00:00:00.000Z',
}

// ═════════════════════════════════════════════════════════════════════════════
// verifyToken()
// ═════════════════════════════════════════════════════════════════════════════
describe('TokenService - verifyToken()', () => {
  beforeEach(() => {
    vi.mocked(redisClient.setEx).mockResolvedValue('OK' as any)
    vi.mocked(PermissionService.getPermissionsByRoleId).mockResolvedValue([])
  })

  // AUTH-UT-034
  it('AUTH-UT-034 - Token hợp lệ (đúng chữ ký, chưa hết hạn) → status 200, trả về userId/role/permissions', async () => {
    vi.mocked(jwt.verify).mockReturnValue({
      userId: 'user-uuid-001',
      email: 'customer@gmail.com',
      role: 'customer',
      roleId: 'role-customer-uuid',
      permissions: ['MOVIE_VIEW', 'BOOKING_CREATE'],
    } as any)

    const result = await TokenService.verifyToken(sampleToken)

    expect(result.status).toBe(200)
    expect(result.id).toBe('user-uuid-001')
    expect(result.role).toBe('customer')
    expect(result.permissions).toContain('MOVIE_VIEW')
  })

  // AUTH-UT-035 — Business requirement: expired token → 401 with session-expiry message
  it('AUTH-UT-035 - Token đã hết hạn (TokenExpiredError) → status 401, message "Invalid or expired token"', async () => {
    console.log('[AUTH-UT-035] TokenExpiredError → kiểm tra status 401 và message thống nhất')
    vi.mocked(jwt.verify).mockImplementation(() => {
      const err: any = new Error('jwt expired')
      err.name = 'TokenExpiredError'
      throw err
    })

    const result = await TokenService.verifyToken(sampleToken)

    expect(result.status).toBe(401)
    // Cả TokenExpiredError và JsonWebTokenError đều trả về cùng message — không phân biệt ở response
    expect(result.message).toBe('Invalid or expired token')
    expect(result.id).toBe('')
    expect(result.permissions).toHaveLength(0)
  })

  // AUTH-UT-036
  it('AUTH-UT-036 - Token bị giả mạo (JsonWebTokenError) → status 401, message "Invalid or expired token"', async () => {
    console.log('[AUTH-UT-036] JsonWebTokenError → kiểm tra status 401 và message thống nhất')
    vi.mocked(jwt.verify).mockImplementation(() => {
      const err: any = new Error('invalid signature')
      err.name = 'JsonWebTokenError'
      throw err
    })

    const result = await TokenService.verifyToken(sampleToken)

    expect(result.status).toBe(401)
    // Cùng message với AUTH-UT-035 — source code không phân biệt loại lỗi JWT
    expect(result.message).toBe('Invalid or expired token')
    expect(result.id).toBe('')
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// verifyTokenFromCache()
// ═════════════════════════════════════════════════════════════════════════════
describe('TokenService - verifyTokenFromCache()', () => {
  // AUTH-UT-037
  it('AUTH-UT-037 - Token có phiên trong cache (hot path) → xác thực thành công không cần decode JWT', async () => {
    vi.mocked(redisClient.get).mockResolvedValue(JSON.stringify(sampleUserInfo))

    const result = await TokenService.verifyTokenFromCache(sampleToken)

    expect(result.status).toBe(200)
    expect(result.id).toBe('user-uuid-001')
    expect(result.role).toBe('customer')
    expect(jwt.verify).not.toHaveBeenCalled()
  })

  // AUTH-UT-038
  it('AUTH-UT-038 - Token chưa có phiên trong cache → fallback decode JWT, tạo phiên mới trong cache', async () => {
    vi.mocked(redisClient.get).mockResolvedValue(null)
    vi.mocked(redisClient.setEx).mockResolvedValue('OK' as any)
    vi.mocked(PermissionService.getPermissionsByRoleId).mockResolvedValue([])
    vi.mocked(jwt.verify).mockReturnValue({
      userId: 'user-uuid-001',
      email: 'customer@gmail.com',
      role: 'customer',
      roleId: 'role-customer-uuid',
      permissions: ['MOVIE_VIEW'],
    } as any)

    const result = await TokenService.verifyTokenFromCache(sampleToken)

    expect(result.status).toBe(200)
    expect(result.id).toBe('user-uuid-001')
    expect(jwt.verify).toHaveBeenCalled()
  })

  it('AUTH-UT-039 - ✗ Token rỗng → status 400 (không cần kiểm tra cache)', async () => {
    const result = await TokenService.verifyTokenFromCache('')

    expect(result.status).toBe(400)
  })
})
