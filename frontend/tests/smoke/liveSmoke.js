#!/usr/bin/env node
// Simple live smoke test against a running Tilt environment.
// Assumptions:
// - Frontend served at https://ingress.local/
// - Dex at https://ingress.local/dex
// - Backend API at https://ingress.local/api
// NOTE: This does NOT perform full OIDC auth (browser flow). It validates
// that the main page and runtime config load, and that unauthenticated
// API calls are rejected with 401.

const https = require('https');

function get(url) {
  return new Promise((resolve, reject) => {
    https.get(url, { rejectUnauthorized: false }, (res) => {
      let data = '';
      res.on('data', chunk => data += chunk);
      res.on('end', () => resolve({ status: res.statusCode, body: data }));
    }).on('error', reject);
  });
}

(async () => {
  const results = { ok: true, steps: [] };
  try {
    const index = await get('https://ingress.local/');
    results.steps.push({ step: 'GET /', status: index.status, containsAppDiv: index.body.includes('<div id="app"></div>') });

    const config = await get('https://ingress.local/config.js');
    results.steps.push({ step: 'GET /config.js', status: config.status, hasIssuer: /VUE_APP_DEX_ISSUER_URL/.test(config.body) });

    const api = await get('https://ingress.local/api/messages');
    results.steps.push({ step: 'GET /api/messages (unauth)', status: api.status });

    console.log(JSON.stringify(results, null, 2));
    if (api.status !== 401) {
      process.exitCode = 1;
    }
  } catch (e) {
    console.error('Smoke test failed:', e);
    process.exit(2);
  }
})();
