package handlers

import (
	"fmt"
	"msg-classifier/pkg"
	"net/http"

	"github.com/gin-gonic/gin"
)

type (
	MsgHandler struct{}

	ReceiveMessageRequest struct {
		Message string `json:"message"`
		UserID  string `json:"user_id"`
	}
)

func NewMsgHandler() *MsgHandler {
	return &MsgHandler{}
}

func (h *MsgHandler) ReceiveMessage(c *gin.Context) {
	request := &ReceiveMessageRequest{}

	err := c.Bind(request)
	if err != nil {
		pkg.ReturnJson(c, http.StatusBadRequest, "Invalid request")
		return
	}

	jevResponse, err := checkMessageCategory(request)
	pkg.ReturnJson(c, http.StatusInternalServerError, err.Error())

	jevResponseKind, err := checkRequestKind(request)
	pkg.ReturnJson(c, http.StatusInternalServerError, err.Error())

	// Verify if user want save or get data
	// if user want save, save the data to database
	// if user want get, get the data from database
	// TODO

	// for while, save the response to a file as json
	pkg.ReturnJson(c, http.StatusOK, gin.H{
		"category":   jevResponse.Answers["classification"].Choice,
		"confidence": fmt.Sprintf("%.2f", jevResponse.Answers["classification"].Confidence),
		"kind":       jevResponseKind.Answers["adding_or_requiring"].Score,
	})
}

func checkMessageCategory(request *ReceiveMessageRequest) (*pkg.JevResponse, error) {
	// Get subject from message
	jevResponse, err := pkg.MakeJevRequest(map[string]any{
		"user":    request.UserID, // TODO: change this userId to user name
		"message": request.Message,
	}, map[string]pkg.JevQuestion{
		"classification": {
			Type:         pkg.ChoiceQuestionType,
			Instructions: "Which category does this message belong to",
			CriteriaChoice: map[string]string{
				"schedule": "alarms, meetings or future events",
				"contact":  "contact information, phone numbers, email addresses, or social media handles",
				"finance":  "payment, billing, or financial information",
				"notes":    "notes, reminders, or to-do lists",
				"other":    "none of other categories",
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return jevResponse, nil
}

func checkRequestKind(request *ReceiveMessageRequest) (*pkg.JevResponse, error) {
	// Get subject from message
	jevResponse, err := pkg.MakeJevRequest(map[string]any{
		"user":    request.UserID, // TODO: change this userId to user name
		"message": request.Message,
	}, map[string]pkg.JevQuestion{
		"adding_or_requiring": {
			Type:         pkg.ScoreQuestionType,
			Instructions: "Is this a request to add or require something?",
			CriteriaScore: []string{
				"add", "require", "both", "neither",
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return jevResponse, nil
}
