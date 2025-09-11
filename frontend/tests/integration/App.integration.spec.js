import { mount } from '@vue/test-utils';
import App from '@/App.vue';

jest.mock('oidc-client', () => ({
  UserManager: jest.fn().mockImplementation(() => ({
    getUser: jest.fn().mockResolvedValue({ profile: { name: 'alice' }, access_token: 'abc', expired: false }),
    signinRedirect: jest.fn(),
    signoutRedirect: jest.fn(),
    signinRedirectCallback: jest.fn(),
    signinSilent: jest.fn()
  })),
  WebStorageStateStore: jest.fn()
}));

// Mock fetch for messages
global.fetch = jest.fn().mockResolvedValue({
  ok: true,
  status: 200,
  json: async () => ([{ user_id: 'alice', content: 'hello from backend', timestamp: new Date().toISOString() }])
});

// Mock WebSocket
class FakeWebSocket {
  constructor() { this.readyState = 1; setTimeout(()=>{ this.onopen && this.onopen(); }, 0); }
  send() {}
  close() {}
}

global.WebSocket = FakeWebSocket;

window.__CHATAPP_CONFIG__ = {
  VUE_APP_DEX_ISSUER_URL: 'https://issuer',
  VUE_APP_DEX_CLIENT_ID: 'client',
  VUE_APP_DEX_REDIRECT_URI: 'https://app/',
  VUE_APP_DEX_SCOPES: 'openid',
  VUE_APP_API_BASE_URL: 'https://api',
  VUE_APP_WS_URL: 'wss://ws'
};

describe('App integration', () => {
  it('mounts and loads messages for General Chat', async () => {
    const wrapper = mount(App);
    // Wait for microtasks
    await new Promise(r => setTimeout(r, 0));
    expect(wrapper.html()).toContain('General Chat');
    expect(wrapper.html()).toContain('hello from backend');
  });
});
