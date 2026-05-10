import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

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
import { userClient } from '../src/services/userGrpcClient.js'
import { PermissionService } from '../src/services/permissionService.js'

afterEach(() => {
  vi.clearAllMocks()
})

const managerRoleId = 'role-manager-uuid'

const managerPermissions = [
  { id: 'perm-001', name: 'Tạo suất chiếu', code: 'SHOWTIME_CREATE', description: '' },
  { id: 'perm-002', name: 'Quản lý ghế', code: 'SEAT_MANAGE', description: '' },
  { id: 'perm-003', name: 'Xem doanh thu', code: 'REVENUE_REPORT_VIEW', description: '' },
]

// ═════════════════════════════════════════════════════════════════════════════
// getPermissionsByRoleId()
// ═════════════════════════════════════════════════════════════════════════════
describe('PermissionService - getPermissionsByRoleId()', () => {
  // AUTH-UT-021
  it('AUTH-UT-021 - Cache hit: quyền đã được cache → trả về ngay lập tức không gọi gRPC', async () => {
    vi.mocked(redisClient.get).mockResolvedValue(JSON.stringify(managerPermissions))

    const result = await PermissionService.getPermissionsByRoleId(managerRoleId)

    expect(result).toEqual(managerPermissions)
    expect(userClient.GetPermissionsByRoleId).not.toHaveBeenCalled()
  })

  // AUTH-UT-022
  it('AUTH-UT-022 - Cache miss: chưa có cache → tải từ gRPC và cache 30 phút', async () => {
    vi.mocked(redisClient.get).mockResolvedValue(null)
    vi.mocked(redisClient.setEx).mockResolvedValue('OK' as any)
    vi.mocked(userClient.GetPermissionsByRoleId).mockImplementation((_: any, cb: any) =>
      cb(null, { success: true, permissions: managerPermissions }),
    )

    const result = await PermissionService.getPermissionsByRoleId(managerRoleId)

    expect(result).toEqual(managerPermissions)
    expect(redisClient.setEx).toHaveBeenCalledWith(
      `permissions:${managerRoleId}`,
      1800,
      JSON.stringify(managerPermissions),
    )
  })

  // AUTH-UT-023 — Business requirement: FAIL-SECURE — when gRPC fails, throw error (not grant default permissions)
  it('AUTH-UT-023 - gRPC lỗi → ném lỗi (fail-secure: không cấp quyền mặc định khi không xác minh được)', async () => {
    vi.mocked(redisClient.get).mockResolvedValue(null)
    vi.mocked(userClient.GetPermissionsByRoleId).mockImplementation((_: any, cb: any) =>
      cb(new Error('user-service unavailable'), null),
    )

    await expect(PermissionService.getPermissionsByRoleId(managerRoleId)).rejects.toThrow()
  })

  // AUTH-UT-024
  it('AUTH-UT-024 - Role mới chưa được gán quyền → trả về danh sách rỗng []', async () => {
    vi.mocked(redisClient.get).mockResolvedValue(null)
    vi.mocked(redisClient.setEx).mockResolvedValue('OK' as any)
    vi.mocked(userClient.GetPermissionsByRoleId).mockImplementation((_: any, cb: any) =>
      cb(null, { success: true, permissions: [] }),
    )

    const result = await PermissionService.getPermissionsByRoleId('new-role-uuid')

    expect(result).toEqual([])
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// hasPermission()
// ═════════════════════════════════════════════════════════════════════════════
describe('PermissionService - hasPermission()', () => {
  beforeEach(() => {
    vi.mocked(redisClient.get).mockResolvedValue(JSON.stringify(managerPermissions))
  })

  // AUTH-UT-025
  it('AUTH-UT-025 - Manager có quyền SHOWTIME_CREATE → trả về true', async () => {
    const result = await PermissionService.hasPermission(managerRoleId, 'SHOWTIME_CREATE')
    expect(result).toBe(true)
  })

  // AUTH-UT-026
  it('AUTH-UT-026 - Ticket staff không có quyền REVENUE_REPORT_VIEW → trả về false', async () => {
    const ticketStaffPermissions = [
      { id: 'perm-010', name: 'Kiểm tra vé', code: 'TICKET_CHECK', description: '' },
    ]
    vi.mocked(redisClient.get).mockResolvedValue(JSON.stringify(ticketStaffPermissions))

    const result = await PermissionService.hasPermission('role-ticket-staff-uuid', 'REVENUE_REPORT_VIEW')
    expect(result).toBe(false)
  })

  // AUTH-UT-027 — Security: empty permission code must not accidentally grant access
  it('AUTH-UT-027 - Mã quyền rỗng → trả về false (không cấp quyền khi mã không hợp lệ)', async () => {
    const result = await PermissionService.hasPermission(managerRoleId, '')
    expect(result).toBe(false)
  })
})
