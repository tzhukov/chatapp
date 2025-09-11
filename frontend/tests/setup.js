// Mock window.__CHATAPP_CONFIG__ for tests
window.__CHATAPP_CONFIG__ = {
  VUE_APP_DEX_ISSUER_URL: 'https://test-issuer',
  VUE_APP_DEX_CLIENT_ID: 'test-client',
  VUE_APP_DEX_REDIRECT_URI: 'https://app/callback',
  VUE_APP_DEX_SCOPES: 'openid profile email',
  VUE_APP_API_BASE_URL: 'https://api',
  VUE_APP_WS_URL: 'wss://ws'
};

// No global Vue shim needed for Vue 3 with proper CJS mapping.
try {
  const vue = require('vue');
  if (!globalThis.Vue) {
    globalThis.Vue = vue;
  }
} catch {}
