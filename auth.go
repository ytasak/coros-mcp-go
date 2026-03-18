package main

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
)

// session holds the current authentication state.
type session struct {
	mu          sync.RWMutex
	accessToken string
	userID      string
}

var currentSession = &session{}

func (s *session) getAuthHeaders() http.Header {
	s.mu.RLock()
	defer s.mu.RUnlock()

	h := http.Header{}
	if s.accessToken != "" {
		h.Set("accessToken", s.accessToken)
	}
	if s.userID != "" {
		yfHeader, _ := json.Marshal(map[string]string{"userId": s.userID})
		h.Set("yfheader", string(yfHeader))
	}
	return h
}

func (s *session) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accessToken = ""
	s.userID = ""
}

func (s *session) isLoggedIn() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.accessToken != "" && s.userID != ""
}

func (s *session) setCredentials(token, userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.accessToken = token
	s.userID = userID
}

type loginRequest struct {
	Account     string `json:"account"`
	AccountType int    `json:"accountType"`
	Pwd         string `json:"pwd"`
}

type loginResponseData struct {
	AccessToken string `json:"accessToken"`
	UserID      string `json:"userId"`
}

func ensureLoggedIn(client *http.Client, force bool) error {
	if !force && currentSession.isLoggedIn() {
		return nil
	}

	email := os.Getenv("COROS_EMAIL")
	password := os.Getenv("COROS_PASSWORD")
	if email == "" || password == "" {
		return fmt.Errorf(
			"COROS_EMAIL and COROS_PASSWORD environment variables are required. " +
				"Set them with your COROS Training Hub login credentials.",
		)
	}

	pwdHash := fmt.Sprintf("%x", md5.Sum([]byte(password)))

	body := loginRequest{
		Account:     email,
		AccountType: 2,
		Pwd:         pwdHash,
	}

	var apiResp apiResponse
	if err := postJSON(client, baseURL+"/account/login", nil, body, &apiResp); err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	if apiResp.Result != "0000" {
		return fmt.Errorf("login failed: %s", apiResp.Message)
	}

	var data loginResponseData
	if err := json.Unmarshal(apiResp.Data, &data); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	currentSession.setCredentials(data.AccessToken, data.UserID)
	log.Printf("Logged in as userId: %s", data.UserID)
	return nil
}
