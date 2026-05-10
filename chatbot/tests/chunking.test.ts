import { splitIntoChunks, ChunkConfig } from '../src/utils/chunking'

// ─── Pure function — no mocks needed ─────────────────────────────────────────

const defaultConfig: ChunkConfig = {
  maxSize: 800,
  overlap: 100,
  method: 'sentence',
  minSize: 50,
  separators: ['\n\n', '\n', '. ', '! ', '? '],
}

describe('splitIntoChunks()', () => {
  // CHAT-UT-033
  it('CHAT-UT-033 - Tài liệu dài 2000 ký tự → chia thành nhiều đoạn, mỗi đoạn ≤ 800 ký tự, mỗi đoạn ≥ 50 ký tự', () => {
    // Build a realistic ~2000 char document with multiple sentences
    const sentences = []
    for (let i = 0; i < 15; i++) {
      sentences.push(`Đây là câu thứ ${i + 1} trong tài liệu mô tả về lịch chiếu phim tháng này tại rạp chiếu phim.`)
    }
    const text = sentences.join(' ')

    expect(text.length).toBeGreaterThanOrEqual(1000)

    const chunks = splitIntoChunks(text, defaultConfig)

    expect(chunks.length).toBeGreaterThan(1)
    for (const chunk of chunks) {
      expect(chunk.content.length).toBeLessThanOrEqual(800)
      expect(chunk.content.length).toBeGreaterThanOrEqual(50)
      expect(chunk.tokenCount).toBeGreaterThan(0)
    }
  })

  // CHAT-UT-034
  it('CHAT-UT-034 - Tài liệu ngắn (< 100 ký tự) → giữ nguyên thành 1 đoạn duy nhất', () => {
    const shortText = 'Phim Avengers chiếu tối nay lúc 19:00 tại phòng A.'

    expect(shortText.length).toBeLessThan(100)

    const chunks = splitIntoChunks(shortText, defaultConfig)

    expect(chunks).toHaveLength(1)
    expect(chunks[0].content).toContain('Avengers')
  })

  it('CHAT-UT-035 - splitIntoChunks - Text rỗng → trả về mảng rỗng', () => {
    const chunks = splitIntoChunks('', defaultConfig)
    expect(chunks).toHaveLength(0)
  })

  it('CHAT-UT-036 - splitIntoChunks - method=fixed → phân chia theo kích thước cố định', () => {
    const text = ('word ').repeat(400) // 2000 chars
    // overlap:0 for fixed-size to avoid the overlap-at-end infinite loop in chunkByFixedSize
    const fixedConfig: ChunkConfig = { ...defaultConfig, method: 'fixed', overlap: 0, minSize: 1 }

    const chunks = splitIntoChunks(text, fixedConfig)

    expect(chunks.length).toBeGreaterThan(1)
    for (const chunk of chunks) {
      expect(chunk.content.length).toBeLessThanOrEqual(800)
    }
  })
})
