package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"msg-classifier/internal/config"
	"net/http"
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
		Questions map[string]JevQuestion
	}

	JevState interface {
		// JevState is a runtime value, not a type constraint.
	}

	JevQuestion struct {
		Type           JevQuestionType   `json:"type"`
		Instructions   string            `json:"instructions"`
		CriteriaChoice map[string]string `json:"criteria,omitempty"`
		CriteriaScore  []string          `json:"criteria,omitempty"`
		True           string            `json:"true,omitempty"`
		False          string            `json:"false,omitempty"`
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

func MapToJevRequest(state JevState, questions map[string]JevQuestion) (*JevRequest, error) {
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

func MakeJevRequest(state JevState, questions map[string]JevQuestion) (*JevResponse, error) {
	jevRequest, err := MapToJevRequest(state, questions)
	if err != nil {
		return nil, fmt.Errorf("failed to create JevRequest: %w", err)
	}

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
		if question.Instructions == "" {
			return fmt.Errorf("instructions are required for question at index %v", i)
		}

		switch question.Type {
		case ChoiceQuestionType:
			if len(question.CriteriaChoice) == 0 {
				return fmt.Errorf("criteria are required for choice question at index %v", i)
			}
		case NoulQuestionType:
			if question.True == "" || question.False == "" {
				return fmt.Errorf("true and false are required for score question at index %v", i)
			}
		case ScoreQuestionType:
			if len(question.CriteriaScore) == 0 {
				return fmt.Errorf("criteria are required for choice question at index %v", i)
			}
		}
	}

	return nil
}
