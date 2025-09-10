<template>
  <div class="chat-window bg-secondary p-4 rounded-lg shadow-md overflow-y-auto h-96">
    <div
      v-for="(message, index) in messages"
      :key="message.message_id || index"
      class="message mb-3 p-3 rounded-lg max-w-[80%]"
      :class="{
        'bg-primary text-white ml-auto': message.user_id === 'user1', // Assuming 'user1' is the current user
        'bg-accent text-white': message.user_id !== 'user1',
      }"
    >
      <span class="font-bold block text-sm opacity-80">{{ message.user_id }}:</span>
      <span class="block text-base">{{ message.content }}</span>
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