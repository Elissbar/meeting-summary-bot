package models

type SaluteAuthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

type SaluteUploadResponse struct {
	Status int    `json:"status"`
	Result result `json:"result"`
}

type result struct {
	RequestFileID string `json:"request_file_id"`
}

type SaluteTaskResponse struct {
	Status int         `json:"status"`
	Result createdTask `json:"result"`
}

type createdTask struct {
	ID             string `json:"id"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	Status         string `json:"status"`
	ResponseFileID string `json:"response_file_id,omitempty"`
}

type SaluteParsedFile struct {
	Results             []Result       `json:"results"`
	Eou                 bool           `json:"eou"`
	EmotionsResult      EmotionsResult `json:"emotions_result"`
	ProcessedAudioStart string         `json:"processed_audio_start"`
	ProcessedAudioEnd   string         `json:"processed_audio_end"`
	BackendInfo         BackendInfo    `json:"backend_info"`
	Channel             int            `json:"channel"`
	SpeakerInfo         SpeakerInfo    `json:"speaker_info"`
	EouReason           string         `json:"eou_reason"`
	Insight             string         `json:"insight"`
	PersonIdentity      PersonIdentity `json:"person_identity"`
}

type Result struct {
	Text           string          `json:"text"`
	NormalizedText string          `json:"normalized_text"`
	Start          string          `json:"start"`
	End            string          `json:"end"`
	WordAlignments []WordAlignment `json:"word_alignments"`
}

type WordAlignment struct {
	Word  string `json:"word"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type EmotionsResult struct {
	Positive float64 `json:"positive"`
	Neutral  float64 `json:"neutral"`
	Negative float64 `json:"negative"`
}

type BackendInfo struct {
	ModelName     string `json:"model_name"`
	ModelVersion  string `json:"model_version"`
	ServerVersion string `json:"server_version"`
}

type SpeakerInfo struct {
	SpeakerID             int     `json:"speaker_id"`
	MainSpeakerConfidence float64 `json:"main_speaker_confidence"`
}

type PersonIdentity struct {
	Age         string  `json:"age"`
	Gender      string  `json:"gender"`
	AgeScore    float64 `json:"age_score"`
	GenderScore float64 `json:"gender_score"`
}

// GigaChat
type ChatResponse struct {
	Choices []Choice `json:"choices"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Object  string   `json:"object"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	FinishReason string  `json:"finish_reason"`
	Index        int     `json:"index"`
	Message      Message `json:"message"`
}

type Message struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}

type Usage struct {
	CompletionTokens int `json:"completion_tokens"`
	PromptTokens     int `json:"prompt_tokens"`
	SystemTokens     int `json:"system_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
