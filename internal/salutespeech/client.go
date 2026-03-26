package salutespeech

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/Elissbar/meeting-summary-bot/internal/models"
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
)

type SaluteSpeechClient struct {
	URL         string
	AuthToken   string
	Scope       string
	client      *resty.Client
	accessToken string
	expiresAt   int64
}

func NewSaluteSpeechClient(url, authToken, scope string) *SaluteSpeechClient {
	return &SaluteSpeechClient{
		URL:       url,
		AuthToken: authToken,
		Scope:     scope,
		client:    resty.New(),
	}
}

func (c *SaluteSpeechClient) Authorization() error {
	uuid := uuid.NewString()

	resp, err := c.client.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetHeader("Accept", "application/json").
		SetHeader("RqUID", uuid).
		SetHeader("Authorization", fmt.Sprintf("Basic %s", c.AuthToken)).
		SetFormData(map[string]string{"scope": c.Scope}).
		Post(c.URL)
	if err != nil {
		return fmt.Errorf("authorization SaluteSpeech API error")
	}
	fmt.Println("Auth result: ", resp.String())

	var authResp models.SaluteAuthResponse
	err = json.Unmarshal(resp.Body(), &authResp)
	if err != nil {
		return fmt.Errorf("error unmarshal SaluteSpeech API authorization")
	}

	c.accessToken = authResp.AccessToken
	c.expiresAt = authResp.ExpiresAt

	return nil
}

func (c *SaluteSpeechClient) Send(file io.ReadCloser) error {
	resp, err := c.client.R().
		SetHeader("Content-Type", "audio/mpeg").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", c.accessToken)).
		SetBody(file).
		Post("https://smartspeech.sber.ru/rest/v1/data:upload")
	if err != nil {
		return fmt.Errorf("error upload file")
	}

	fmt.Println("Upload file result: ", resp.String())
	return nil
}
