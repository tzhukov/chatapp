<template>
  <div id="app" class="bg-background h-screen flex text-text">
    <div v-if="!isAuthenticated" class="w-full h-full flex items-center justify-center">
      <button @click="login" class="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded">
        Login
      </button>
    </div>
    <div v-else class="w-full h-full flex">
      <Sidebar 
        :chats="chats" 
        :activeChat="activeChat" 
        @selectChat="selectChat"
        @createChat="createChat" 
      />
      <!-- Main Content -->
      <div class="flex-grow flex flex-col px-4">
        <Topbar :user="user" @logout="logout" />
        <div class="flex-grow flex flex-col">
          <ChatWindow 
            :messages="currentChatMessages" 
            :chatName="activeChat ? activeChat.name : ''" 
            class="w-full max-w-2xl flex-grow" 
          />
          <MessageInput @send-message="sendMessage" class="w-full max-w-2xl mt-4" />
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
import { chatService } from "./services/chatService.js";

export default {
  name: "App",
  components: {
    ChatWindow,
    MessageInput,
    Sidebar,
    Topbar,
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
      chats: [
        {
          id: 1,
          name: "General Chat",
          messages: [],
          unreadCount: 0,
          lastActivity: new Date(),
        },
        {
          id: 2,
          name: "Random",
          messages: [],
          unreadCount: 0,
          lastActivity: new Date(Date.now() - 60 * 60 * 1000), // 1 hour ago
        },
        {
          id: 3,
          name: "Project Discussion",
          messages: [],
          unreadCount: 0,
          lastActivity: new Date(Date.now() - 2 * 60 * 60 * 1000), // 2 hours ago
        },
      ],
      activeChat: null,
      nextChatId: 4,
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
    },
    createChat() {
      const chatName = prompt("Enter chat name:");
      if (chatName && chatName.trim()) {
        const newChat = {
          id: this.nextChatId++,
          name: chatName.trim(),
          messages: [],
          unreadCount: 0,
          lastActivity: new Date(),
        };
        this.chats.push(newChat);
        this.selectChat(newChat);
      }
    },
    sendMessage(message) {
      if (!this.activeChat) {
        alert("Please select a chat first!");
        return;
      }
      
      const messageData = {
        user_id: this.user.profile.name, // Using user's name from profile
        content: message,
        timestamp: new Date(),
        chat_id: this.activeChat.id,
      };
      
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
      const user = await chatService.getUser();
      this.user = user;
      this.isAuthenticated = !!user;

      if (this.isAuthenticated) {
        // Set the first chat (General Chat) as active by default
        this.activeChat = this.chats[0];
        
        this.socket = await chatService.connectWebSocket((message) => {
          // Add incoming messages to the General Chat (first chat)
          if (this.chats[0]) {
            this.chats[0].messages.push(message);
            this.chats[0].lastActivity = new Date();
            
            // If not currently viewing General Chat, increment unread count
            if (this.activeChat && this.activeChat.id !== 1) {
              this.chats[0].unreadCount++;
            }
          }
        });
        
        const messages = await chatService.getMessages();
        // Load existing messages into the General Chat
        if (this.chats[0]) {
          this.chats[0].messages = messages;
        }
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
