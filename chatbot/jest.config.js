/** @type {import('ts-jest').JestConfigWithTsJest} */
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  testMatch: ['**/tests/**/*.test.ts'],
  collectCoverageFrom: [
    'src/services/ChatService.ts',
    'src/services/DocumentService.ts',
    'src/services/EmbeddingService.ts',
  ],
  coverageReporters: ['text', 'html', 'json', 'json-summary', 'lcov'],
  coveragePathIgnorePatterns: ['/node_modules/', '/tests/', '/coverage/', '/dist/'],
  coverageThreshold: {
    global: {
      lines: 80,
      functions: 60,
      branches: 30,
      statements: 80,
    },
  },
  setupFilesAfterEnv: ['./tests/setup.js'],
  testTimeout: 3000,
  clearMocks: true,
  restoreMocks: true,
  reporters: ['./tests/reporter.js'],
  moduleNameMapper: {
    // Resolve .js imports to TypeScript source files
    '^(\\.{1,2}/.*)\\.js$': '$1',
  },
  transform: {
    '^.+\\.tsx?$': ['ts-jest', {
      isolatedModules: true,
      tsconfig: {
        module: 'commonjs',
        esModuleInterop: true,
      },
    }],
  },
};
