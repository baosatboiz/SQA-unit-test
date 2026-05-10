import { EmbeddingService } from '../src/services/EmbeddingService'

// ─── Mock utilities ───────────────────────────────────────────────────────────

const mockRetryWithApiKey = jest.fn()

jest.mock('../src/utils/index', () => ({
  retryWithApiKey: (...args: any[]) => mockRetryWithApiKey(...args),
  logger: { error: jest.fn(), warn: jest.fn() },
}))

// ─── Mock KeyManager ──────────────────────────────────────────────────────────

const mockKeyManager = {}

// ─── Setup ────────────────────────────────────────────────────────────────────

let service: EmbeddingService

beforeEach(() => {
  service = new EmbeddingService({ keyManager: mockKeyManager as any })
})

afterEach(() => {
  // Rollback: xoá toàn bộ lịch sử gọi mock sau mỗi test
  jest.clearAllMocks()
})

// ═════════════════════════════════════════════════════════════════════════════
// embeddingText()
// ═════════════════════════════════════════════════════════════════════════════
describe('EmbeddingService - embeddingText()', () => {
  // CHAT-UT-056
  it('CHAT-UT-056 - Chuyển đổi text "Avengers Endgame" thành vector embedding → trả về mảng số thực', async () => {
    const mockVector = [0.12, -0.34, 0.56, 0.78]
    mockRetryWithApiKey.mockResolvedValue(mockVector)

    const result = await service.embeddingText('Avengers Endgame')

    expect(result).toEqual(mockVector)
    expect(Array.isArray(result)).toBe(true)
    expect(result.length).toBeGreaterThan(0)
  })

  // CHAT-UT-057 — Empty input
  it('CHAT-UT-057 - Input rỗng → Gemini trả về embeddings rỗng → trả về []', async () => {
    mockRetryWithApiKey.mockResolvedValue([])

    const result = await service.embeddingText('')

    expect(result).toEqual([])
  })

  // CHAT-UT-058 — Gemini timeout / API error
  it('CHAT-UT-058 - Gemini API timeout → retryWithApiKey ném lỗi, embeddingText ném lỗi lên caller', async () => {
    mockRetryWithApiKey.mockRejectedValue(new Error('Request timeout'))

    await expect(service.embeddingText('phim hành động')).rejects.toThrow()
    // Caller (ChatService) catches this and returns [] gracefully — tested in chatService.test.ts
  })
})
