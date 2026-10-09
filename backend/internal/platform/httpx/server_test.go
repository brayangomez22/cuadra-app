package httpx_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/brayangomez22/cuadra-app/backend/internal/platform/httpx"
)

// startServer serves handler on a random port and returns its URL, the cancel
// function that triggers shutdown and a channel with Serve's result.
func startServer(t *testing.T, handler http.Handler, shutdownTimeout time.Duration) (string, context.CancelFunc, <-chan error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	srv := httpx.NewServer(ln.Addr().String(), handler, shutdownTimeout)
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ctx, ln) }()
	return "http://" + ln.Addr().String(), cancel, done
}

func waitFor(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timeout esperando: %s", what)
	}
}

func TestServerGracefulShutdown(t *testing.T) {
	t.Run("el apagado elegante espera las requests en curso", func(t *testing.T) {
		started, release := make(chan struct{}), make(chan struct{})
		handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			close(started)
			<-release
			w.WriteHeader(http.StatusOK)
		})
		url, cancel, done := startServer(t, handler, 5*time.Second)

		respErr := make(chan error, 1)
		respStatus := make(chan int, 1)
		go func() {
			resp, err := http.Get(url)
			if err != nil {
				respErr <- err
				return
			}
			_ = resp.Body.Close()
			respStatus <- resp.StatusCode
		}()
		waitFor(t, started, "que la request llegue al handler")

		cancel()
		select {
		case err := <-done:
			t.Fatalf("Serve terminó con una request en curso: %v", err)
		case <-time.After(100 * time.Millisecond):
		}

		close(release)
		select {
		case status := <-respStatus:
			require.Equal(t, http.StatusOK, status)
		case err := <-respErr:
			t.Fatalf("la request en curso falló: %v", err)
		case <-time.After(2 * time.Second):
			t.Fatal("la request en curso no terminó")
		}
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(2 * time.Second):
			t.Fatal("Serve no terminó después de drenar las requests")
		}
	})

	t.Run("el apagado elegante corta al vencer el timeout", func(t *testing.T) {
		started, release := make(chan struct{}), make(chan struct{})
		t.Cleanup(func() { close(release) })
		handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			close(started)
			<-release
		})
		url, cancel, done := startServer(t, handler, 50*time.Millisecond)
		go func() {
			if resp, err := http.Get(url); err == nil {
				_ = resp.Body.Close()
			}
		}()
		waitFor(t, started, "que la request llegue al handler")

		cancel()
		select {
		case err := <-done:
			require.True(t, errors.Is(err, context.DeadlineExceeded), "error inesperado: %v", err)
		case <-time.After(2 * time.Second):
			t.Fatal("Serve no respetó el timeout de apagado")
		}
	})
}
