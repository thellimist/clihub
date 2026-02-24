package oauth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
)

// CallbackResult holds the result from the OAuth callback.
type CallbackResult struct {
	Code  string
	State string
	Err   error
}

// CallbackServer listens on 127.0.0.1:0 for the OAuth redirect callback.
type CallbackServer struct {
	Port     int
	listener net.Listener
	server   *http.Server
	result   chan CallbackResult
	once     sync.Once
}

// Start begins listening on a random port. Call Close() when done.
func (s *CallbackServer) Start() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("start callback server: %w", err)
	}
	s.listener = ln
	s.Port = ln.Addr().(*net.TCPAddr).Port
	s.result = make(chan CallbackResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)
	s.server = &http.Server{Handler: mux}

	go s.server.Serve(ln)
	return nil
}

// RedirectURI returns the full callback URL.
func (s *CallbackServer) RedirectURI() string {
	return fmt.Sprintf("http://127.0.0.1:%d/callback", s.Port)
}

// WaitForCallback blocks until the callback is received or ctx is cancelled.
// Validates the state parameter matches expectedState.
func (s *CallbackServer) WaitForCallback(ctx context.Context, expectedState string) (string, error) {
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("timed out waiting for authorization callback")
	case r := <-s.result:
		if r.Err != nil {
			return "", r.Err
		}
		if expectedState != "" && r.State != expectedState {
			return "", fmt.Errorf("OAuth state mismatch (possible CSRF)")
		}
		return r.Code, nil
	}
}

// Close shuts down the callback server.
func (s *CallbackServer) Close() {
	s.once.Do(func() {
		if s.server != nil {
			s.server.Close()
		}
	})
}

func (s *CallbackServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	oauthErr := r.URL.Query().Get("error")

	if oauthErr != "" {
		desc := r.URL.Query().Get("error_description")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "<html><body><h1>Authorization failed</h1><p>%s: %s</p></body></html>", oauthErr, desc)
		s.result <- CallbackResult{Err: fmt.Errorf("OAuth error: %s — %s", oauthErr, desc)}
		return
	}

	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "<html><body><h1>Missing authorization code</h1></body></html>")
		s.result <- CallbackResult{Err: fmt.Errorf("callback missing authorization code")}
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "<html><body><h1>Authentication successful!</h1><p>You can close this window and return to the terminal.</p></body></html>")
	s.result <- CallbackResult{Code: code, State: state}
}
