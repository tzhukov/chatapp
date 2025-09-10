import { UserManager } from "oidc-client";


const config = {
  authority: import.meta.env.VITE_DEX_ISSUER_URL,
  client_id: import.meta.env.VITE_DEX_CLIENT_ID,
  redirect_uri: import.meta.env.VITE_DEX_REDIRECT_URI,
  response_type: "code",
  scope: import.meta.env.VITE_DEX_SCOPES,
  post_logout_redirect_uri: import.meta.env.VITE_DEX_REDIRECT_URI,
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

  async getMessages() {
    const user = await userManager.getUser();
    const token = user ? user.access_token : null;
    if (!token) {
      throw new Error("User not authenticated");
    }

  const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/messages`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    });

    if (!response.ok) {
      throw new Error("Failed to fetch messages");
    }

    return response.json();
  },

  async sendMessage(message) {
    const user = await userManager.getUser();
    const token = user ? user.access_token : null;
    if (!token) {
      throw new Error("User not authenticated");
    }

  const response = await fetch(`${import.meta.env.VITE_API_BASE_URL}/messages`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(message),
    });

    if (!response.ok) {
      throw new Error("Failed to send message");
    }

    return response.json();
  },

  async connectWebSocket(onMessage) {
    const user = await userManager.getUser();
    const token = user ? user.access_token : null;
    if (!token) {
      throw new Error("User not authenticated");
    }

  const socket = new WebSocket(`${import.meta.env.VITE_WS_URL}?token=${token}`);

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