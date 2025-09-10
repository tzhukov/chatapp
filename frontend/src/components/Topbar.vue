<template>
  <div class="w-full bg-topbar-bg p-4 flex justify-end items-center shadow-lg border-b-2 border-border-subtle text-white">
    <div v-if="user">
      <span class="mr-4 font-medium">Welcome, {{ user.profile.name }}</span>
      <button @click="$emit('logout')" class="bg-red-500 hover:bg-red-700 text-white font-bold py-2 px-4 rounded shadow-md transition-colors duration-200">
        Logout
      </button>
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
};
</script>

<style scoped>
/* Add any specific topbar styles here if needed, but Tailwind is preferred */
</style>
