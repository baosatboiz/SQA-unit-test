import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { UserService } from '../src/services/userService.js'
import { ErrorMessages } from '../src/types/index.js'

const mockUserInstance = {
  toJSON: vi.fn(),
  update: vi.fn(),
  destroy: vi.fn(),
}

const mockModels = {
  User: {
    findOne: vi.fn(),
    findAll: vi.fn(),
    count: vi.fn(),
  },
  Role: {
    findOne: vi.fn(),
    findAll: vi.fn(),
  },
} as any

describe('UserService - Write Operations', () => {
  let userService: UserService

  beforeEach(() => {
    vi.clearAllMocks()
    userService = new UserService(mockModels)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  describe('updateUser() - Cập nhật thông tin user', () => {
    // USER-UT-016: Cập nhật tên và số điện thoại → thành công
    it('USER-UT-016 - ✓ Cập nhật name và phone_number hợp lệ → thành công', async () => {
      const updatedData = {
        id: 'user-uuid-001',
        name: 'Trang Nguyen',
        email: 'user@gmail.com',
        phone_number: '0987654321',
        status: 'ACTIVE',
      }
      mockUserInstance.toJSON.mockReturnValue(updatedData)
      mockUserInstance.update.mockResolvedValue(undefined)
      mockModels.User.findOne.mockResolvedValue(mockUserInstance)

      const result = await userService.updateUser(
        'user-uuid-001',
        { name: 'Trang Nguyen', phone_number: '0987654321' },
        'user-uuid-001',
      )

      expect(result.name).toBe('Trang Nguyen')
      expect(result).not.toHaveProperty('password')
      expect(mockUserInstance.update).toHaveBeenCalledWith(
        expect.objectContaining({ name: 'Trang Nguyen', phone_number: '0987654321' }),
      )
    })

    // USER-UT-017: User không tồn tại → throw USER_NOT_FOUND
    it('USER-UT-017 - ✗ User không tồn tại khi update → throw USER_NOT_FOUND', async () => {
      mockModels.User.findOne.mockResolvedValue(null)

      await expect(
        userService.updateUser('nonexistent-uuid', { name: 'New Name' }, 'nonexistent-uuid'),
      ).rejects.toThrow(ErrorMessages.USER_NOT_FOUND)
    })

    // USER-UT-018: User cố update người khác mà không có quyền → throw (privilege check)
    it('USER-UT-018 - ✗ User thường cố update profile của user khác → throw lỗi bảo mật', async () => {
      await expect(
        userService.updateUser(
          'other-user-uuid',
          { name: 'Hacker' },
          'my-user-uuid',
          'customer',
        ),
      ).rejects.toThrow('You can only update your own profile')
    })

    it('USER-UT-019 - ✓ Admin cập nhật user khác → được phép', async () => {
      const updatedData = {
        id: 'target-user-uuid',
        name: 'Updated Name',
        email: 'target@gmail.com',
        status: 'ACTIVE',
      }
      mockUserInstance.toJSON.mockReturnValue(updatedData)
      mockUserInstance.update.mockResolvedValue(undefined)
      mockModels.User.findOne.mockResolvedValue(mockUserInstance)

      const result = await userService.updateUser(
        'target-user-uuid',
        { name: 'Updated Name' },
        'admin-uuid',
        'admin',
      )

      expect(result.name).toBe('Updated Name')
    })

    it('USER-UT-020 - ✗ userId rỗng → throw INVALID_REQUEST', async () => {
      await expect(
        userService.updateUser('', { name: 'A' }),
      ).rejects.toThrow(ErrorMessages.INVALID_REQUEST)
    })

    // USER-UT-025 — SECURITY: Ngăn chặn privilege escalation
    it('USER-UT-025 - ✗ Gửi trường role trong updateData → bị bỏ qua hoàn toàn (silent ignore, không nâng quyền)', async () => {
      console.log('[USER-UT-025] role field trong updateData → kiểm tra bị loại bỏ khỏi DB update call')
      const userData = {
        id: 'user-uuid-001',
        name: 'Nguyen Van A',
        email: 'customer@gmail.com',
        status: 'ACTIVE',
      }
      mockUserInstance.toJSON.mockReturnValue(userData)
      mockModels.User.findOne.mockResolvedValue(mockUserInstance)

      await userService.updateUser(
        'user-uuid-001',
        { role: 'admin' } as any, // cố tình gửi trường role
        'user-uuid-001',
        'customer',
      )

      // Trường role KHÔNG được truyền vào user.update()
      const updateCall = mockUserInstance.update.mock.calls[0][0]
      expect(updateCall).not.toHaveProperty('role')
      expect(updateCall).not.toHaveProperty('role_id')
    })
  })

  describe('deleteUser() - Xóa user với kiểm soát bảo mật', () => {
    // USER-UT-021: Admin xóa customer khác → thành công
    it('USER-UT-021 - ✓ Admin xóa customer (ID khác nhau) → xóa thành công', async () => {
      mockUserInstance.toJSON.mockReturnValue({
        id: 'customer-uuid-001',
        role_id: 'role-customer-uuid',
      })
      mockUserInstance.destroy.mockResolvedValue(undefined)
      mockModels.User.findOne.mockResolvedValue(mockUserInstance)

      await expect(
        userService.deleteUser('customer-uuid-001', 'admin-uuid-001'),
      ).resolves.not.toThrow()

      expect(mockUserInstance.destroy).toHaveBeenCalled()
    })

    // USER-UT-022: Admin tự xóa chính mình → từ chối (self-delete prevention)
    it('USER-UT-022 - ✗ Admin tự xóa tài khoản của mình → throw "cannot delete own account"', async () => {
      await expect(
        userService.deleteUser('admin-uuid-001', 'admin-uuid-001'),
      ).rejects.toThrow('You cannot delete your own account')
    })

    it('USER-UT-023 - ✗ User không tồn tại → throw USER_NOT_FOUND', async () => {
      mockModels.User.findOne.mockResolvedValue(null)

      await expect(
        userService.deleteUser('nonexistent-uuid', 'admin-uuid-001'),
      ).rejects.toThrow(ErrorMessages.USER_NOT_FOUND)
    })

    it('USER-UT-024 - ✗ userId rỗng → throw INVALID_REQUEST', async () => {
      await expect(
        userService.deleteUser('', 'admin-uuid-001'),
      ).rejects.toThrow(ErrorMessages.INVALID_REQUEST)
    })
  })
})
