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
	ScoreQuestionType  JevQuestionType = "score"
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

	JevQuestionScore struct {
		JevQuestion
		Criteria []string `json:"criteria"`
	}

	JevNoulCriteria struct {
		True  string `json:"true"`
		False string `json:"false"`
	}

	JevQuestionNoul struct {
		JevQuestion
		Criteria JevNoulCriteria `json:"criteria"`
	}

	JevAnswer interface {
		GetType() JevQuestionType
	}

	JevResponse struct {
		Model   string               `json:"model"`
		Answers map[string]JevAnswer `json:"answers"`
	}

	JevAnswerNoul struct {
		Type JevQuestionType `json:"type"`
		Noul float64         `json:"noul,omitempty"`
	}

	JevAnswerChoice struct {
		Type          JevQuestionType    `json:"type"`
		Choice        string             `json:"choice"`
		Probabilities map[string]float64 `json:"probabilities"`
		Confidence    float64            `json:"confidence"`
	}

	JevAnswerScore struct {
		Type          JevQuestionType    `json:"type"`
		Score         float64            `json:"score,omitempty"`
		Legend        map[string]string  `json:"legend,omitempty"`
		Probabilities map[string]float64 `json:"probabilities"`
		Confidence    float64            `json:"confidence"`
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
	responseMap := make(map[string]any)

	if err := json.NewDecoder(resp.Body).Decode(&responseMap); err != nil {
		return nil, fmt.Errorf("failed to decode typesafe response body: %w", err)
	}

	model, ok := responseMap["model"].(string)
	if !ok {
		return nil, fmt.Errorf("typesafe response field %q is missing or not a string", "model")
	}

	rawAnswers, ok := responseMap["answers"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("typesafe response field %q is missing or not an object", "answers")
	}

	jevResp := &JevResponse{Model: model, Answers: make(map[string]JevAnswer, len(rawAnswers))}
	for key, value := range rawAnswers {
		answer, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("answer %q is not a JSON object", key)
		}

		answerType, ok := answer["type"].(string)
		if !ok {
			return nil, fmt.Errorf("answer %q is missing a string %q field", key, "type")
		}

		jsonBytes, err := json.Marshal(answer)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal answer %q: %w", key, err)
		}

		switch answerType {
		case string(ChoiceQuestionType):
			choice := &JevAnswerChoice{}
			if err := json.Unmarshal(jsonBytes, choice); err != nil {
				return nil, fmt.Errorf("failed to decode choice answer %q: %w", key, err)
			}
			jevResp.Answers[key] = choice
		case string(ScoreQuestionType):
			score := &JevAnswerScore{}
			if err := json.Unmarshal(jsonBytes, score); err != nil {
				return nil, fmt.Errorf("failed to decode score answer %q: %w", key, err)
			}
			jevResp.Answers[key] = score
		case string(NoulQuestionType):
			noul := &JevAnswerNoul{}
			if err := json.Unmarshal(jsonBytes, noul); err != nil {
				return nil, fmt.Errorf("failed to decode noul answer %q: %w", key, err)
			}
			jevResp.Answers[key] = noul
		default:
			return nil, fmt.Errorf("unsupported answer type %q for answer %q", answerType, key)
		}
	}

	return jevResp, nil
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
		case *JevQuestionScore:
			if len(q.Criteria) == 0 {
				return fmt.Errorf("criteria are required for score question %q", name)
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
		case ScoreQuestionType:
			question = &JevQuestionScore{}
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

func (r *JevQuestionChoice) GetType() JevQuestionType {
	return ChoiceQuestionType
}

func (r *JevQuestionNoul) GetType() JevQuestionType {
	return NoulQuestionType
}

func (r *JevQuestionScore) GetType() JevQuestionType {
	return ScoreQuestionType
}

func (r *JevQuestionChoice) GetInstructions() string {
	return r.Instructions
}

func (r *JevQuestionNoul) GetInstructions() string {
	return r.Instructions
}

func (r *JevQuestionScore) GetInstructions() string {
	return r.Instructions
}

func (r *JevAnswerChoice) GetType() JevQuestionType {
	return ChoiceQuestionType
}

func (r *JevAnswerNoul) GetType() JevQuestionType {
	return NoulQuestionType
}

func (r *JevAnswerScore) GetType() JevQuestionType {
	return ScoreQuestionType
}
