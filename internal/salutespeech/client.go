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
	AuthURL     string
	AuthToken   string
	Scope       string
	client      *resty.Client
	accessToken string
	expiresAt   int64
}

func NewSaluteSpeechClient(authURL, authToken, scope string) *SaluteSpeechClient {
	salute := &SaluteSpeechClient{
		AuthURL:   authURL,
		AuthToken: authToken,
		Scope:     scope,
		client:    resty.New(),
	}
	salute.Authorization()
	return salute
}

func (c *SaluteSpeechClient) Authorization() error {
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
	// fmt.Println("Status code: ", resp.StatusCode(), "Auth result: ", resp.String())

	var authResp models.SaluteAuthResponse
	err = json.Unmarshal(resp.Body(), &authResp)
	if err != nil {
		return fmt.Errorf("error unmarshal SaluteSpeech API authorization")
	}

	c.accessToken = authResp.AccessToken
	c.expiresAt = authResp.ExpiresAt

	return nil
}

func (c *SaluteSpeechClient) Send(file io.Reader) (models.SaluteUploadResponse, error) {
	// fmt.Printf("Access token for Salute: %s\n", fmt.Sprintf("Bearer %s", c.accessToken))

	resp, err := c.client.R().
		SetHeader("Content-Type", "audio/mpeg").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", c.accessToken)).
		SetBody(file).
		Post("https://smartspeech.sber.ru/rest/v1/data:upload")
	if err != nil {
		return models.SaluteUploadResponse{}, fmt.Errorf("error upload file into Salute: %w", err)
	}

	if resp.StatusCode() == 401 {
		return models.SaluteUploadResponse{}, fmt.Errorf("Status code from salute - 401")
	}

	var res models.SaluteUploadResponse
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return models.SaluteUploadResponse{}, fmt.Errorf("error unmarshall salute upload response: %w", err)
	}
	fmt.Println("Upload file result: ", resp.String())
	return res, nil
}

func (c *SaluteSpeechClient) StartProcess(fileID, audio_encoding string) (models.SaluteTaskResponse, error) {
	var res models.SaluteTaskResponse

	resp, err := c.client.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", c.accessToken)).
		SetBody(fmt.Sprintf(task, audio_encoding, true, fileID)).
		Post("https://smartspeech.sber.ru/rest/v1/speech:async_recognize")
	if err != nil {
		return res, err
	}

	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		return res, fmt.Errorf("error unmarshall salute create task response: %w", err)
	}
	fmt.Println("Create task result: ", resp.String())
	return res, nil
}

func (c *SaluteSpeechClient) CheckTask(taskID string) (models.SaluteTaskResponse, error) {
	var taskStatus models.SaluteTaskResponse

	url := fmt.Sprintf("https://smartspeech.sber.ru/rest/v1/task:get?id=%s", taskID)
	fmt.Println("Check file status URL: ", url)

	resp, err := c.client.R().
		SetHeader("Accept", "application/octet-stream").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", c.accessToken)).
		SetResult(&taskStatus).
		Get(url)
	fmt.Println("Check task status: ", resp.String())
	if err != nil {
		fmt.Println("err:", err.Error())
		return models.SaluteTaskResponse{}, err
	}

	return taskStatus, nil
}

func (c *SaluteSpeechClient) DownloadFile(responseFileID string) (string, error) {
	var fileData []models.SaluteParsedFile

	url := fmt.Sprintf("https://smartspeech.sber.ru/rest/v1/data:download?response_file_id=%s", responseFileID)
	fmt.Println("Download file URL: ", url)

	resp, err := c.client.R().
		SetHeader("Accept", "application/octet-stream").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", c.accessToken)).
		Get(url)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(resp.Body(), &fileData)
	if err != nil {
		return "", err
	}
	fmt.Println("Download file result: ", fileData[0].Results[0].NormalizedText)

	return fileData[0].Results[0].NormalizedText, nil
}
