/** @type {import('jest').Config} */
module.exports = {
  ...require('./jest.config'),
  testMatch: ['**/__tests__/integration/**/*.test.ts'],
  testTimeout: 60000, // Longer timeout for integration tests
  collectCoverage: false, // Don't collect coverage for integration tests
};
