package httpx

import (
	"context"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

type clientIPKey struct{}

// ClientIP stores the client's IP in the request context (ClientIPFrom):
// the connection's, or, when the connection comes from one of the trusted
// proxies (the cloud's ingress), the right-most address of X-Forwarded-For
// that is not a trusted proxy. A client can prepend any address to the
// header, but not the one the ingress appended, so the rate limits count
// the real client and not the ingress (P05). Without trusted proxies the
// header is ignored.
func ClientIP(trusted []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r, trusted)
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey{}, ip)))
		})
	}
}

// ClientIPFrom returns the IP ClientIP stored; empty without it.
func ClientIPFrom(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)
	return ip
}

func clientIP(r *http.Request, trusted []netip.Prefix) string {
	remote := r.RemoteAddr
	if host, _, err := net.SplitHostPort(remote); err == nil {
		remote = host
	}
	addr, err := netip.ParseAddr(remote)
	if err != nil || !isTrusted(addr, trusted) {
		return remote
	}
	hops := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
	for i := len(hops) - 1; i >= 0; i-- {
		hop, err := netip.ParseAddr(strings.TrimSpace(hops[i]))
		if err != nil {
			return remote
		}
		if !isTrusted(hop, trusted) {
			return hop.Unmap().String()
		}
	}
	return remote
}

func isTrusted(addr netip.Addr, trusted []netip.Prefix) bool {
	addr = addr.Unmap()
	for _, p := range trusted {
		if p.Contains(addr) {
			return true
		}
	}
	return false
}
