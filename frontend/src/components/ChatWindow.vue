<template>
  <div class="chat-window bg-chat-bg p-6 rounded-lg shadow-lg border-2 border-border-subtle overflow-y-auto min-h-0">
    <!-- Chat header -->
    <div v-if="chatName" class="border-b-2 border-border-subtle pb-3 mb-4">
      <h3 class="text-lg font-semibold text-white">{{ chatName }}</h3>
    </div>
    
    <!-- Messages -->
    <div v-if="messages.length > 0" class="space-y-2">
      <div
        v-for="(message, index) in messages"
        :key="message.message_id || index"
        class="flex w-full"
      >
        <div
          class="px-3 py-2 rounded-lg max-w-[80%] shadow-sm text-sm flex items-center gap-2"
          :class="[
            isCurrentUserMessage(message) ? 'bg-primary text-white ml-auto' : 'bg-accent text-white mr-auto',
            message.pending ? 'opacity-70' : ''
          ]"
        >
          <span class="opacity-70">{{ formatTimestamp(message.timestamp) }}</span>
          <span class="font-semibold">{{ message.user_id }}:</span>
          <span class="whitespace-pre-line">{{ message.content }}</span>
          <span v-if="message.pending" class="italic opacity-70">(sending…)</span>
        </div>
      </div>
    </div>
    
    <!-- Empty state -->
    <div v-else class="flex items-center justify-center h-full text-center">
      <div class="text-white opacity-80">
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