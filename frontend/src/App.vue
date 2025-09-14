<template>
  <div id="app" class="bg-background h-screen flex flex-col text-text">
  <Topbar :user="user" @logout="logout" @login="login" @toggle-health="showHealth = !showHealth" />
    <div v-if="backendNotReady" class="bg-yellow-100 text-yellow-800 text-sm py-2 text-center border-b border-yellow-300">
      Backend not ready; retrying checks...
    </div>
    <div v-if="!isAuthenticated" class="w-full flex-1 flex items-center justify-center">
      <!-- Login button now lives in Topbar next to Register -->
    </div>
    <div v-else class="w-full flex-1 flex">
      <Sidebar 
        :chats="chats" 
        :activeChat="activeChat" 
        @selectChat="selectChat"
        @createChat="createChat" 
      />
      <HealthPanel v-if="showHealth" @close="showHealth = false" />
      <!-- Main Content -->
      <div class="flex-grow flex flex-col px-4 py-2 bg-gray-100 min-h-0">
        <div class="flex-1 min-h-0 flex flex-col gap-3">
          <ChatWindow 
            :messages="currentChatMessages" 
            :chatName="activeChat ? activeChat.name : ''" 
            class="w-full max-w-4xl flex-grow min-h-0 mx-auto" 
          />
          <div class="w-full max-w-4xl mx-auto flex-none sticky bottom-0 z-10 bg-gray-100 border-t border-gray-200 shadow-sm">
            <MessageInput @send-message="sendMessage" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import ChatWindow from "./components/ChatWindow.vue";
import MessageInput from "./components/MessageInput.vue";
import Sidebar from "./components/Sidebar.vue";
import Topbar from "./components/Topbar.vue";
import HealthPanel from "./components/HealthPanel.vue";
import { chatService } from "./services/chatService.js";

export default {
  name: "App",
  components: {
    ChatWindow,
    MessageInput,
    Sidebar,
    Topbar,
    HealthPanel,
  },
  provide() {
    return {
      getCurrentUser: () => this.user,
    };
  },
  data() {
    return {
      messages: [],
      socket: null,
      user: null,
      isAuthenticated: false,
      chats: [],
      activeChat: null,
      nextChatId: 4,
  showHealth: false,
      backendNotReady: false,
    };
  },
  computed: {
    currentChatMessages() {
      return this.activeChat ? this.activeChat.messages : [];
    },
  },
  methods: {
    selectChat(chat) {
      this.activeChat = chat;
      // Reset unread count when selecting a chat
      chat.unreadCount = 0;
      // Lazy-load history for this chat if empty
      if (!Array.isArray(chat.messages) || chat.messages.length === 0) {
        chat.messages = [];
        chatService.getMessages(chat.id).then(msgs => {
          chat.messages = Array.isArray(msgs) ? msgs : [];
        }).catch(() => {});
      }
    },
    createChat() {
      const chatName = prompt("Enter chat name:");
      if (!chatName || !chatName.trim()) return;
      chatService.createRoom(chatName.trim()).then((room) => {
        const newChat = { id: room.id, name: room.name, messages: [], unreadCount: 0, lastActivity: new Date() };
        this.chats.push(newChat);
        this.selectChat(newChat);
      }).catch(() => {});
    },
    sendMessage(message) {
      if (!this.activeChat) {
        alert("Please select a chat first!");
        return;
      }
      if (!this.activeChat.messages) {
        this.activeChat.messages = [];
      }
      const messageData = {
        user_id: this.user.profile.name, // Using user's name from profile
        content: message,
        timestamp: new Date(),
        chat_id: this.activeChat.id,
      };
      // Enqueue to outbox so it will retry until ack
      const queued = chatService.enqueueOutgoing(messageData);
      // Add a temporary marker so we could style unsent state later if desired
      messageData.temp_id = queued.temp_id;
      messageData.pending = true;
      // Add message to current chat
      this.activeChat.messages.push(messageData);
      this.activeChat.lastActivity = new Date();
      
      // For now, we'll only send WebSocket messages for the first chat (General Chat)
      // to maintain compatibility with the existing backend
      if (this.activeChat.id === 1 && this.socket) {
        const backendMessage = {
          user_id: this.user.profile.name,
          content: message,
        };
        this.socket.send(JSON.stringify(backendMessage));
      }
    },
    login() {
      chatService.login();
    },
    logout() {
      chatService.logout();
    },
    async initializeApp() {
      try {
        const user = await chatService.getUser();
        this.user = user;
        this.isAuthenticated = !!user;

        if (!this.isAuthenticated) return;

        // Load rooms
  const rooms = await chatService.listRooms().catch(() => []);
  const safeRooms = Array.isArray(rooms) ? rooms : [];
  this.chats = safeRooms.map(r => ({ id: r.id, name: r.name, messages: [], unreadCount: 0, lastActivity: new Date() }));
        // Ensure General exists; if empty, create default
        if (this.chats.length === 0) {
          try {
            const r = await chatService.createRoom('General Chat');
            this.chats.push({ id: r.id, name: r.name, messages: [], unreadCount: 0, lastActivity: new Date() });
          } catch {}
        }
        // Set the first chat as active by default
        this.activeChat = this.chats[0];

        this.socket = await chatService.connectWebSocket((message) => {
          const targetId = message.chat_id || 1;
          const chat = this.chats.find(c => c.id === targetId) || this.chats[0];
          if (chat) {
            if (!Array.isArray(chat.messages)) chat.messages = [];
            chat.messages.push(message);
            chat.lastActivity = new Date();
            if (this.activeChat && this.activeChat.id !== chat.id) {
              chat.unreadCount = (chat.unreadCount || 0) + 1;
            }
          }
        });
        // Start outbox processor; on ack, clear pending marker in the correct chat
        chatService.startOutboxProcessor((item, messageId) => {
          const chat = this.chats.find(c => c.id === item.chat_id);
          if (!chat || !Array.isArray(chat.messages)) return;
          const idx = chat.messages.findIndex(m => m.temp_id === item.temp_id);
          if (idx !== -1) {
            const updated = { ...chat.messages[idx] };
            delete updated.pending;
            updated.message_id = messageId;
            chat.messages.splice(idx, 1, updated);
          }
        });
        const messages = await chatService.getMessages(this.activeChat?.id || 1);
        if (this.chats[0]) {
          this.chats[0].messages = Array.isArray(messages) ? messages : [];
        }
        // start light polling for rooms to catch updates
        setInterval(async () => {
          try {
            const updated = await chatService.listRooms();
            const knownIds = new Set(this.chats.map(c => c.id));
            for (const r of updated) {
              if (!knownIds.has(r.id)) this.chats.push({ id: r.id, name: r.name, messages: [], unreadCount: 0, lastActivity: new Date() });
            }
          } catch {}
        }, 5000);
        // Readiness watcher for banner
        const base = (window.__CHATAPP_CONFIG__ || {}).VUE_APP_API_BASE_URL || process.env.VUE_APP_API_BASE_URL;
        setInterval(async () => {
          try {
            const r = await fetch(`${base}/readyz`);
            this.backendNotReady = !r.ok;
          } catch {
            this.backendNotReady = true;
          }
        }, 5000);
      } catch (e) {
        console.error('Initialization failed:', e);
      }
    },
  },
  async created() {
    if (window.location.pathname === "/callback") {
      await chatService.handleAuthentication();
      window.history.replaceState({}, document.title, "/");
    }
    this.initializeApp();
  },
};
</script>

<style>
#app {
  font-family: Avenir, Helvetica, Arial, sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  text-align: center;
}
</style>
