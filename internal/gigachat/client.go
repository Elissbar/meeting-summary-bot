package gigachat

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Elissbar/meeting-summary-bot/internal/models"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
)

type GigaChatClient struct {
	AuthURL     string
	AuthToken   string
	Scope       string
	client      *resty.Client
	accessToken string
	expiresAt   int64
}

func NewGigaChatClient(authURL, authToken, scope string) *GigaChatClient {
	giga := &GigaChatClient{
		AuthURL:   authURL,
		AuthToken: authToken,
		Scope:     scope,
		client:    resty.New(),
	}
	giga.Authorization()
	return giga
}

func (c *GigaChatClient) Authorization() error {
	err := c.auth()
	if err != nil {
		return fmt.Errorf("authorization SaluteSpeech API error")
	}

	return nil
}

func (c *GigaChatClient) auth() error {
	uuid := uuid.NewString()

	resp, err := c.client.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("Accept", "application/json").
		SetHeader("RqUID", uuid).
		SetHeader("Authorization", fmt.Sprintf("Basic %s", c.AuthToken)).
		SetFormData(map[string]string{"scope": c.Scope}).
		Post(c.AuthURL)
	if err != nil {
		return fmt.Errorf("authorization SaluteSpeech API error")
	}

	var authResp models.SaluteAuthResponse
	err = json.Unmarshal(resp.Body(), &authResp)
	if err != nil {
		return fmt.Errorf("error unmarshal SaluteSpeech API authorization")
	}

	c.accessToken = authResp.AccessToken
	c.expiresAt = authResp.ExpiresAt

	return nil
}

func (c *GigaChatClient) UpdateToken() error {
	// Если авторизация просрочилась - обновляем
	expTime := time.Unix(c.expiresAt, 0).Add(-15 * time.Second)
	if time.Now().After(expTime) {
		err := c.auth()
		if err != nil {
			return fmt.Errorf("authorization SaluteSpeech API error")
		}
	}
	return nil
}

func (c *GigaChatClient) Send(data string, chat bool) (string, error) {
	prompt := processTranscriptionPrompt
	if chat { // Команда /chat
		prompt = userQuestion
	}
	resp, err := c.client.R().
		SetHeader("Content-Type", "audio/mpeg").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", c.accessToken)).
		SetBody(fmt.Sprintf(prompt, data)).
		Post("https://gigachat.devices.sberbank.ru/api/v1/chat/completions")
	if err != nil {
		return "", fmt.Errorf("error upload file into Salute: %w", err)
	}

	var res models.ChatResponse
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return "", fmt.Errorf("error unmarshall salute upload response: %w", err)
	}
	fmt.Println("Giga reponse: ", resp.String())
	return res.Choices[0].Message.Content, nil
}
