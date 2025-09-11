module.exports = {
  testEnvironment: 'jsdom',
  moduleFileExtensions: ['js', 'json', 'vue'],
  transform: {
    '^.+\\.(vue)$': ['@vue/vue3-jest'],
    '^.+\\.js$': ['babel-jest']
  },
  setupFilesAfterEnv: ['<rootDir>/tests/setup.js'],
  testMatch: ['**/tests/unit/**/*.spec.[jt]s'],
  moduleNameMapper: {
  '^@/(.*)$': '<rootDir>/src/$1',
  '^@vue/test-utils$': '<rootDir>/node_modules/@vue/test-utils/dist/vue-test-utils.cjs.js'
  }
};
