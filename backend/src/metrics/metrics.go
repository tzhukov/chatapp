package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

// Prometheus-style counters (uint64 via atomic)
var (
	oidcPrimarySuccess    atomic.Uint64
	oidcFallbackSuccess   atomic.Uint64
	oidcInitFailure       atomic.Uint64
	oidcLoopbackAutoDial  atomic.Uint64
	oidcFallbackActivated atomic.Uint64
	oidcLastInitAttempts  atomic.Uint64 // gauge semantics
	wsConnections         atomic.Uint64
	msgIngestedTotal      atomic.Uint64
	msgBroadcastTotal     atomic.Uint64
)

// Increment helpers
func IncOIDCPrimarySuccess(attempts uint64) {
	oidcPrimarySuccess.Add(1)
	oidcLastInitAttempts.Store(attempts)
}
func IncOIDCFallbackSuccess(attempts uint64) {
	oidcFallbackSuccess.Add(1)
	oidcLastInitAttempts.Store(attempts)
}
func IncOIDCInitFailure(attempts uint64) {
	oidcInitFailure.Add(1)
	oidcLastInitAttempts.Store(attempts)
}
func IncOIDCLoopbackAutoDial()  { oidcLoopbackAutoDial.Add(1) }
func IncOIDCFallbackActivated() { oidcFallbackActivated.Add(1) }

// WebSocket metrics
func IncWSConnections() { wsConnections.Add(1) }
func DecWSConnections() { wsConnections.Add(^uint64(0)) } // atomic decrement
func IncMsgIngested()   { msgIngestedTotal.Add(1) }
func IncMsgBroadcast()  { msgBroadcastTotal.Add(1) }

// Handler exposes metrics in a minimal Prometheus exposition format.
func Handler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP chatapp_oidc_provider_init_success_total OIDC provider successful initializations\n")
	fmt.Fprintf(w, "# TYPE chatapp_oidc_provider_init_success_total counter\n")
	fmt.Fprintf(w, "chatapp_oidc_provider_init_success_total{mode=\"primary\"} %d\n", oidcPrimarySuccess.Load())
	fmt.Fprintf(w, "chatapp_oidc_provider_init_success_total{mode=\"fallback\"} %d\n", oidcFallbackSuccess.Load())

	fmt.Fprintf(w, "# HELP chatapp_oidc_provider_init_failure_total OIDC provider initialization failures (process exiting)\n")
	fmt.Fprintf(w, "# TYPE chatapp_oidc_provider_init_failure_total counter\n")
	fmt.Fprintf(w, "chatapp_oidc_provider_init_failure_total %d\n", oidcInitFailure.Load())

	fmt.Fprintf(w, "# HELP chatapp_oidc_loopback_auto_dial_total Loopback-only DNS detections triggering internal dial\n")
	fmt.Fprintf(w, "# TYPE chatapp_oidc_loopback_auto_dial_total counter\n")
	fmt.Fprintf(w, "chatapp_oidc_loopback_auto_dial_total %d\n", oidcLoopbackAutoDial.Load())

	fmt.Fprintf(w, "# HELP chatapp_oidc_fallback_activated_total Times fallback issuer path was engaged\n")
	fmt.Fprintf(w, "# TYPE chatapp_oidc_fallback_activated_total counter\n")
	fmt.Fprintf(w, "chatapp_oidc_fallback_activated_total %d\n", oidcFallbackActivated.Load())

	fmt.Fprintf(w, "# HELP chatapp_oidc_last_init_attempts Attempts used in the most recent successful/failed init\n")
	fmt.Fprintf(w, "# TYPE chatapp_oidc_last_init_attempts gauge\n")
	fmt.Fprintf(w, "chatapp_oidc_last_init_attempts %d\n", oidcLastInitAttempts.Load())

	fmt.Fprintf(w, "# HELP chatapp_ws_connections Current websocket connections\n")
	fmt.Fprintf(w, "# TYPE chatapp_ws_connections gauge\n")
	fmt.Fprintf(w, "chatapp_ws_connections %d\n", wsConnections.Load())

	fmt.Fprintf(w, "# HELP chatapp_messages_ingested_total Messages accepted and enqueued\n")
	fmt.Fprintf(w, "# TYPE chatapp_messages_ingested_total counter\n")
	fmt.Fprintf(w, "chatapp_messages_ingested_total %d\n", msgIngestedTotal.Load())

	fmt.Fprintf(w, "# HELP chatapp_messages_broadcast_total Messages broadcast to websocket clients\n")
	fmt.Fprintf(w, "# TYPE chatapp_messages_broadcast_total counter\n")
	fmt.Fprintf(w, "chatapp_messages_broadcast_total %d\n", msgBroadcastTotal.Load())
}
