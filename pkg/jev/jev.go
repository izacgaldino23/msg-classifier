package jev

import (
	"bytes"
	"encoding/json"
	"fmt"
	"msg-classifier/internal/config"
	"net/http"
	"os"
)

const (
	ChoiceQuestionType JevQuestionType = "choice"
	ScoreQuestionType  JevQuestionType = "score"
	NoulQuestionType   JevQuestionType = "noul"
)

type (
	JevRequest struct {
		State     JevState `json:"state"`
		Model     string   `json:"model"`
		Questions map[string]JevQuestionInterface
	}

	JevState interface {
		// JevState is a runtime value, not a type constraint.
	}

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

	JevQuestionNoul struct {
		JevQuestion
		True  string `json:"true,omitempty"`
		False string `json:"false,omitempty"`
	}

	JevAnswer struct {
		Type       JevQuestionType `json:"type"`
		Noul       string          `json:"noul,omitempty"`
		Choice     string          `json:"choice,omitempty"`
		Score      int             `json:"score,omitempty"`
		Confidence float64         `json:"confidence,omitempty"`
	}

	JevQuestionType string

	JevResponse struct {
		Model   string               `json:"model"`
		Answers map[string]JevAnswer `json:"answers"`
	}
)

var client = &http.Client{}

func JevRequestFromFile(filePath string) (*JevRequest, error) {
	return nil, nil
}

func MapToJevRequest(state JevState, questions map[string]JevQuestionInterface) (*JevRequest, error) {
	jevReq := &JevRequest{State: state, Questions: questions}

	if err := validateJevRequest(jevReq); err != nil {
		return nil, err
	}

	return jevReq, nil
}

func HttpResponseToJevResponse(resp *http.Response) (*JevResponse, error) {
	jevResp := &JevResponse{}

	if err := json.NewDecoder(resp.Body).Decode(jevResp); err != nil {
		return nil, err
	}

	return jevResp, nil
}

func MakeJevRequest(jevRequest *JevRequest) (*JevResponse, error) {
	// jevRequest, err := MapToJevRequest(state, questions)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create JevRequest: %w", err)
	// }

	bodyData, err := json.Marshal(jevRequest)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal JevRequest to JSON: %w", err)
	}

	payload := bytes.NewBuffer(bodyData)

	// Post to typesafe API
	req, err := http.NewRequest("POST", config.GetEnv().TypesafeApiUrl, payload)
	if err != nil {
		return nil, fmt.Errorf("Failed to create request to Typesafe API %w", err)
	}
	req.Header.Add("Authentication", "Bearer "+config.GetEnv().TypesafeToken)
	req.Header.Add("Content-Type", "application/json")

	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Failed to send request to Typesafe API %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Typesafe API returned an error status: %v", response.Status)
	}

	jevResponse, err := HttpResponseToJevResponse(response)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse response from Typesafe API to JevResponse: %w", err)
	}

	return jevResponse, nil
}

func MakeJevRequestFromFile(state JevState, filePath string) (*JevResponse, error) {
	request, err := LoadJevRequestFromFile(filePath)
	if err != nil {
		return nil, err
	}

	return MakeJevRequest(request)
}

func validateJevRequest(request *JevRequest) error {
	// if state is string, check if it is empty
	// else, if state is map, check if it is empty
	// else, if state is other type, return error
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

	for i, question := range request.Questions {
		if question.GetType() == "" || question.GetInstructions() == "" {
			return fmt.Errorf("instructions are required for question at index %v", i)
		}

		switch q := question.(type) {
		case *JevQuestionChoice:
			if len(q.Criteria) == 0 {
				return fmt.Errorf("criteria are required for choice question at index %v", i)
			}
		case *JevQuestionNoul:
			if q.True == "" || q.False == "" {
				return fmt.Errorf("true and false are required for score question at index %v", i)
			}
		case *JevQuestionScore:
			if len(q.Criteria) == 0 {
				return fmt.Errorf("criteria are required for choice question at index %v", i)
			}
		}
	}

	return nil
}

func LoadJevRequestFromFile(filePath string) (*JevRequest, error) {
	finalPath := "./requests/" + filePath
	data, err := os.ReadFile(finalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JevRequest file %q: %w", finalPath, err)
	}

	var rawRequest struct {
		State     JevState                   `json:"state"`
		Model     string                     `json:"model"`
		Questions map[string]json.RawMessage `json:"questions"`
	}
	if err := json.Unmarshal(data, &rawRequest); err != nil {
		return nil, fmt.Errorf("failed to decode JevRequest file %q: %w", finalPath, err)
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
			return nil, fmt.Errorf("failed to decode question %q in %q: %w", name, finalPath, err)
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
			return nil, fmt.Errorf("unsupported question type %q for question %q in %q", questionType.Type, name, finalPath)
		}

		if err := json.Unmarshal(rawQuestion, question); err != nil {
			return nil, fmt.Errorf("failed to decode question %q in %q: %w", name, finalPath, err)
		}
		request.Questions[name] = question
	}

	if err := validateJevRequest(request); err != nil {
		return nil, fmt.Errorf("invalid JevRequest in %q: %w", finalPath, err)
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
