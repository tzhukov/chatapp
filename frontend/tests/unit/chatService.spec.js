import { chatService } from '@/services/chatService.js';

// Mock OIDC UserManager internals by intercepting lazy getter
jest.mock('oidc-client', () => {
  return {
    UserManager: jest.fn().mockImplementation((cfg) => ({
      settings: cfg,
      getUser: jest.fn().mockResolvedValue({ access_token: 'abc', expired: false }),
      signinRedirect: jest.fn(),
      signoutRedirect: jest.fn(),
      signinRedirectCallback: jest.fn(),
      signinSilent: jest.fn().mockResolvedValue({ access_token: 'renewed' })
    })),
    WebStorageStateStore: jest.fn()
  };
});

describe('chatService runtime config', () => {
  beforeEach(() => {
    window.__CHATAPP_CONFIG__ = {
      VUE_APP_DEX_ISSUER_URL: 'https://issuer',
      VUE_APP_DEX_CLIENT_ID: 'client',
      VUE_APP_DEX_REDIRECT_URI: 'https://app/',
      VUE_APP_DEX_SCOPES: 'openid',
      VUE_APP_API_BASE_URL: 'https://api',
      VUE_APP_WS_URL: 'wss://ws'
    };
  });

  it('provides access token', async () => {
    const token = await chatService.getAccessToken();
    expect(token).toBe('abc');
  });
});
