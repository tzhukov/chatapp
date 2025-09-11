import { UserManager, WebStorageStateStore } from "oidc-client";

function resolveConfig() {
  const runtime = window.__CHATAPP_CONFIG__ || {};
  const authority = process.env.VUE_APP_DEX_ISSUER_URL || runtime.VUE_APP_DEX_ISSUER_URL;
  const clientId = process.env.VUE_APP_DEX_CLIENT_ID || runtime.VUE_APP_DEX_CLIENT_ID;
  const redirectUri = process.env.VUE_APP_DEX_REDIRECT_URI || runtime.VUE_APP_DEX_REDIRECT_URI;
  const scopes = process.env.VUE_APP_DEX_SCOPES || runtime.VUE_APP_DEX_SCOPES || "openid profile email";
  if (!authority) {
    throw new Error("OIDC authority not configured (VUE_APP_DEX_ISSUER_URL).");
  }
  return {
    authority,
    client_id: clientId,
    redirect_uri: redirectUri,
    response_type: "code",
    scope: scopes,
    post_logout_redirect_uri: redirectUri,
    userStore: new WebStorageStateStore({ store: window.localStorage }),
  };
}

let userManagerInstance;
function getUserManager() {
  if (!userManagerInstance) {
    userManagerInstance = new UserManager(resolveConfig());
  }
  return userManagerInstance;
}

function apiBase() {
  const runtime = window.__CHATAPP_CONFIG__ || {};
  return process.env.VUE_APP_API_BASE_URL || runtime.VUE_APP_API_BASE_URL;
}

function wsBase() {
  const runtime = window.__CHATAPP_CONFIG__ || {};
  return process.env.VUE_APP_WS_URL || runtime.VUE_APP_WS_URL;
}

export const chatService = {
  get userManager() { return getUserManager(); },

  async login() {
    return getUserManager().signinRedirect();
  },

  async logout() {
    return getUserManager().signoutRedirect();
  },

  async handleAuthentication() {
    return getUserManager().signinRedirectCallback();
  },

  async getUser() {
    return getUserManager().getUser();
  },

  async getAccessToken() {
    const user = await getUserManager().getUser();
    if (!user) return null;
    if (user.expired && user.refresh_token) {
      try {
        const renewed = await getUserManager().signinSilent();
        return renewed.access_token;
      } catch (e) {
        await getUserManager().signoutRedirect();
        throw new Error("Session expired. Please log in again.");
      }
    }
    return user.access_token;
  },

  async getMessages() {
    const token = await this.getAccessToken();
    if (!token) throw new Error("User not authenticated");
    const response = await fetch(`${apiBase()}/messages`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (response.status === 401) {
      await getUserManager().signoutRedirect();
      throw new Error("Session expired. Please log in again.");
    }
    if (!response.ok) throw new Error("Failed to fetch messages");
    return response.json();
  },

  async sendMessage(message) {
    const token = await this.getAccessToken();
    if (!token) throw new Error("User not authenticated");
    const response = await fetch(`${apiBase()}/messages`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify(message),
    });
    if (response.status === 401) {
      await getUserManager().signoutRedirect();
      throw new Error("Session expired. Please log in again.");
    }
    if (!response.ok) throw new Error("Failed to send message");
    return response.json();
  },

  async connectWebSocket(onMessage) {
    const token = await this.getAccessToken();
    if (!token) throw new Error("User not authenticated");
    const socket = new WebSocket(`${wsBase()}?token=${token}`);
    socket.onopen = () => console.log("WebSocket connection established.");
    socket.onmessage = (ev) => onMessage(JSON.parse(ev.data));
    socket.onerror = (err) => console.error("WebSocket error:", err);
    socket.onclose = (ev) => {
      if (ev.wasClean) {
        console.log(`WebSocket closed cleanly code=${ev.code} reason=${ev.reason}`);
      } else {
        console.error("WebSocket connection died");
      }
    };
    return socket;
  },
};