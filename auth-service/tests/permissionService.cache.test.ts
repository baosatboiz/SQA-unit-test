import { describe, it, expect, vi, afterEach } from 'vitest'

vi.mock('../src/config/redis.js', () => ({
  redisClient: {
    get: vi.fn(),
    setEx: vi.fn(),
    del: vi.fn(),
    keys: vi.fn(),
  },
}))

vi.mock('../src/services/userGrpcClient.js', () => ({
  userClient: {
    GetPermissionsByRoleId: vi.fn(),
  },
}))

import { redisClient } from '../src/config/redis.js'
import { PermissionService } from '../src/services/permissionService.js'

afterEach(() => {
  vi.clearAllMocks()
})

const managerRoleId = 'role-manager-uuid'

// ═════════════════════════════════════════════════════════════════════════════
// clearPermissionsCache()
// ═════════════════════════════════════════════════════════════════════════════
describe('PermissionService - clearPermissionsCache()', () => {
  // AUTH-UT-018
  it('AUTH-UT-018 - Xoá cache quyền sau khi admin thay đổi phân quyền → del key ngay lập tức', async () => {
    vi.mocked(redisClient.del).mockResolvedValue(1 as any)

    await PermissionService.clearPermissionsCache(managerRoleId)

    expect(redisClient.del).toHaveBeenCalledWith(`permissions:${managerRoleId}`)
  })

  // AUTH-UT-019
  it('AUTH-UT-019 - Xoá cache của role chưa từng được cache → hoàn tất không lỗi (idempotent)', async () => {
    vi.mocked(redisClient.del).mockResolvedValue(0 as any)

    await expect(PermissionService.clearPermissionsCache('uncached-role-uuid')).resolves.toBeUndefined()
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// clearAllPermissionsCache()
// ═════════════════════════════════════════════════════════════════════════════
describe('PermissionService - clearAllPermissionsCache()', () => {
  // AUTH-UT-020
  it('AUTH-UT-020 - Xoá toàn bộ cache quyền sau tái cấu trúc phân quyền → del tất cả key permissions:*', async () => {
    const allPermKeys = ['permissions:role-1', 'permissions:role-2', 'permissions:role-3']
    vi.mocked(redisClient.keys).mockResolvedValue(allPermKeys as any)
    vi.mocked(redisClient.del).mockResolvedValue(3 as any)

    await PermissionService.clearAllPermissionsCache()

    expect(redisClient.del).toHaveBeenCalledWith(allPermKeys)
  })
})
