package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/MKVAppDev/go-ingestion/internal/model"
)

const (
	authURL = "https://api.dnse.com.vn/user-service/api/auth"
	meURL   = "https://api.dnse.com.vn/user-service/api/me"
)

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
}

func Authentication(ctx context.Context, username, password string) (string, error) {

	request := authRequest{
		Username: username,
		Password: password,
	}

	body, err := json.Marshal(request)

	if err != nil {
		return "", nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, authURL, bytes.NewReader(body))

	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("auth status %d", resp.StatusCode)
	}

	var ar authResponse

	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return "", err
	}

	if ar.Token == "" {
		return "", fmt.Errorf("empty token")
	}

	return ar.Token, nil
}

func GetInvestorInfo(ctx context.Context, token string) (*model.InvestorInfo, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, meURL, nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("me status %d", resp.StatusCode)
	}

	var info model.InvestorInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}

	if info.InvestorID == "" {
		return nil, fmt.Errorf("empty investorId")
	}

	return &info, nil
}
