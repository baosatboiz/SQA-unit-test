import { cosineSimilarity } from '../src/utils/similarity'

// ─── Pure function — no mocks needed ─────────────────────────────────────────
describe('cosineSimilarity()', () => {
  // CHAT-UT-059
  it('CHAT-UT-059 - cosineSimilarity - Vector âm → kết quả trong khoảng [-1, 1], hai vector ngược chiều = -1.0', () => {
    console.log('[CHAT-UT-059] Vector âm → kiểm tra cosineSimilarity trả đúng -1.0')
    const v1 = [1, 0]
    const v2 = [-1, 0]

    const result = cosineSimilarity(v1, v2)

    expect(result).toBeCloseTo(-1.0, 5)
    // Đảm bảo kết quả vẫn nằm trong khoảng hợp lệ [-1, 1]
    expect(result).toBeGreaterThanOrEqual(-1)
    expect(result).toBeLessThanOrEqual(1)
  })
})
