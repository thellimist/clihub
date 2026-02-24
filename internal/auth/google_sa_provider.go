package auth

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleSAProvider provides Google Service Account authentication
// using JWT signed with a service account key file.
type GoogleSAProvider struct {
	// KeyFile is the path to the Google service account JSON key file.
	KeyFile string
	// Scopes are the OAuth2 scopes to request.
	Scopes []string

	mu          sync.Mutex
	tokenSource oauth2.TokenSource
}

func (p *GoogleSAProvider) GetHeaders(ctx context.Context) (map[string]string, error) {
	ts, err := p.getTokenSource(ctx)
	if err != nil {
		return nil, err
	}
	token, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("Google SA token: %w", err)
	}
	return map[string]string{"Authorization": "Bearer " + token.AccessToken}, nil
}

func (p *GoogleSAProvider) OnUnauthorized(ctx context.Context, _ *http.Response) (bool, error) {
	// Force refresh by clearing the cached token source
	p.mu.Lock()
	p.tokenSource = nil
	p.mu.Unlock()

	// Verify we can get a new token
	_, err := p.GetHeaders(ctx)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (p *GoogleSAProvider) getTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.tokenSource != nil {
		return p.tokenSource, nil
	}

	keyData, err := os.ReadFile(p.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("read service account key file %s: %w", p.KeyFile, err)
	}

	scopes := p.Scopes
	if len(scopes) == 0 {
		scopes = []string{"https://www.googleapis.com/auth/cloud-platform"}
	}

	creds, err := google.CredentialsFromJSON(ctx, keyData, scopes...)
	if err != nil {
		return nil, fmt.Errorf("parse service account key: %w", err)
	}

	p.tokenSource = creds.TokenSource
	return p.tokenSource, nil
}
