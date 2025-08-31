/// <reference types="@vitest/browser/matchers" />
/// <reference types="@vitest/browser/providers/playwright" />

// Mock environment variables for tests
Object.defineProperty(import.meta, 'env', {
  value: {
    VITE_POCKETBASE_URL: 'http://localhost:8090'
  }
});
