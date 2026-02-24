package oauth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCallbackServer_StartAndPort(t *testing.T) {
	s := &CallbackServer{}
	if err := s.Start(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer s.Close()

	if s.Port <= 0 {
		t.Errorf("expected positive port, got %d", s.Port)
	}
}

func TestCallbackServer_RedirectURI(t *testing.T) {
	s := &CallbackServer{}
	if err := s.Start(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer s.Close()

	uri := s.RedirectURI()
	want := fmt.Sprintf("http://127.0.0.1:%d/callback", s.Port)
	if uri != want {
		t.Errorf("got %q, want %q", uri, want)
	}
}

func TestCallbackServer_SuccessfulCallback(t *testing.T) {
	s := &CallbackServer{}
	if err := s.Start(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer s.Close()

	// Simulate browser redirect in a goroutine
	go func() {
		time.Sleep(50 * time.Millisecond)
		http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback?code=test-code&state=test-state", s.Port))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	code, err := s.WaitForCallback(ctx, "test-state")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != "test-code" {
		t.Errorf("got code %q, want %q", code, "test-code")
	}
}

func TestCallbackServer_StateValidation(t *testing.T) {
	s := &CallbackServer{}
	if err := s.Start(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer s.Close()

	go func() {
		time.Sleep(50 * time.Millisecond)
		http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback?code=x&state=wrong", s.Port))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := s.WaitForCallback(ctx, "expected-state")
	if err == nil {
		t.Fatal("expected error for state mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "state mismatch") {
		t.Errorf("expected state mismatch error, got: %s", err)
	}
}

func TestCallbackServer_ErrorResponse(t *testing.T) {
	s := &CallbackServer{}
	if err := s.Start(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer s.Close()

	go func() {
		time.Sleep(50 * time.Millisecond)
		http.Get(fmt.Sprintf("http://127.0.0.1:%d/callback?error=access_denied&error_description=user+denied", s.Port))
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := s.WaitForCallback(ctx, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "access_denied") {
		t.Errorf("expected 'access_denied' in error, got: %s", err)
	}
}

func TestCallbackServer_ContextCancellation(t *testing.T) {
	s := &CallbackServer{}
	if err := s.Start(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer s.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := s.WaitForCallback(ctx, "")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout error, got: %s", err)
	}
}
