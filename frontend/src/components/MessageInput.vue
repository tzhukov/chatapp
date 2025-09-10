<template>
  <div class="message-input flex p-4 bg-input-bg rounded-lg shadow-lg border-2 border-border-subtle mt-4">
    <input
      v-model="message"
      @keyup.enter="send"
      placeholder="Type a message..."
      class="flex-grow p-3 rounded-l-lg border-2 border-border-subtle focus:ring-2 focus:ring-primary focus:border-primary outline-none bg-white text-text shadow-sm"
    />
    <button
      @click="send"
      class="bg-primary hover:opacity-90 text-white font-bold py-3 px-6 rounded-r-lg border-2 border-primary transition-opacity duration-200 ml-1 shadow-md"
    >
      Send
    </button>
  </div>
</template>

<script>
export default {
  name: "MessageInput",
  data() {
    return {
      message: "",
    };
  },
  methods: {
    send() {
      if (this.message.trim() !== "") {
        console.log("Sending message from MessageInput:", this.message);
        this.$emit("send-message", this.message);
        this.message = "";
      }
    },
  },
};
</script>

<style scoped>
/* No scoped styles needed, using Tailwind CSS */
</style>