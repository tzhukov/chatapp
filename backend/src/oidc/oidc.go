package oidcutil

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"

	"src/config"
	"src/logger"
	"src/metrics"
)

// Init initializes the OIDC provider (with backoff, fallback, optional dial override) and returns the provider & verifier.
func Init(ctx context.Context) (*coreoidc.Provider, *coreoidc.IDTokenVerifier) {
	p := initProviderWithBackoff(ctx, config.DexIssuer)
	verifier := p.Verifier(&coreoidc.Config{ClientID: config.ClientID})
	return p, verifier
}

// VerifyToken verifies a raw token using the provided verifier and validates audience & expiration.
func VerifyToken(ctx context.Context, verifier *coreoidc.IDTokenVerifier, raw string) (*coreoidc.IDToken, error) {
	tok, err := verifier.Verify(ctx, raw)
	if err != nil {
		return nil, err
	}
	var claims struct {
		Aud string `json:"aud"`
		Exp int64  `json:"exp"`
	}
	if err := tok.Claims(&claims); err != nil {
		return nil, err
	}
	if claims.Aud != config.Audience {
		return nil, ErrInvalidAudience{Expected: config.Audience, Got: claims.Aud}
	}
	if time.Now().Unix() > claims.Exp {
		return nil, ErrTokenExpired{}
	}
	return tok, nil
}

// AuthMiddleware returns an HTTP middleware enforcing Bearer token auth.
func AuthMiddleware(verifier *coreoidc.IDTokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}
			token := strings.TrimPrefix(auth, "Bearer ")
			if _, err := VerifyToken(r.Context(), verifier, token); err != nil {
				logger.Error("token verification failed", err)
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Errors

type ErrInvalidAudience struct{ Expected, Got string }

func (e ErrInvalidAudience) Error() string {
	return "invalid audience: expected " + e.Expected + " got " + e.Got
}

type ErrTokenExpired struct{}

func (e ErrTokenExpired) Error() string { return "token expired" }

// Internal backoff + fallback logic (moved from main)
func initProviderWithBackoff(ctx context.Context, issuer string) *coreoidc.Provider {
	var provider *coreoidc.Provider
	var err error
	maxAttempts := 8
	if v, perr := strconv.Atoi(config.DexOIDCMaxAttempts); perr == nil && v > 0 {
		maxAttempts = v
	}

	var httpClient *http.Client
	loopbackDetected := false
	if config.DexIssuerDialOverride != "" {
		logger.Info("using dex issuer dial override", logger.FieldKV("dial", config.DexIssuerDialOverride))
		transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
		baseDialer := &net.Dialer{Timeout: 5 * time.Second}
		transport.DialContext = func(c context.Context, network, _ string) (net.Conn, error) {
			return baseDialer.DialContext(c, network, config.DexIssuerDialOverride)
		}
		httpClient = &http.Client{Transport: transport, Timeout: 10 * time.Second}
	} else {
		// No explicit override; detect if issuer host resolves only to loopback -> switch to internal dial target
		if u, uerr := url.Parse(issuer); uerr == nil {
			if addrs, rerr := net.DefaultResolver.LookupHost(ctx, u.Hostname()); rerr == nil {
				loopOnly := true
				for _, a := range addrs {
					if !isLoopbackIP(a) {
						loopOnly = false
						break
					}
				}
				if loopOnly {
					loopbackDetected = true
					metrics.IncOIDCLoopbackAutoDial()
					internalDial := config.DexIssuerInternalDial
					transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
					baseDialer := &net.Dialer{Timeout: 5 * time.Second}
					transport.DialContext = func(c context.Context, network, _ string) (net.Conn, error) {
						return baseDialer.DialContext(c, network, internalDial)
					}
					httpClient = &http.Client{Transport: transport, Timeout: 10 * time.Second}
					logger.Info("issuer resolves only to loopback; auto internal dial in effect", logger.FieldKV("dial", internalDial), logger.FieldKV("host", u.Hostname()))
				}
			}
		}
	}

	var attemptsUsed uint64
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt == 1 && strings.EqualFold(config.DexOIDCDebug, "true") {
			if u, uerr := url.Parse(issuer); uerr == nil {
				if addrs, rerr := net.DefaultResolver.LookupHost(ctx, u.Hostname()); rerr == nil {
					logger.Info("issuer host dns lookup", logger.FieldKV("host", u.Hostname()), logger.FieldKV("addresses", strings.Join(addrs, ",")), logger.FieldKV("loopback_only", loopbackDetected))
				} else {
					logger.Error("issuer host dns lookup failed", rerr, logger.FieldKV("host", u.Hostname()))
				}
			}
		}
		if httpClient != nil {
			// Attach custom CA if provided and not already configured
			if config.DexCACertFile != "" {
				addCustomCA(httpClient, config.DexCACertFile)
			}
			provider, err = coreoidc.NewProvider(coreoidc.ClientContext(ctx, httpClient), issuer)
		} else {
			if config.DexCACertFile != "" {
				// Build custom client with CA
				c := &http.Client{Timeout: 10 * time.Second}
				addCustomCA(c, config.DexCACertFile)
				provider, err = coreoidc.NewProvider(coreoidc.ClientContext(ctx, c), issuer)
			} else {
				provider, err = coreoidc.NewProvider(ctx, issuer)
			}
		}
		if err == nil {
			attemptsUsed = uint64(attempt)
			logger.Info("oidc provider initialized", logger.FieldKV("issuer", issuer), logger.FieldKV("attempt", attempt))
			metrics.IncOIDCPrimarySuccess(attemptsUsed)
			return provider
		}
		// Detect common misconfiguration: using https issuer while endpoint serves plain http
		if strings.Contains(err.Error(), "server gave HTTP response to HTTPS client") {
			logger.Error("oidc issuer scheme mismatch (https expected but endpoint is http)", err,
				logger.FieldKV("issuer", issuer),
				logger.FieldKV("hint", "Ensure Dex issuer AND DEX_ISSUER_URL both use http:// if ingress has no TLS, or enable TLS on ingress and set both to https://"))
		}
		sleep := time.Duration(math.Min(float64(time.Second*30), float64(time.Second)*math.Pow(2, float64(attempt))))
		logger.Error("oidc provider init failed", err, logger.FieldKV("attempt", attempt), logger.FieldKV("next_sleep", sleep.String()))
		select {
		case <-time.After(sleep):
			continue
		case <-ctx.Done():
			logger.Error("context canceled during oidc init", ctx.Err())
			log.Fatalf("Failed to initialize OIDC provider: %v", err)
		}
	}

	if strings.EqualFold(config.DexOIDCFallbackEnabled, "true") && issuer != config.InternalDexIssuer {
		metrics.IncOIDCFallbackActivated()
		logger.Error("primary issuer failed, attempting internal fallback", err, logger.FieldKV("primary_issuer", issuer), logger.FieldKV("fallback_issuer", config.InternalDexIssuer))
		fallbackIssuer := config.InternalDexIssuer
		for attempt := 1; attempt <= 4; attempt++ {
			provider, ferr := coreoidc.NewProvider(ctx, fallbackIssuer)
			if ferr == nil {
				logger.Info("oidc provider initialized via fallback", logger.FieldKV("issuer", fallbackIssuer), logger.FieldKV("attempt", attempt))
				metrics.IncOIDCFallbackSuccess(uint64(attempt))
				return provider
			}
			sleep := time.Duration(500*time.Millisecond) * time.Duration(attempt)
			logger.Error("fallback issuer init failed", ferr, logger.FieldKV("attempt", attempt), logger.FieldKV("next_sleep", sleep.String()))
			select {
			case <-time.After(sleep):
				continue
			case <-ctx.Done():
				logger.Error("context canceled during fallback oidc init", ctx.Err())
				log.Fatalf("Failed to initialize OIDC provider (fallback): %v", ferr)
			}
		}
		log.Fatalf("Failed to initialize OIDC provider after fallback attempts: %v", err)
	}

	metrics.IncOIDCInitFailure(uint64(maxAttempts))
	log.Fatalf("Failed to initialize OIDC provider after retries: %v", err)
	return nil
}

// isLoopbackIP determines if a string IP is loopback (IPv4 127.0.0.0/8 or IPv6 ::1)
func isLoopbackIP(ipStr string) bool {
	parsed := net.ParseIP(ipStr)
	if parsed == nil {
		return false
	}
	return parsed.IsLoopback()
}

// addCustomCA loads a PEM bundle from path and appends to the http client's transport RootCAs.
func addCustomCA(c *http.Client, path string) {
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Error("failed to read custom CA file", err, logger.FieldKV("path", path))
		return
	}
	p := x509.NewCertPool()
	if ok := p.AppendCertsFromPEM(data); !ok {
		logger.Error("no certs appended from custom CA file", nil, logger.FieldKV("path", path))
		return
	}
	tr, _ := c.Transport.(*http.Transport)
	if tr == nil {
		tr = &http.Transport{Proxy: http.ProxyFromEnvironment}
	}
	if tr.TLSClientConfig == nil {
		tr.TLSClientConfig = &tls.Config{RootCAs: p}
	} else {
		// Replace existing roots (simpler for controlled dev environment)
		tr.TLSClientConfig.RootCAs = p
	}
	c.Transport = tr
	logger.Info("custom CA trust added for OIDC", logger.FieldKV("path", path))
}
