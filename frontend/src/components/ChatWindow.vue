<template>
  <div class="chat-window bg-secondary p-4 rounded-lg shadow-md overflow-y-auto h-96">
    <!-- Chat header -->
    <div v-if="chatName" class="border-b border-gray-300 pb-2 mb-4">
      <h3 class="text-lg font-semibold text-text">{{ chatName }}</h3>
    </div>
    
    <!-- Messages -->
    <div v-if="messages.length > 0" class="space-y-3">
      <div
        v-for="(message, index) in messages"
        :key="message.message_id || index"
        class="message p-3 rounded-lg max-w-[80%]"
        :class="{
          'bg-primary text-white ml-auto': isCurrentUserMessage(message),
          'bg-accent text-white': !isCurrentUserMessage(message),
        }"
      >
        <span class="font-bold block text-sm opacity-80">{{ message.user_id }}:</span>
        <span class="block text-base">{{ message.content }}</span>
        <span v-if="message.timestamp" class="block text-xs opacity-60 mt-1">
          {{ formatTimestamp(message.timestamp) }}
        </span>
      </div>
    </div>
    
    <!-- Empty state -->
    <div v-else class="flex items-center justify-center h-full text-center">
      <div class="text-text opacity-60">
        <p class="text-lg mb-2">No messages yet</p>
        <p class="text-sm">Start the conversation!</p>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: "ChatWindow",
  props: {
    messages: {
      type: Array,
      required: true,
    },
    chatName: {
      type: String,
      default: "",
    },
  },
  inject: ['getCurrentUser'], // To get current user for message comparison
  methods: {
    isCurrentUserMessage(message) {
      const currentUser = this.getCurrentUser?.() || null;
      return currentUser && message.user_id === currentUser.profile.name;
    },
    formatTimestamp(timestamp) {
      if (!timestamp) return "";
      const date = new Date(timestamp);
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    },
  },
  updated() {
    console.log("ChatWindow component updated with new messages:", this.messages);
    const chatWindow = this.$el;
    chatWindow.scrollTop = chatWindow.scrollHeight;
  },
};
</script>

<style scoped>
/* No scoped styles needed, using Tailwind CSS */
</style>