<template>
  <div class="w-full bg-topbar-bg p-4 flex items-center justify-between shadow-lg border-b-2 border-border-subtle text-white">
    <div class="text-lg font-semibold">Chat App</div>
    <div v-if="user">
      <span class="mr-4 font-medium">Welcome, {{ user.profile.name }}</span>
      <button @click="$emit('logout')" class="bg-red-500 hover:bg-red-700 text-white font-bold py-2 px-4 rounded shadow-md transition-colors duration-200">
        Logout
      </button>
    </div>
    <div v-else>
      <button @click="goRegister" class="mr-2 bg-green-500 hover:bg-green-700 text-white font-bold py-2 px-4 rounded shadow-md transition-colors duration-200">Register</button>
      <button @click="$emit('login')" class="bg-blue-500 hover:bg-blue-700 text-white font-bold py-2 px-4 rounded shadow-md transition-colors duration-200">Login</button>
    </div>
  </div>
</template>

<script>
export default {
  name: "Topbar",
  props: {
    user: {
      type: Object,
      default: null,
    },
  },
  emits: ['logout','login'],
  methods: {
    goRegister() {
      window.location.href = 'https://ingress.local/register';
    }
  }
};
</script>

<style scoped>
/* Add any specific topbar styles here if needed, but Tailwind is preferred */
</style>
