package handlers

import (
	"fmt"
	"msg-classifier/pkg"
	"msg-classifier/pkg/jev"
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

func checkMessageCategory(request *ReceiveMessageRequest) (*jev.JevResponse, error) {
	jevResponse, err := jev.MakeJevRequestFromFile(map[string]any{
		"user":    request.UserID, // TODO: change this userId to user name
		"message": request.Message,
	}, "classification.json")
	if err != nil {
		return nil, err
	}

	return jevResponse, nil
}

func checkRequestKind(request *ReceiveMessageRequest) (*jev.JevResponse, error) {
	jevResponse, err := jev.MakeJevRequestFromFile(map[string]any{
		"user":    request.UserID, // TODO: change this userId to user name
		"message": request.Message,
	}, "request_kind.json")
	if err != nil {
		return nil, err
	}

	return jevResponse, nil
}
