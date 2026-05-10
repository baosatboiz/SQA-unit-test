import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    include: ['tests/**/*.test.ts'],
    reporters: ['./tests/reporter.ts'],
    environment: 'node',
    clearMocks: true,
    restoreMocks: true,
    coverage: {
      provider: 'v8',
      include: [
        'src/services/tokenService.ts',
        'src/services/permissionService.ts',
        'src/controllers/authController.ts',
      ],
      exclude: ['tests/**', 'coverage/**', 'dist/**'],
      reporter: ['text', 'html', 'json', 'json-summary', 'lcov'],
      reportsDirectory: './coverage',
      reportOnFailure: true,
      thresholds: {
        lines: 75,
        functions: 90,
        branches: 65,
        statements: 75,
      },
    }
  }
});
