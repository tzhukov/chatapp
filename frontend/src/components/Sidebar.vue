<template>
  <div class="w-1/4 bg-dark-purple p-4 shadow-md flex flex-col text-white">
    <div class="flex justify-between items-center mb-4">
      <h2 class="text-xl font-bold">Chats</h2>
      <button 
        @click="$emit('createChat')"
        class="bg-primary hover:bg-opacity-80 text-dark-purple font-bold px-3 py-1 rounded-full text-sm transition-all duration-200"
        title="Create new chat"
      >
        +
      </button>
    </div>
    
    <!-- Chat List -->
    <div class="flex-grow overflow-y-auto space-y-2">
      <div
        v-for="chat in sortedChats"
        :key="chat.id"
        @click="$emit('selectChat', chat)"
        class="p-3 rounded-lg cursor-pointer transition-all duration-200 hover:bg-opacity-20 hover:bg-white"
        :class="{
          'bg-primary bg-opacity-30 border-l-4 border-primary': isActiveChat(chat),
          'bg-transparent': !isActiveChat(chat)
        }"
      >
        <div class="flex justify-between items-start">
          <div class="flex-grow min-w-0">
            <h3 class="font-semibold text-sm truncate" :class="{ 'text-primary': isActiveChat(chat) }">
              {{ chat.name }}
            </h3>
            <p class="text-xs opacity-70 truncate mt-1">
              {{ getLastMessagePreview(chat) }}
            </p>
            <p class="text-xs opacity-50 mt-1">
              {{ formatLastActivity(chat.lastActivity) }}
            </p>
          </div>
          <div v-if="chat.unreadCount > 0" class="ml-2">
            <span class="bg-red-500 text-white text-xs font-bold px-2 py-1 rounded-full min-w-[20px] text-center">
              {{ chat.unreadCount > 99 ? '99+' : chat.unreadCount }}
            </span>
          </div>
        </div>
      </div>
    </div>
    
    <!-- No chats message -->
    <div v-if="chats.length === 0" class="text-center text-sm opacity-70 mt-4">
      No chats available
    </div>
  </div>
</template>

<script>
export default {
  name: "Sidebar",
  props: {
    chats: {
      type: Array,
      required: true,
    },
    activeChat: {
      type: Object,
      default: null,
    },
  },
  computed: {
    sortedChats() {
      // Sort chats by last activity (most recent first)
      return [...this.chats].sort((a, b) => new Date(b.lastActivity) - new Date(a.lastActivity));
    },
  },
  methods: {
    isActiveChat(chat) {
      return this.activeChat && this.activeChat.id === chat.id;
    },
    getLastMessagePreview(chat) {
      if (chat.messages && chat.messages.length > 0) {
        const lastMessage = chat.messages[chat.messages.length - 1];
        return `${lastMessage.user_id}: ${lastMessage.content}`;
      }
      return "No messages yet";
    },
    formatLastActivity(date) {
      if (!date) return "";
      
      const now = new Date();
      const diffMs = now - new Date(date);
      const diffMins = Math.floor(diffMs / (1000 * 60));
      const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
      const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));
      
      if (diffMins < 1) return "Just now";
      if (diffMins < 60) return `${diffMins}m ago`;
      if (diffHours < 24) return `${diffHours}h ago`;
      if (diffDays < 7) return `${diffDays}d ago`;
      
      return new Date(date).toLocaleDateString();
    },
  },
};
</script>

<style scoped>
/* Add any specific sidebar styles here if needed, but Tailwind is preferred */
</style>