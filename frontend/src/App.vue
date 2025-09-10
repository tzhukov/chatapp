<template>
  <div id="app" class="bg-background h-screen flex text-text">
    <div v-if="!isAuthenticated" class="w-full h-full flex items-center justify-center">
      <button @click="login" class="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded">
        Login
      </button>
    </div>
    <div v-else class="w-full h-full flex">
      <Sidebar />
      <!-- Main Content -->
      <div class="flex-grow flex flex-col px-4">
        <Topbar :user="user" @logout="logout" />
        <div class="flex-grow flex flex-col">
          <ChatWindow :messages="messages" class="w-full max-w-2xl flex-grow" />
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
  data() {
    return {
      messages: [],
      socket: null,
      user: null,
      isAuthenticated: false,
    };
  },
  methods: {
    sendMessage(message) {
      const messageData = {
        user_id: this.user.profile.name, // Using user's name from profile
        content: message,
      };
      this.socket.send(JSON.stringify(messageData));
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
        this.socket = await chatService.connectWebSocket((message) => {
          this.messages.push(message);
        });
        const messages = await chatService.getMessages();
        this.messages = messages;
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
