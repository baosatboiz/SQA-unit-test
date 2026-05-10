import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { Op } from 'sequelize'
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

describe('UserService - Read Operations', () => {
  let userService: UserService

  beforeEach(() => {
    vi.clearAllMocks()
    userService = new UserService(mockModels)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  describe('getUserById() - Lấy thông tin user theo ID', () => {
    // USER-UT-008: Lấy user hợp lệ → trả về user info, KHÔNG có trường password
    it('USER-UT-008 - ✓ userId hợp lệ → trả về user info (loại bỏ password)', async () => {
      const fullUserData = {
        id: 'user-uuid-001',
        name: 'Nguyen Van A',
        email: 'user@gmail.com',
        phone_number: '0901234567',
        address: '123 Main St',
        status: 'ACTIVE',
        role_id: 'role-customer-uuid',
        password: 'hashed-password-should-not-appear',
      }
      mockUserInstance.toJSON.mockReturnValue(fullUserData)
      mockModels.User.findOne.mockResolvedValue(mockUserInstance)

      const result = await userService.getUserById('user-uuid-001')

      expect(result).not.toHaveProperty('password')
      expect(result.id).toBe('user-uuid-001')
      expect(result.email).toBe('user@gmail.com')
      expect(result.name).toBe('Nguyen Van A')
    })

    // USER-UT-009: User không tồn tại → throw 'User not found'
    it('USER-UT-009 - ✗ User không tồn tại → throw ErrorMessages.USER_NOT_FOUND', async () => {
      mockModels.User.findOne.mockResolvedValue(null)

      await expect(
        userService.getUserById('nonexistent-uuid'),
      ).rejects.toThrow(ErrorMessages.USER_NOT_FOUND)
    })

    it('USER-UT-010 - ✗ userId rỗng → throw ErrorMessages.INVALID_REQUEST', async () => {
      await expect(
        userService.getUserById(''),
      ).rejects.toThrow(ErrorMessages.INVALID_REQUEST)
    })
  })

  describe('getAllUsers() - Danh sách user với pagination & filtering', () => {
    // USER-UT-011: Lấy page 1 với size 10 → trả về đúng dữ liệu và paging info
    it('USER-UT-011 - ✓ page=1, size=10 → trả về 10 user và paging info chính xác', async () => {
      const mockUsers = Array.from({ length: 10 }, (_, i) => ({
        toJSON: () => ({ id: `user-${i}`, name: `User ${i}`, email: `user${i}@gmail.com` }),
      }))
      mockModels.User.count.mockResolvedValue(50)
      mockModels.User.findAll.mockResolvedValue(mockUsers)

      const result = await userService.getAllUsers({ page: 1, size: 10 })

      expect(result.data).toHaveLength(10)
      expect(result.paging.page).toBe(1)
      expect(result.paging.size).toBe(10)
      expect(result.paging.total).toBe(50)
      expect(result.paging.total_pages).toBe(5)
    })

    // USER-UT-012: Lọc theo role → chỉ query user có role đó
    it('USER-UT-012 - ✓ Filter theo role="customer" → truyền where={name:"customer"} vào include query', async () => {
      console.log('[USER-UT-012] getAllUsers role filter → kiểm tra where clause đúng trong include')
      mockModels.User.count.mockResolvedValue(3)
      mockModels.User.findAll.mockResolvedValue([])

      await userService.getAllUsers({ page: 1, size: 10, role: 'customer' })

      expect(mockModels.User.findAll).toHaveBeenCalledWith(
        expect.objectContaining({
          include: expect.arrayContaining([
            expect.objectContaining({
              as: 'role',
              where: { name: 'customer' }, // simple equality, bukan Op.ne
            }),
          ]),
        }),
      )
    })

    it('USER-UT-013 - ✓ Offset tính đúng: page=2, size=10 → offset=10', async () => {
      mockModels.User.count.mockResolvedValue(50)
      mockModels.User.findAll.mockResolvedValue([])

      await userService.getAllUsers({ page: 2, size: 10 })

      expect(mockModels.User.findAll).toHaveBeenCalledWith(
        expect.objectContaining({ offset: 10, limit: 10 }),
      )
    })

    it('USER-UT-014 - ✓ Kết quả rỗng → trả về data=[], total=0', async () => {
      mockModels.User.count.mockResolvedValue(0)
      mockModels.User.findAll.mockResolvedValue([])

      const result = await userService.getAllUsers({ page: 1, size: 10 })

      expect(result.data).toHaveLength(0)
      expect(result.paging.total).toBe(0)
      expect(result.paging.total_pages).toBe(0)
    })
  })

  describe('getAllStaffs() - Danh sách nhân viên (loại trừ customer)', () => {
    it('USER-UT-015 - ✓ Query loại trừ role "customer" → include where sử dụng Op.ne để exclude customer', async () => {
      console.log('[USER-UT-015] getAllStaffs → kiểm tra Op.ne loại trừ customer trong where clause')
      mockModels.User.count.mockResolvedValue(5)
      mockModels.User.findAll.mockResolvedValue([])

      await userService.getAllStaffs({ page: 1, size: 10 })

      expect(mockModels.User.findAll).toHaveBeenCalledWith(
        expect.objectContaining({
          include: expect.arrayContaining([
            expect.objectContaining({
              as: 'role',
              where: expect.objectContaining({
                name: { [Op.ne]: 'customer' }, // phải dùng Op.ne, KHÔNG phải string equality
              }),
            }),
          ]),
        }),
      )
    })
  })
})
