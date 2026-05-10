import { describe, it, expect } from '@jest/globals'
import { cosineSimilarity } from '../src/utils/similarity'
import { splitIntoChunks, generateTextHash } from '../src/utils/chunking'

// Pure utility functions - no mocks required

describe('cosineSimilarity() - Tính độ tương đồng vector', () => {
  // CHAT-UT-022: Hai vector giống nhau → similarity = 1.0
  it('CHAT-UT-022 - ✓ Hai vector giống nhau hoàn toàn → similarity = 1.0 (cache hit cho câu hỏi giống nhau)', () => {
    const v1 = [1, 0, 0]
    const v2 = [1, 0, 0]

    const result = cosineSimilarity(v1, v2)

    expect(result).toBeCloseTo(1.0, 5)
  })

  // CHAT-UT-023: Hai vector trực giao → similarity = 0.0
  it('CHAT-UT-023 - ✓ Hai vector trực giao (chủ đề hoàn toàn khác nhau) → similarity = 0.0', () => {
    const v1 = [1, 0]
    const v2 = [0, 1]

    const result = cosineSimilarity(v1, v2)

    expect(result).toBeCloseTo(0.0, 5)
  })

  it('CHAT-UT-024 - ✓ Vector zero → trả về 0.0 (không gây lỗi chia cho 0)', () => {
    const zeroVec = [0, 0, 0]
    const normalVec = [1, 0, 0]

    expect(cosineSimilarity(zeroVec, normalVec)).toBe(0)
    expect(cosineSimilarity(normalVec, zeroVec)).toBe(0)
  })

  it('CHAT-UT-025 - ✓ Độ dài vector khác nhau → trả về 0.0 (không crash)', () => {
    expect(cosineSimilarity([1, 0], [1, 0, 0])).toBe(0)
  })

  it('CHAT-UT-026 - ✓ Hai vector tương đồng → similarity > threshold 0.3', () => {
    const v1 = [1, 0.5, 0.2]
    const v2 = [0.95, 0.48, 0.25]

    const similarity = cosineSimilarity(v1, v2)

    expect(similarity).toBeGreaterThan(0.3)
  })
})

describe('splitIntoChunks() - Chia tài liệu thành chunks', () => {
  const fixedConfig = {
    maxSize: 800,
    overlap: 0,
    method: 'fixed' as const,
    minSize: 50,
    separators: [],
  }

  // CHAT-UT-027: Tài liệu dài → chia thành nhiều chunks, mỗi chunk ≤ maxSize
  it('CHAT-UT-027 - ✓ Text dài (2000 ký tự) → chia thành nhiều chunks, mỗi chunk ≤ 800 ký tự', () => {
    const longText = 'Đây là nội dung tài liệu về lịch chiếu phim tại rạp. '.repeat(40)

    const chunks = splitIntoChunks(longText, fixedConfig)

    expect(chunks.length).toBeGreaterThan(1)
    chunks.forEach((chunk) => {
      expect(chunk.content.length).toBeLessThanOrEqual(800)
    })
  })

  // CHAT-UT-028: Tài liệu ngắn (< maxSize) → không chia, trả về 1 chunk
  it('CHAT-UT-028 - ✓ Text ngắn (< 800 ký tự) → không chia, trả về đúng 1 chunk', () => {
    const shortText = 'Phim Inception chiếu lúc 10 giờ sáng.'

    const chunks = splitIntoChunks(shortText, { ...fixedConfig, minSize: 1 })

    expect(chunks.length).toBe(1)
    expect(chunks[0].content).toContain('Inception')
  })

  it('CHAT-UT-029 - ✓ Text rỗng → trả về mảng rỗng (không crash)', () => {
    const chunks = splitIntoChunks('', fixedConfig)

    expect(chunks).toEqual([])
  })

  it('CHAT-UT-030 - ✓ Mỗi chunk có đủ các trường: content, startPos, endPos, tokenCount', () => {
    const text = 'Nội dung phim A. '.repeat(60)

    const chunks = splitIntoChunks(text, { ...fixedConfig, minSize: 10 })

    chunks.forEach((chunk) => {
      expect(chunk).toHaveProperty('content')
      expect(chunk).toHaveProperty('startPos')
      expect(chunk).toHaveProperty('endPos')
      expect(chunk).toHaveProperty('tokenCount')
      expect(chunk.tokenCount).toBeGreaterThan(0)
    })
  })
})

describe('generateTextHash() - Tạo MD5 hash từ text', () => {
  it('CHAT-UT-031 - ✓ Cùng text → cùng hash (deterministic)', () => {
    const text = 'Phim gì đang chiếu tháng 5?'

    const hash1 = generateTextHash(text)
    const hash2 = generateTextHash(text)

    expect(hash1).toBe(hash2)
    expect(hash1).toMatch(/^[a-f0-9]{32}$/)
  })

  it('CHAT-UT-032 - ✓ Text khác nhau → hash khác nhau', () => {
    const hash1 = generateTextHash('Phim A')
    const hash2 = generateTextHash('Phim B')

    expect(hash1).not.toBe(hash2)
  })
})
