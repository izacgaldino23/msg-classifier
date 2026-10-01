package jev

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// requests/*.json are embedded to avoid CWD-relative paths.
//
//go:embed requests/*.json
var requestTemplates embed.FS

const httpTimeout = 30 * time.Second

const (
	ChoiceQuestionType JevQuestionType = "choice"
	NoulQuestionType   JevQuestionType = "noul"
)

type (
	JevRequest struct {
		State     JevState                        `json:"state"`
		Model     string                          `json:"model"`
		Questions map[string]JevQuestionInterface `json:"questions"`
	}

	JevState interface{}

	JevQuestionType string

	JevQuestionInterface interface {
		GetType() JevQuestionType
		GetInstructions() string
	}

	JevQuestion struct {
		Type         JevQuestionType `json:"type"`
		Instructions string          `json:"instructions"`
	}

	JevQuestionChoice struct {
		JevQuestion
		Criteria map[string]string `json:"criteria"`
	}

	JevNoulCriteria struct {
		True  string `json:"true"`
		False string `json:"false"`
	}

	JevQuestionNoul struct {
		JevQuestion
		Criteria JevNoulCriteria `json:"criteria"`
	}

	JevResponse struct {
		Model   string         `json:"model"`
		Answers map[string]any `json:"answers"`
	}

	JevAnswerNoul struct {
		Noul float64 `json:"noul,omitempty"`
	}

	JevAnswerChoice struct {
		Choice     string  `json:"choice"`
		Confidence float64 `json:"confidence"`
	}
)

// Client calls the TypeSafe Jev API with injected configuration.
type Client struct {
	apiURL string
	token  string
	model  string
	http   *http.Client
}

func NewClient(apiURL, token, model string) *Client {
	return &Client{
		apiURL: apiURL,
		token:  token,
		model:  model,
		http:   &http.Client{Timeout: httpTimeout},
	}
}

// HttpResponseToJevResponse decodes a Typesafe API body into a JevResponse.
func HttpResponseToJevResponse(resp *http.Response) (*JevResponse, error) {
	var raw struct {
		Model   string                     `json:"model"`
		Answers map[string]json.RawMessage `json:"answers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode typesafe response body: %w", err)
	}
	if raw.Model == "" {
		return nil, fmt.Errorf("typesafe response field %q is missing or empty", "model")
	}

	jevResp := &JevResponse{Model: raw.Model, Answers: make(map[string]any, len(raw.Answers))}
	for key, rawAnswer := range raw.Answers {
		answer, err := decodeAnswer(key, rawAnswer)
		if err != nil {
			return nil, err
		}
		jevResp.Answers[key] = answer
	}

	return jevResp, nil
}

// decodeAnswer peeks at the answer type and decodes into the matching struct.
// It never panics: a missing or unknown type is an explicit error.
func decodeAnswer(key string, raw json.RawMessage) (any, error) {
	var header struct {
		Type JevQuestionType `json:"type"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return nil, fmt.Errorf("failed to decode answer %q: %w", key, err)
	}

	var answer any
	switch header.Type {
	case ChoiceQuestionType:
		answer = &JevAnswerChoice{}
	case NoulQuestionType:
		answer = &JevAnswerNoul{}
	default:
		return nil, fmt.Errorf("unsupported answer type %q for answer %q", header.Type, key)
	}
	if err := json.Unmarshal(raw, answer); err != nil {
		return nil, fmt.Errorf("failed to decode %s answer %q: %w", header.Type, key, err)
	}
	return answer, nil
}

// MakeJevRequest validates and POSTs a JevRequest to the Typesafe API.
func (c *Client) MakeJevRequest(jevRequest *JevRequest) (*JevResponse, error) {
	jevRequest.Model = c.model

	if err := validateJevRequest(jevRequest); err != nil {
		return nil, err
	}

	bodyData, err := json.Marshal(jevRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal jev request to json: %w", err)
	}

	payload := bytes.NewBuffer(bodyData)

	req, err := http.NewRequest(http.MethodPost, c.apiURL, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to typesafe api: %w", err)
	}
	req.Header.Add("Authorization", "Bearer "+c.token)
	req.Header.Add("Content-Type", "application/json")

	response, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to typesafe api: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		// Best-effort read of the upstream error body for logging only.
		body, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		log.Printf("Typesafe error: %v", string(body))
		return nil, fmt.Errorf("typesafe api returned an error status: %v", response.Status)
	}

	jevResponse, err := HttpResponseToJevResponse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response from typesafe api to jevresponse: %w", err)
	}

	return jevResponse, nil
}

// MakeJevRequestFromFile loads an embedded prompt template, injects state,
// and delegates to MakeJevRequest.
func (c *Client) MakeJevRequestFromFile(state JevState, fileName string) (*JevResponse, error) {
	request, err := LoadJevRequestFromFile(fileName)
	if err != nil {
		return nil, err
	}

	request.State = state

	return c.MakeJevRequest(request)
}

func validateJevRequest(request *JevRequest) error {
	switch v := request.State.(type) {
	case string:
		if v == "" {
			return fmt.Errorf("state is required")
		}
	case map[string]any:
		if len(v) == 0 {
			return fmt.Errorf("state is required")
		}
	default:
		return fmt.Errorf("state is required")
	}

	if request.Model == "" {
		return fmt.Errorf("model is required")
	}

	if len(request.Questions) == 0 {
		return fmt.Errorf("questions are required")
	}

	for name, question := range request.Questions {
		if question.GetType() == "" || question.GetInstructions() == "" {
			return fmt.Errorf("instructions are required for question %q", name)
		}

		switch q := question.(type) {
		case *JevQuestionChoice:
			if len(q.Criteria) == 0 {
				return fmt.Errorf("criteria are required for choice question %q", name)
			}
		case *JevQuestionNoul:
			if q.Criteria.True == "" || q.Criteria.False == "" {
				return fmt.Errorf("true and false are required for noul question %q", name)
			}
		}
	}

	return nil
}

// LoadJevRequestFromFile reads and polymorphically decodes a prompt template
// embedded from requests/*.json.
func LoadJevRequestFromFile(fileName string) (*JevRequest, error) {
	data, err := requestTemplates.ReadFile("requests/" + fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded jevrequest template %q: %w", fileName, err)
	}

	var rawRequest struct {
		State     JevState                   `json:"state"`
		Model     string                     `json:"model"`
		Questions map[string]json.RawMessage `json:"questions"`
	}
	if err := json.Unmarshal(data, &rawRequest); err != nil {
		return nil, fmt.Errorf("failed to decode jevrequest template %q: %w", fileName, err)
	}

	request := &JevRequest{
		State:     rawRequest.State,
		Model:     rawRequest.Model,
		Questions: make(map[string]JevQuestionInterface, len(rawRequest.Questions)),
	}
	for name, rawQuestion := range rawRequest.Questions {
		var questionType struct {
			Type JevQuestionType `json:"type"`
		}
		if err := json.Unmarshal(rawQuestion, &questionType); err != nil {
			return nil, fmt.Errorf("failed to decode question %q in %q: %w", name, fileName, err)
		}

		var question JevQuestionInterface
		switch questionType.Type {
		case ChoiceQuestionType:
			question = &JevQuestionChoice{}
		case NoulQuestionType:
			question = &JevQuestionNoul{}
		default:
			return nil, fmt.Errorf("unsupported question type %q for question %q in %q", questionType.Type, name, fileName)
		}

		if err := json.Unmarshal(rawQuestion, question); err != nil {
			return nil, fmt.Errorf("failed to decode question %q in %q: %w", name, fileName, err)
		}
		request.Questions[name] = question
	}

	return request, nil
}

// GetType returns the question type. It is declared on each question struct
// because the type is what discriminates the polymorphic decode.
func (r *JevQuestionChoice) GetType() JevQuestionType {
	return ChoiceQuestionType
}

func (r *JevQuestionNoul) GetType() JevQuestionType {
	return NoulQuestionType
}

// GetInstructions returns the question instructions; it is promoted to every
// question struct through the embedded JevQuestion.
func (r *JevQuestion) GetInstructions() string {
	return r.Instructions
}
