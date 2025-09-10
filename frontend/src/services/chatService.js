import { UserManager, WebStorageStateStore } from "oidc-client";


const config = {
  authority: process.env.VUE_APP_DEX_ISSUER_URL,
  client_id: process.env.VUE_APP_DEX_CLIENT_ID,
  redirect_uri: process.env.VUE_APP_DEX_REDIRECT_URI,
  response_type: "code",
  scope: process.env.VUE_APP_DEX_SCOPES,
  post_logout_redirect_uri: process.env.VUE_APP_DEX_REDIRECT_URI,
  userStore: new WebStorageStateStore({ store: window.localStorage }),
};

const userManager = new UserManager(config);

export const chatService = {
  userManager,

  async login() {
    return userManager.signinRedirect();
  },

  async logout() {
    return userManager.signoutRedirect();
  },

  async handleAuthentication() {
    return userManager.signinRedirectCallback();
  },

  async getUser() {
    return userManager.getUser();
  },

  async getAccessToken() {
    const user = await userManager.getUser();
    if (!user) return null;
    // If token expired, try to renew using refresh token
    if (user.expired && user.refresh_token) {
      try {
        const renewedUser = await userManager.signinSilent();
        return renewedUser.access_token;
      } catch (err) {
        await userManager.signoutRedirect();
        throw new Error("Session expired. Please log in again.");
      }
    }
    return user.access_token;
  },

  async getMessages() {
    const token = await this.getAccessToken();
    if (!token) {
      throw new Error("User not authenticated");
    }
    const response = await fetch(`${process.env.VUE_APP_API_BASE_URL}/messages`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });
    if (response.status === 401) {
      await userManager.signoutRedirect();
      throw new Error("Session expired. Please log in again.");
    }
    if (!response.ok) {
      throw new Error("Failed to fetch messages");
    }
    return response.json();
  },

  async sendMessage(message) {
    const token = await this.getAccessToken();
    if (!token) {
      throw new Error("User not authenticated");
    }
    const response = await fetch(`${process.env.VUE_APP_API_BASE_URL}/messages`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(message),
    });
    if (response.status === 401) {
      await userManager.signoutRedirect();
      throw new Error("Session expired. Please log in again.");
    }
    if (!response.ok) {
      throw new Error("Failed to send message");
    }
    return response.json();
  },

  async connectWebSocket(onMessage) {
    const token = await this.getAccessToken();
    if (!token) {
      throw new Error("User not authenticated");
    }
    const socket = new WebSocket(`${process.env.VUE_APP_WS_URL}?token=${token}`);
    socket.onopen = () => {
      console.log("WebSocket connection established.");
    };
    socket.onmessage = (event) => {
      onMessage(JSON.parse(event.data));
    };
    socket.onerror = (error) => {
      console.error("WebSocket error:", error);
    };
    socket.onclose = (event) => {
      if (event.wasClean) {
        console.log(`WebSocket connection closed cleanly, code=${event.code} reason=${event.reason}`);
      } else {
        console.error("WebSocket connection died");
      }
    };
    return socket;
  },
};