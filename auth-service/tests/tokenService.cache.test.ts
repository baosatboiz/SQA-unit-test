import { describe, it, expect, vi, afterEach } from 'vitest'

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
// cacheUserInfo()
// ═════════════════════════════════════════════════════════════════════════════
describe('TokenService - cacheUserInfo()', () => {
  // AUTH-UT-028
  it('AUTH-UT-028 - Lưu thông tin phiên sau đăng nhập → Redis setEx với TTL 3600 giây (1 giờ)', async () => {
    vi.mocked(redisClient.setEx).mockResolvedValue('OK' as any)

    await TokenService.cacheUserInfo(sampleToken, sampleUserInfo)

    expect(redisClient.setEx).toHaveBeenCalledWith(
      `auth:token:${sampleToken}`,
      3600,
      JSON.stringify(sampleUserInfo),
    )
  })

  // AUTH-UT-029
  it('AUTH-UT-029 - Redis không khả dụng → không ném lỗi, hệ thống tiếp tục hoạt động bình thường', async () => {
    vi.mocked(redisClient.setEx).mockRejectedValue(new Error('ECONNREFUSED: Redis connection refused'))

    await expect(TokenService.cacheUserInfo(sampleToken, sampleUserInfo)).resolves.toBeUndefined()
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// getCachedUserInfo()
// ═════════════════════════════════════════════════════════════════════════════
describe('TokenService - getCachedUserInfo()', () => {
  // AUTH-UT-030
  it('AUTH-UT-030 - Token có phiên đang hoạt động trong cache → trả về thông tin phiên (id, email, role, permissions)', async () => {
    vi.mocked(redisClient.get).mockResolvedValue(JSON.stringify(sampleUserInfo))

    const result = await TokenService.getCachedUserInfo(sampleToken)

    expect(result).toEqual(sampleUserInfo)
    expect(redisClient.get).toHaveBeenCalledWith(`auth:token:${sampleToken}`)
  })

  // AUTH-UT-031
  it('AUTH-UT-031 - Token không có phiên trong cache → trả về null, không gây lỗi hệ thống', async () => {
    vi.mocked(redisClient.get).mockResolvedValue(null)

    const result = await TokenService.getCachedUserInfo(sampleToken)

    expect(result).toBeNull()
  })

  it('AUTH-UT-032 - ✓ Redis lỗi → trả về null (không crash toàn bộ request)', async () => {
    vi.mocked(redisClient.get).mockRejectedValue(new Error('Redis timeout'))

    const result = await TokenService.getCachedUserInfo(sampleToken)

    expect(result).toBeNull()
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// removeCachedUserInfo()
// ═════════════════════════════════════════════════════════════════════════════
describe('TokenService - removeCachedUserInfo()', () => {
  // AUTH-UT-033
  it('AUTH-UT-033 - Đăng xuất: xoá phiên khỏi cache → token vô hiệu hoá ngay lập tức', async () => {
    vi.mocked(redisClient.del).mockResolvedValue(1 as any)

    await TokenService.removeCachedUserInfo(sampleToken)

    expect(redisClient.del).toHaveBeenCalledWith(`auth:token:${sampleToken}`)
  })
})
