<template>
  <div class="fixed top-16 right-4 w-96 bg-white text-gray-900 border border-gray-300 rounded shadow-xl z-50">
    <div class="flex items-center justify-between px-4 py-2 border-b">
      <h3 class="font-semibold">API Health</h3>
      <button @click="$emit('close')" class="text-gray-500 hover:text-gray-700">✕</button>
    </div>
    <div class="p-4 space-y-3">
      <div>
        <div class="text-sm text-gray-600">Healthz</div>
        <div :class="statusClass(healthz)">{{ healthzText }}</div>
      </div>
      <div>
        <div class="text-sm text-gray-600">Readyz</div>
        <div :class="statusClass(readyz)">{{ readyzText }}</div>
      </div>
      <div class="flex gap-2 pt-2">
        <button @click="refresh" class="bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-3 py-1 rounded">Refresh</button>
        <a :href="metricsUrl" target="_blank" class="text-blue-600 text-sm underline">Open Metrics</a>
      </div>
    </div>
  </div>
</template>

<script>
import { chatService } from "../services/chatService";
export default {
  name: "HealthPanel",
  data() {
  return { healthz: null, readyz: null, loading: false, error: null, _timer: null, _last: null };
  },
  computed: {
    healthzText() {
      if (this.healthz === null) return "—";
      return this.healthz.ok ? "OK" : `ERROR (${this.healthz.status})`;
    },
    readyzText() {
      if (this.readyz === null) return "—";
      return this.readyz.ok ? "READY" : `NOT READY (${this.readyz.status})`;
    },
    metricsUrl() {
      const base = (window.__CHATAPP_CONFIG__ || {}).VUE_APP_API_BASE_URL || process.env.VUE_APP_API_BASE_URL;
      return `${base}/metrics`;
    }
  },
  methods: {
    statusClass(s) { return s && s.ok ? 'text-green-700 font-semibold' : 'text-red-700 font-semibold'; },
    async refresh() {
      this.loading = true; this.error = null;
      try {
        const base = (window.__CHATAPP_CONFIG__ || {}).VUE_APP_API_BASE_URL || process.env.VUE_APP_API_BASE_URL;
        const h = await fetch(`${base}/healthz`);
        const r = await fetch(`${base}/readyz`);
        this.healthz = { ok: h.ok, status: h.status };
        this.readyz = { ok: r.ok, status: r.status };
        this._last = new Date();
      } catch (e) {
        this.error = e.message || String(e);
      } finally { this.loading = false; }
    }
  },
  mounted() {
    this.refresh();
    this._timer = window.setInterval(() => this.refresh(), 5000);
  },
  beforeUnmount() {
    if (this._timer) { clearInterval(this._timer); this._timer = null; }
  }
}
</script>

<style scoped>
</style>
