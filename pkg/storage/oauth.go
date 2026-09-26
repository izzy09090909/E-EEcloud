package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	googleAuthEndpoint  = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenEndpoint = "https://oauth2.googleapis.com/token"
	googleDriveScope    = "https://www.googleapis.com/auth/drive.file"
)

// OAuthTokenResult carries the tokens returned by Google after a successful exchange.
type OAuthTokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

// RunGoogleOAuthLoopback starts a short-lived local HTTP listener on 127.0.0.1, builds the
// Google OAuth2 "installed app" authorization URL, invokes openURL with it (expected to open
// the system browser, e.g. via Wails runtime.BrowserOpenURL), and blocks until Google
// redirects back to the loopback listener with an authorization code -- then exchanges that
// code for tokens. Times out after 3 minutes if the user never completes the flow.
func RunGoogleOAuthLoopback(ctx context.Context, clientID, clientSecret string, openURL func(string)) (*OAuthTokenResult, error) {
	if clientID == "" || clientSecret == "" {
		return nil, errors.New("google OAuth client ID/secret are not configured")
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to bind OAuth callback listener: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/oauth/callback", port)

	authURL := buildGoogleAuthURL(clientID, redirectURI)

	type callbackResult struct {
		code string
		err  error
	}
	resultCh := make(chan callbackResult, 1)
	var once sync.Once

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if errStr := q.Get("error"); errStr != "" {
			once.Do(func() { resultCh <- callbackResult{err: fmt.Errorf("google returned OAuth error: %s", errStr)} })
			fmt.Fprint(w, "Authorization failed. You can close this window and return to CipherDrive.")
			return
		}
		code := q.Get("code")
		if code == "" {
			once.Do(func() { resultCh <- callbackResult{err: errors.New("no authorization code returned")} })
			fmt.Fprint(w, "Authorization failed. You can close this window and return to CipherDrive.")
			return
		}
		once.Do(func() { resultCh <- callbackResult{code: code} })
		fmt.Fprint(w, "CipherDrive connected to Google Drive. You can close this window now.")
	})

	srv := &http.Server{Handler: mux}
	go srv.Serve(listener)
	defer srv.Shutdown(context.Background())

	if openURL != nil {
		openURL(authURL)
	}

	select {
	case res := <-resultCh:
		if res.err != nil {
			return nil, res.err
		}
		return exchangeGoogleCode(ctx, clientID, clientSecret, res.code, redirectURI)
	case <-time.After(3 * time.Minute):
		return nil, errors.New("timed out waiting for Google authorization (3 minutes)")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func buildGoogleAuthURL(clientID, redirectURI string) string {
	v := url.Values{}
	v.Set("client_id", clientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("response_type", "code")
	v.Set("scope", googleDriveScope)
	v.Set("access_type", "offline")
	v.Set("prompt", "consent")
	return googleAuthEndpoint + "?" + v.Encode()
}

func exchangeGoogleCode(ctx context.Context, clientID, clientSecret, code, redirectURI string) (*OAuthTokenResult, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", redirectURI)

	return doGoogleTokenRequest(ctx, form)
}

// RefreshGoogleAccessToken exchanges a stored refresh token for a fresh access token. Google
// generally omits refresh_token from refresh responses, so the caller should keep using the
// original refresh token if this result's RefreshToken is empty (handled by GDriveDriver).
func RefreshGoogleAccessToken(ctx context.Context, clientID, clientSecret, refreshToken string) (*OAuthTokenResult, error) {
	if refreshToken == "" {
		return nil, errors.New("no refresh token available; user must reconnect Google Drive")
	}
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("client_secret", clientSecret)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")

	return doGoogleTokenRequest(ctx, form)
}

func doGoogleTokenRequest(ctx context.Context, form url.Values) (*OAuthTokenResult, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", googleTokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request to google failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google token endpoint returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse google token response: %w", err)
	}

	return &OAuthTokenResult{
		AccessToken:  parsed.AccessToken,
		RefreshToken: parsed.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(parsed.ExpiresIn) * time.Second),
	}, nil
}
