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
  // Outbox queue (durable via localStorage)
  _outboxKey: 'chat_outbox_v1',
  _processorTimer: null,
  _processing: false,

  _loadOutbox() {
    try {
      const raw = window.localStorage.getItem(this._outboxKey);
      const parsed = raw ? JSON.parse(raw) : [];
      return Array.isArray(parsed) ? parsed : [];
    } catch { return []; }
  },
  _saveOutbox(list) {
    try { window.localStorage.setItem(this._outboxKey, JSON.stringify(list || [])); } catch {}
  },
  getOutbox() { return this._loadOutbox(); },
  enqueueOutgoing({ chat_id, user_id, content, timestamp }) {
    const item = {
      temp_id: `${Date.now()}-${Math.random().toString(36).slice(2,8)}`,
      chat_id, user_id, content,
      timestamp: timestamp || new Date().toISOString(),
      attempts: 0,
      nextAttemptAt: Date.now(),
    };
    const q = this._loadOutbox();
    q.push(item);
    this._saveOutbox(q);
    return item;
  },
  async _trySendOne(item) {
    const token = await this.getIdToken();
    const body = { chat_id: item.chat_id, user_id: item.user_id, content: item.content, timestamp: new Date(item.timestamp) };
    const resp = await fetch(`${apiBase()}/messages`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify(body),
    });
    if (!resp.ok) {
      const text = await resp.text().catch(() => '');
      throw new Error(`send failed ${resp.status} ${text}`);
    }
    const data = await resp.json().catch(() => ({}));
    return data && data.message_id ? data.message_id : null;
  },
  startOutboxProcessor(onAck) {
    if (this._processorTimer) return;
    const tick = async () => {
      if (this._processing) return; this._processing = true;
      try {
        let q = this._loadOutbox();
        const now = Date.now();
        const keep = [];
        for (const item of q) {
          if (item.nextAttemptAt && now < item.nextAttemptAt) { keep.push(item); continue; }
          try {
            const messageId = await this._trySendOne(item);
            if (onAck && messageId) onAck(item, messageId);
            // drop on success
          } catch (e) {
            // backoff: 1s,2s,4s up to 30s
            const attempts = (item.attempts || 0) + 1;
            const delay = Math.min(30000, 1000 * Math.pow(2, attempts - 1));
            keep.push({ ...item, attempts, nextAttemptAt: Date.now() + delay });
          }
        }
        this._saveOutbox(keep);
      } finally { this._processing = false; }
    };
    this._processorTimer = window.setInterval(tick, 1000);
    // also run once immediately
    tick();
  },
  stopOutboxProcessor() {
    if (this._processorTimer) { clearInterval(this._processorTimer); this._processorTimer = null; }
  },

  async login() {
    return getUserManager().signinRedirect();
  },

  async logout() {
    // Try RP-initiated logout; fallback to local sign-out if OP lacks end_session_endpoint
    const um = getUserManager();
    try {
      const meta = await um.metadataService.getMetadata();
      if (meta && (meta.end_session_endpoint || meta.revocation_endpoint)) {
        return um.signoutRedirect();
      }
    } catch (_) { /* ignore and fallback */ }
    try { await um.removeUser(); } catch (_) {}
    const cfg = resolveConfig();
    if (cfg && cfg.redirect_uri) {
      window.location.href = cfg.redirect_uri;
    }
  },

  async handleAuthentication() {
    return getUserManager().signinRedirectCallback();
  },

  async getUser() {
    return getUserManager().getUser();
  },

  async getIdToken() {
    const user = await getUserManager().getUser();
    if (!user) return null;
    if (user.expired && user.refresh_token) {
      try {
        const renewed = await getUserManager().signinSilent();
        return renewed.id_token;
      } catch (e) {
        await this.logout();
        throw new Error("Session expired. Please log in again.");
      }
    }
    return user.id_token;
  },

  async getMessages(chatId) {
    const token = await this.getIdToken();
    if (!token) throw new Error("User not authenticated");
    const base = apiBase();
    if (!base) throw new Error("API base URL not configured (VUE_APP_API_BASE_URL)");
    const q = chatId ? `?chat_id=${encodeURIComponent(chatId)}` : '';
    const response = await fetch(`${base}/messages${q}` , {
      headers: { Authorization: `Bearer ${token}` },
    });
    if (response.status === 401) {
      await this.logout();
      throw new Error("Session expired. Please log in again.");
    }
    if (!response.ok) throw new Error("Failed to fetch messages");
    return response.json();
  },

  async getAccessToken() {
    const user = await getUserManager().getUser();
    if (!user) return null;
    return user.access_token || user.id_token || null;
  },

  async sendMessage(message) {
    const token = await this.getIdToken();
    if (!token) throw new Error("User not authenticated");
    const response = await fetch(`${apiBase()}/messages`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${token}` },
      body: JSON.stringify(message),
    });
    if (response.status === 401) {
      await this.logout();
      throw new Error("Session expired. Please log in again.");
    }
    if (!response.ok) throw new Error("Failed to send message");
    return response.json();
  },

  async connectWebSocket(onMessage) {
    const token = await this.getIdToken();
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

  // Rooms API
  async listRooms() {
    const token = await this.getIdToken();
    if (!token) throw new Error("User not authenticated");
    const base = apiBase();
    const resp = await fetch(`${base}/rooms`, { headers: { Authorization: `Bearer ${token}` } });
    if (!resp.ok) {
      const msg = `Failed to fetch rooms (status ${resp.status})`;
      throw new Error(msg);
    }
    return resp.json();
  },
  async createRoom(name) {
    const token = await this.getIdToken();
    if (!token) throw new Error("User not authenticated");
    const base = apiBase();
    const resp = await fetch(`${base}/rooms`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify({ name })
    });
    if (!resp.ok) throw new Error("Failed to create room");
    return resp.json();
  },
};