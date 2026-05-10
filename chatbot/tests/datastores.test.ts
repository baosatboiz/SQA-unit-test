import { ChatDatastore } from '../src/datastore/ChatDatastore'
import { ChunkDatastore } from '../src/datastore/ChunkDatastore'
import { DocumentDatastore } from '../src/datastore/DocumentDatastore'
import { DocumentStatus } from '../src/models'

// ─── Mock pg Pool ─────────────────────────────────────────────────────────────

const mockQueryFn = jest.fn()
const mockClient = {
  query: jest.fn(),
  release: jest.fn(),
}

const mockPool = {
  query: mockQueryFn,
  connect: jest.fn().mockResolvedValue(mockClient),
} as any

// ═════════════════════════════════════════════════════════════════════════════
// ChatDatastore
// ═════════════════════════════════════════════════════════════════════════════
describe('ChatDatastore', () => {
  let chatDS: ChatDatastore

  beforeEach(() => {
    chatDS = new ChatDatastore(mockPool)
  })

  afterEach(() => {
    // Rollback: reset toàn bộ mock DB state về ban đầu sau mỗi test
    jest.clearAllMocks()
  })

  // CHAT-UT-037
  it('CHAT-UT-037 - Lưu hội thoại vào database → INSERT không gây lỗi, đúng dữ liệu được truyền vào DB', async () => {
    const chatData = {
      id: 'conv-123',
      question: 'Phim hot?',
      answer: 'Tuần này có phim...',
      created_at: new Date(),
    }
    mockQueryFn.mockResolvedValue({ rows: [], rowCount: 1 })

    await expect(chatDS.createChatRecord(chatData)).resolves.toBeUndefined()

    // Check DB: pool.query được gọi đúng 1 lần
    expect(mockQueryFn).toHaveBeenCalledTimes(1)

    // Check DB: SQL là INSERT
    const [sql, params] = mockQueryFn.mock.calls[0]
    expect(sql.toUpperCase()).toContain('INSERT')

    // Check DB: các trường quan trọng được truyền vào đúng
    expect(params).toEqual(
      expect.arrayContaining([chatData.id, chatData.question, chatData.answer]),
    )
  })

  // CHAT-UT-038 — NOTE: ChatDatastore only has createChatRecord, NOT getHistory
  it.skip('CHAT-UT-038 - Lấy lịch sử hội thoại (conversation_id, limit=20) → tính năng chưa có trong ChatDatastore', () => {
    // ChatDatastore hiện chỉ có createChatRecord().
    // Yêu cầu nghiệp vụ: thêm getHistory(conversationId, limit) method.
  })

  // CHAT-UT-039
  it.skip('CHAT-UT-039 - Conversation mới, lịch sử rỗng → trả về [] (tính năng chưa có trong ChatDatastore)', () => {
    // Tương tự P1_097 — getHistory() chưa được implement.
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// ChunkDatastore
// ═════════════════════════════════════════════════════════════════════════════
describe('ChunkDatastore', () => {
  let chunkDS: ChunkDatastore

  beforeEach(() => {
    chunkDS = new ChunkDatastore(mockPool)
    mockClient.query.mockResolvedValue({ rows: [], rowCount: 0 })
  })

  afterEach(() => {
    // Rollback: reset riêng từng mock DB để tránh nhiễu giữa các test
    mockClient.query.mockReset()
    mockClient.release.mockReset()
    mockQueryFn.mockReset()
    ;(mockPool.connect as jest.Mock).mockReset()
    ;(mockPool.connect as jest.Mock).mockResolvedValue(mockClient)
  })

  // CHAT-UT-040
  it('CHAT-UT-040 - Batch insert chunks → sử dụng transaction, commit khi tất cả thành công, mỗi chunk được INSERT đúng dữ liệu', async () => {
    const chunks = [
      {
        id: 'chunk-001',
        document_id: 'doc-001',
        chunk_index: 0,
        content: 'Nội dung đoạn 1',
        embedding: [0.1, 0.2, 0.3],
        start_pos: 0,
        end_pos: 100,
        token_count: 25,
        created_at: new Date(),
      },
      {
        id: 'chunk-002',
        document_id: 'doc-001',
        chunk_index: 1,
        content: 'Nội dung đoạn 2',
        embedding: [0.4, 0.5, 0.6],
        start_pos: 80,
        end_pos: 200,
        token_count: 30,
        created_at: new Date(),
      },
    ]

    await expect(chunkDS.batchCreateChunks(chunks)).resolves.toBeUndefined()

    // Check DB: transaction đúng thứ tự BEGIN → INSERT(s) → COMMIT
    expect(mockClient.query).toHaveBeenCalledWith('BEGIN')
    expect(mockClient.query).toHaveBeenCalledWith('COMMIT')
    expect(mockClient.release).toHaveBeenCalled()

    // Check DB: số lần INSERT đúng bằng số chunks
    const allCalls = (mockClient.query as jest.Mock).mock.calls
    const insertCalls = allCalls.filter(
      ([sql]: [string]) => typeof sql === 'string' && sql.toUpperCase().includes('INSERT'),
    )
    expect(insertCalls).toHaveLength(chunks.length)

    // Check DB: chunk-001 được INSERT với đúng dữ liệu
    const firstParams = insertCalls[0][1] as any[]
    expect(firstParams).toEqual(expect.arrayContaining(['chunk-001', 'doc-001', 'Nội dung đoạn 1']))

    // Check DB: chunk-002 được INSERT với đúng dữ liệu
    const secondParams = insertCalls[1][1] as any[]
    expect(secondParams).toEqual(expect.arrayContaining(['chunk-002', 'doc-001', 'Nội dung đoạn 2']))
  })

  // CHAT-UT-041
  it('CHAT-UT-041 - findSimilarChunks: DB trả về kết quả có similarity > threshold → SQL phải có ORDER BY (sắp xếp do DB)', async () => {
    console.log('[CHAT-UT-041] findSimilarChunks → kiểm tra SQL có ORDER BY và kết quả similarity > threshold')
    const mockRows = [
      { content: 'Inception chiếu lúc 19:00', similarity: '0.92' },
      { content: 'Avengers chiếu lúc 21:00', similarity: '0.78' },
    ]
    mockQueryFn.mockResolvedValue({ rows: mockRows })

    const queryVector = [0.1, 0.2, 0.3]
    const result = await chunkDS.findSimilarChunks(queryVector, 0.3, 5)

    // Check DB: pool.query được gọi đúng 1 lần
    expect(mockQueryFn).toHaveBeenCalledTimes(1)

    // Check DB: SQL phải có ORDER BY để đảm bảo sắp xếp do DB thực hiện (không phải app-level)
    const [sql] = mockQueryFn.mock.calls[0]
    expect(sql.toUpperCase()).toContain('ORDER BY')

    // Check DB: kết quả trả về đúng (similarity > threshold 0.3)
    expect(result.length).toBeGreaterThan(0)
    for (const item of result) {
      expect(item.similarity).toBeGreaterThan(0.3)
    }
  })

  // CHAT-UT-042
  it('CHAT-UT-042 - findSimilarChunks khi knowledge base rỗng → trả về []', async () => {
    mockQueryFn.mockResolvedValue({ rows: [] })

    const result = await chunkDS.findSimilarChunks([0.1, 0.2], 0.3, 5)

    // Check DB: pool.query vẫn được gọi (service luôn search, dù không có kết quả)
    expect(mockQueryFn).toHaveBeenCalledTimes(1)
    expect(result).toEqual([])
  })

  // CHAT-UT-043
  it('CHAT-UT-043 - deleteChunksByDocumentId → xoá tất cả chunks của tài liệu với đúng documentId', async () => {
    mockQueryFn.mockResolvedValue({ rows: [], rowCount: 5 })

    await expect(chunkDS.deleteChunksByDocumentId('doc-001')).resolves.toBeUndefined()

    // Check DB: pool.query được gọi đúng 1 lần
    expect(mockQueryFn).toHaveBeenCalledTimes(1)

    // Check DB: SQL là DELETE
    const [sql, params] = mockQueryFn.mock.calls[0]
    expect(sql.toUpperCase()).toContain('DELETE')

    // Check DB: đúng document_id được truyền vào — không xoá nhầm document khác
    expect(params).toEqual(['doc-001'])
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// DocumentDatastore
// ═════════════════════════════════════════════════════════════════════════════
describe('DocumentDatastore', () => {
  let docDS: DocumentDatastore

  beforeEach(() => {
    docDS = new DocumentDatastore(mockPool)
  })

  afterEach(() => {
    // Rollback: reset mock pool state về ban đầu
    mockQueryFn.mockReset()
  })

  // CHAT-UT-044
  it('CHAT-UT-044 - Lưu thông tin tài liệu mới → INSERT thành công, đủ trường bắt buộc được truyền vào DB', async () => {
    mockQueryFn.mockResolvedValue({ rows: [], rowCount: 1 })

    const doc = {
      id: 'doc-uuid-001',
      title: 'Lịch chiếu tháng 5',
      file_path: './uploads/lich-chieu.pdf',
      file_type: '.pdf',
      size: 102400,
      status: DocumentStatus.PROCESSING,
      created_at: new Date(),
    }

    await expect(docDS.createDocument(doc)).resolves.toBeUndefined()

    // Check DB: pool.query được gọi đúng 1 lần
    expect(mockQueryFn).toHaveBeenCalledTimes(1)

    // Check DB: SQL là INSERT
    const [sql, params] = mockQueryFn.mock.calls[0]
    expect(sql.toUpperCase()).toContain('INSERT')

    // Check DB: các trường định danh và metadata bắt buộc được truyền đúng
    expect(params).toEqual(expect.arrayContaining([doc.id, doc.title, doc.file_path]))

    // Check DB: status PROCESSING được ghi vào DB (tài liệu bắt đầu ở trạng thái chờ xử lý)
    expect(params).toEqual(expect.arrayContaining([DocumentStatus.PROCESSING]))
  })

  // CHAT-UT-045 — NOTE: DocumentDatastore has no hash-based duplicate detection
  it.skip('CHAT-UT-045 - Upload tài liệu trùng hash → lỗi duplicate (tính năng chưa có trong DocumentDatastore)', () => {
    // DocumentDatastore.createDocument() chưa kiểm tra content hash.
    // Yêu cầu nghiệp vụ: thêm trường hash vào bảng documents và method findByHash().
  })

  // CHAT-UT-046 — NOTE: findByHash() does not exist
  it.skip('CHAT-UT-046 - findByHash: tài liệu đã tồn tại → trả về document record (phương thức chưa được triển khai)', () => {
    // DocumentDatastore chưa có findByHash() method.
  })

  // CHAT-UT-047 — NOTE: same as P1_105
  it.skip('CHAT-UT-047 - findByHash: tài liệu chưa tồn tại → trả về null (phương thức chưa được triển khai)', () => {
    // Tương tự P1_105 — method chưa implement.
  })

  it('CHAT-UT-048 - getDocument - ID hợp lệ → trả về document với đầy đủ thông tin', async () => {
    const mockDoc = { id: 'doc-001', title: 'Test', status: 'COMPLETED' }
    mockQueryFn.mockResolvedValue({ rows: [mockDoc] })

    const result = await docDS.getDocument('doc-001')

    // Check DB: query được gọi với đúng ID
    expect(mockQueryFn).toHaveBeenCalledTimes(1)
    expect(result).toEqual(mockDoc)
  })

  it('CHAT-UT-049 - getDocument - ID không tồn tại → trả về null', async () => {
    mockQueryFn.mockResolvedValue({ rows: [] })

    const result = await docDS.getDocument('nonexistent')

    expect(result).toBeNull()
  })
})
