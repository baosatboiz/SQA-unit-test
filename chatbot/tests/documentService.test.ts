import { DocumentService } from '../src/services/DocumentService'
import { DocumentStatus } from '../src/models'

// ─── Mock datastores ──────────────────────────────────────────────────────────

const mockDocumentDatastore = {
  createDocument: jest.fn().mockResolvedValue(undefined),
  updateDocumentStatus: jest.fn().mockResolvedValue(undefined),
  getDocument: jest.fn(),
  getAllDocuments: jest.fn(),
  deleteDocument: jest.fn().mockResolvedValue(undefined),
}

const mockChunkDatastore = {
  batchCreateChunks: jest.fn().mockResolvedValue(undefined),
  deleteChunksByDocumentId: jest.fn().mockResolvedValue(undefined),
  getChunksByDocumentId: jest.fn(),
}

jest.mock('../src/datastore/DocumentDatastore', () => ({
  DocumentDatastore: jest.fn().mockImplementation(() => mockDocumentDatastore),
}))

jest.mock('../src/datastore/ChunkDatastore', () => ({
  ChunkDatastore: jest.fn().mockImplementation(() => mockChunkDatastore),
}))

// ─── Mock TextExtractor and utilities ────────────────────────────────────────

const mockExtractor = {
  validateFile: jest.fn().mockResolvedValue(undefined),
  extractText: jest.fn().mockResolvedValue('Đây là nội dung tài liệu về lịch chiếu phim.'),
  getFileInfo: jest.fn().mockResolvedValue({ size: 102400 }),
}

jest.mock('../src/utils/index', () => ({
  TextExtractor: jest.fn().mockImplementation(() => mockExtractor),
  splitIntoChunks: jest.fn().mockReturnValue([
    { content: 'Chunk 1 content', startPos: 0, endPos: 100, tokenCount: 25 },
    { content: 'Chunk 2 content', startPos: 80, endPos: 200, tokenCount: 30 },
  ]),
  createDirIfNotExists: jest.fn().mockResolvedValue(undefined),
  sanitizeFilename: jest.fn((name: string) => name),
}))

// ─── Mock EmbeddingService and CacheManager ───────────────────────────────────

const mockEmbeddingService = {
  embeddingText: jest.fn().mockResolvedValue([0.1, 0.2, 0.3]),
}

const mockCacheManager = {
  get: jest.fn(),
  set: jest.fn().mockResolvedValue(undefined),
  invalidatePattern: jest.fn().mockResolvedValue(undefined),
}

// ─── Setup ────────────────────────────────────────────────────────────────────

let service: DocumentService

beforeEach(() => {
  service = new DocumentService({
    pool: {} as any,
    cacheManager: mockCacheManager as any,
    embeddingService: mockEmbeddingService as any,
  })
})

afterEach(() => {
  // Rollback: xoá toàn bộ lịch sử gọi mock sau mỗi test
  jest.clearAllMocks()
})

// ═════════════════════════════════════════════════════════════════════════════
// processDocument()
// ═════════════════════════════════════════════════════════════════════════════
describe('DocumentService - processDocument()', () => {
  // CHAT-UT-050
  it('CHAT-UT-050 - Admin upload tài liệu PDF → trả về ngay với status PROCESSING, xử lý nền', async () => {
    const result = await service.processDocument('./uploads/lich-chieu-thang-5.pdf', 'Lịch chiếu tháng 5/2025')

    expect(result).toHaveProperty('id')
    expect(result.title).toBe('Lịch chiếu tháng 5/2025')
    expect(result.status).toBe(DocumentStatus.PROCESSING)
    expect(mockDocumentDatastore.createDocument).toHaveBeenCalled()
  })

  // CHAT-UT-051 — Business requirement: detect duplicate content
  // NOTE: Current implementation has NO duplicate detection (no hash comparison)
  // This test documents the feature gap — should be implemented
  it.skip('CHAT-UT-051 - Upload tài liệu trùng nội dung → phát hiện trùng lặp qua hash (tính năng chưa có trong code)', () => {
    // Current DocumentService does NOT hash content for duplicate detection.
    // Business requirement: compare content hash before creating document record.
    // Implementing this requires: generateTextHash(content), check DocumentDatastore for existing hash.
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// deleteDocument()
// ═════════════════════════════════════════════════════════════════════════════
describe('DocumentService - deleteDocument()', () => {
  // CHAT-UT-052
  it('CHAT-UT-052 - Xoá tài liệu và tất cả chunks liên quan → chatbot không còn dùng tài liệu này', async () => {
    await service.deleteDocument('doc-uuid-001')

    expect(mockChunkDatastore.deleteChunksByDocumentId).toHaveBeenCalledWith('doc-uuid-001')
    expect(mockDocumentDatastore.deleteDocument).toHaveBeenCalledWith('doc-uuid-001')
    expect(mockCacheManager.invalidatePattern).toHaveBeenCalledWith('document_chunks')
  })

  // CHAT-UT-053 — Business requirement: 404 for not found
  // NOTE: Current implementation deletes without checking existence → no 404 error raised
  it.skip('CHAT-UT-053 - Xoá tài liệu không tồn tại → lỗi 404 Not Found (hiện tại code không kiểm tra tồn tại — bug)', () => {
    // DocumentService.deleteDocument() does not call getDocument() first.
    // Business requirement: check document exists, throw 404 if not.
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// getDocument()
// ═════════════════════════════════════════════════════════════════════════════
describe('DocumentService - getDocument()', () => {
  it('CHAT-UT-054 - getDocument - ID tồn tại → trả về thông tin tài liệu', async () => {
    const mockDoc = { id: 'doc-001', title: 'Lịch chiếu', status: 'COMPLETED' }
    mockDocumentDatastore.getDocument.mockResolvedValue(mockDoc)

    const result = await service.getDocument('doc-001')

    expect(result).toEqual(mockDoc)
  })

  it('CHAT-UT-055 - getDocument - ID không tồn tại → trả về null', async () => {
    mockDocumentDatastore.getDocument.mockResolvedValue(null)

    const result = await service.getDocument('nonexistent')

    expect(result).toBeNull()
  })
})

// ═════════════════════════════════════════════════════════════════════════════
// processDocument() — kiểm tra chi tiết data và early exit
// CHAT-UT-060 - CHAT-UT-061
// ═════════════════════════════════════════════════════════════════════════════
describe('DocumentService - processDocument() DB check và validation', () => {
  // CHAT-UT-060
  it('CHAT-UT-060 - ✓ processDocument file hợp lệ → createDocument được gọi ngay với status=PROCESSING, đúng metadata', async () => {
    console.log('[CHAT-UT-060] processDocument thành công → kiểm tra createDocument với đúng title, status, file_path')

    const result = await service.processDocument('./uploads/schedule.pdf', 'Lịch chiếu mới nhất')

    // Document trả về ngay với status PROCESSING (chunk processing chạy nền)
    expect(result.status).toBe(DocumentStatus.PROCESSING)
    expect(result.id).toBeTruthy()

    // Check DB: createDocument phải được gọi với đúng dữ liệu
    expect(mockDocumentDatastore.createDocument).toHaveBeenCalledTimes(1)
    expect(mockDocumentDatastore.createDocument).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Lịch chiếu mới nhất',
        file_path: './uploads/schedule.pdf',
        status: DocumentStatus.PROCESSING,
      }),
    )

    // validateFile và extractText phải được gọi trước khi lưu DB
    expect(mockExtractor.validateFile).toHaveBeenCalledWith('./uploads/schedule.pdf')
    expect(mockExtractor.extractText).toHaveBeenCalledWith('./uploads/schedule.pdf')
  })

  // CHAT-UT-061
  it('CHAT-UT-061 - ✗ File không hợp lệ (validateFile throw) → throw error, createDocument không được gọi (early exit trước DB)', async () => {
    console.log('[CHAT-UT-061] validateFile lỗi → kiểm tra không tạo DB record khi file không hợp lệ')
    mockExtractor.validateFile.mockRejectedValue(new Error('Unsupported file type'))

    await expect(
      service.processDocument('./uploads/malware.exe', 'File nguy hiểm'),
    ).rejects.toThrow('Unsupported file type')

    // Check DB: KHÔNG tạo record khi file chưa được validate
    expect(mockDocumentDatastore.createDocument).not.toHaveBeenCalled()
    expect(mockEmbeddingService.embeddingText).not.toHaveBeenCalled()
    expect(mockChunkDatastore.batchCreateChunks).not.toHaveBeenCalled()
  })
})
