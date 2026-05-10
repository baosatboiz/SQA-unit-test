import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { Op } from 'sequelize'
import { RoleService } from '../src/services/roleService.js'

// ─── Mock Sequelize models ────────────────────────────────────────────────────

const mockRoleInstance = {
  toJSON: vi.fn().mockReturnValue({
    id: 'role-manager-uuid',
    name: 'manager_staff',
    description: 'Quản lý rạp chiếu phim',
  }),
}

const mockRoleModel = {
  findAll: vi.fn(),
  findOne: vi.fn(),
}

const mockPermissionModel = {
  findAll: vi.fn(),
}

const mockRolePermissionModel = {
  findAll: vi.fn(),
  findOne: vi.fn(),
  create: vi.fn(),
  destroy: vi.fn(),
  bulkCreate: vi.fn(),
}

const mockTransaction = {
  commit: vi.fn().mockResolvedValue(undefined),
  rollback: vi.fn().mockResolvedValue(undefined),
}

const mockModels = {
  Role: mockRoleModel,
  Permission: mockPermissionModel,
  RolePermission: mockRolePermissionModel,
  sequelize: {
    transaction: vi.fn().mockResolvedValue(mockTransaction),
  },
} as any

let service: RoleService

beforeEach(() => {
  service = new RoleService(mockModels)
  mockRoleModel.findOne.mockResolvedValue(mockRoleInstance)
})

afterEach(() => {
  // Rollback: xoá toàn bộ lịch sử gọi mock sau mỗi test
  vi.clearAllMocks()
})

// ═════════════════════════════════════════════════════════════════════════════
// getAllRoles()
// ═════════════════════════════════════════════════════════════════════════════
describe('RoleService - getAllRoles()', () => {
  // USER-UT-001
  it('USER-UT-001 - Lấy danh sách role nội bộ → KHÔNG bao gồm role "customer"', async () => {
    const internalRoles = [
      { toJSON: () => ({ id: 'r1', name: 'admin', description: 'Quản trị toàn hệ thống' }) },
      { toJSON: () => ({ id: 'r2', name: 'manager_staff', description: 'Quản lý rạp chiếu' }) },
      { toJSON: () => ({ id: 'r3', name: 'ticket_staff', description: 'Bán vé và kiểm tra vé' }) },
    ]
    mockRoleModel.findAll.mockResolvedValue(internalRoles)

    const result = await service.getAllRoles()

    expect(result).toHaveLength(3)
    expect(result.map((r) => r.name)).not.toContain('customer')
    // Verify query excluded 'customer'
    expect(mockRoleModel.findAll).toHaveBeenCalledWith(
      expect.objectContaining({
        where: expect.objectContaining({ name: expect.anything() }),
      }),
    )
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// getRoleWithPermissions()
// ═════════════════════════════════════════════════════════════════════════════
describe('RoleService - getRoleWithPermissions()', () => {
  // USER-UT-002
  it('USER-UT-002 - Xem chi tiết quyền hạn của manager_staff → trả về role và danh sách permissions', async () => {
    const mockPermissionObj = {
      toJSON: () => ({ id: 'perm-001', name: 'Tạo suất chiếu', code: 'SHOWTIME_CREATE' }),
    }
    mockRolePermissionModel.findAll.mockResolvedValue([
      { permission: mockPermissionObj },
    ])

    const result = await service.getRoleWithPermissions('role-manager-uuid')

    expect(result).toHaveProperty('name', 'manager_staff')
    expect(result.permissions).toHaveLength(1)
    expect(result.permissions[0]).toHaveProperty('code', 'SHOWTIME_CREATE')
  })

  // USER-UT-003
  it('USER-UT-003 - Xem role không tồn tại → ném lỗi "Role not found"', async () => {
    mockRoleModel.findOne.mockResolvedValue(null)

    await expect(service.getRoleWithPermissions('nonexistent-role')).rejects.toThrow('Role not found')
  })

  it('USER-UT-004 - getRoleWithPermissions - roleId rỗng → ném lỗi INVALID_REQUEST', async () => {
    await expect(service.getRoleWithPermissions('')).rejects.toThrow('Invalid request data')
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// updateRolePermissions()
// ═════════════════════════════════════════════════════════════════════════════
describe('RoleService - updateRolePermissions()', () => {
  // USER-UT-005
  it('USER-UT-005 - Cập nhật quyền hạn role (replace) → atomic transaction, xoá cũ và thêm mới', async () => {
    mockRolePermissionModel.destroy.mockResolvedValue(3)
    mockRolePermissionModel.bulkCreate.mockResolvedValue([])

    const newPermissionIds = ['showtime_create', 'seat_manage', 'report_view']
    await service.updateRolePermissions('role-manager-uuid', newPermissionIds)

    expect(mockModels.sequelize.transaction).toHaveBeenCalled()
    expect(mockRolePermissionModel.destroy).toHaveBeenCalledWith(
      expect.objectContaining({ where: { role_id: 'role-manager-uuid' } }),
    )
    expect(mockRolePermissionModel.bulkCreate).toHaveBeenCalledWith(
      expect.arrayContaining([
        expect.objectContaining({ role_id: 'role-manager-uuid' }),
      ]),
      expect.anything(),
    )
    expect(mockTransaction.commit).toHaveBeenCalled()
  })

  // USER-UT-006
  it('USER-UT-006 - Cập nhật quyền cho role không tồn tại → ném lỗi trước khi thay đổi dữ liệu', async () => {
    mockRoleModel.findOne.mockResolvedValue(null)

    await expect(
      service.updateRolePermissions('nonexistent-role', ['perm-001']),
    ).rejects.toThrow('Role not found')

    // No database mutation should have occurred
    expect(mockRolePermissionModel.destroy).not.toHaveBeenCalled()
    expect(mockRolePermissionModel.bulkCreate).not.toHaveBeenCalled()
  })

  it('USER-UT-007 - updateRolePermissions - transaction fail → rollback thực hiện', async () => {
    mockRolePermissionModel.destroy.mockResolvedValue(1)
    mockRolePermissionModel.bulkCreate.mockRejectedValue(new Error('DB error'))

    await expect(
      service.updateRolePermissions('role-manager-uuid', ['bad-perm']),
    ).rejects.toThrow('DB error')

    expect(mockTransaction.rollback).toHaveBeenCalled()
  })

  // USER-UT-026
  it('USER-UT-026 - ✓ updateRolePermissions với permissionIds=[] → destroy gọi để xoá hết, bulkCreate bị skip (no-op)', async () => {
    console.log('[USER-UT-026] permissionIds=[] → kiểm tra destroy được gọi, bulkCreate không được gọi')
    mockRolePermissionModel.destroy.mockResolvedValue(3)

    await service.updateRolePermissions('role-manager-uuid', [])

    // Xoá hết permissions cũ
    expect(mockRolePermissionModel.destroy).toHaveBeenCalledWith(
      expect.objectContaining({ where: { role_id: 'role-manager-uuid' } }),
    )
    // KHÔNG tạo mới vì danh sách rỗng — branch: if (permissionIds.length > 0)
    expect(mockRolePermissionModel.bulkCreate).not.toHaveBeenCalled()
    expect(mockTransaction.commit).toHaveBeenCalled()
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// getAllRoles() — kiểm tra Op.notIn loại trừ customer
// ═════════════════════════════════════════════════════════════════════════════
describe('RoleService - getAllRoles() query filter', () => {
  // USER-UT-027
  it('USER-UT-027 - ✓ getAllRoles → query dùng Op.notIn loại trừ "customer" khỏi danh sách role', async () => {
    console.log('[USER-UT-027] getAllRoles → kiểm tra Op.notIn: ["customer"] trong where clause')
    const internalRoles = [
      { toJSON: () => ({ id: 'r1', name: 'admin' }) },
      { toJSON: () => ({ id: 'r2', name: 'manager_staff' }) },
    ]
    mockRoleModel.findAll.mockResolvedValue(internalRoles)

    await service.getAllRoles()

    expect(mockRoleModel.findAll).toHaveBeenCalledWith(
      expect.objectContaining({
        where: {
          name: { [Op.notIn]: ['customer'] },
        },
      }),
    )
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// getRoleWithPermissions() — role không có permissions
// ═════════════════════════════════════════════════════════════════════════════
describe('RoleService - getRoleWithPermissions() edge cases', () => {
  // USER-UT-028
  it('USER-UT-028 - ✓ Role tồn tại nhưng chưa có permissions nào → trả về permissions=[] (mảng rỗng)', async () => {
    console.log('[USER-UT-028] Role không có permissions → kiểm tra permissions=[] trong kết quả trả về')
    // Role tồn tại
    mockRoleModel.findOne.mockResolvedValue(mockRoleInstance)
    // Không có RolePermission nào
    mockRolePermissionModel.findAll.mockResolvedValue([])

    const result = await service.getRoleWithPermissions('role-manager-uuid')

    expect(result).toHaveProperty('name', 'manager_staff')
    expect(result.permissions).toEqual([]) // permissions phải là mảng rỗng, không phải undefined
    expect(Array.isArray(result.permissions)).toBe(true)
  })
})
