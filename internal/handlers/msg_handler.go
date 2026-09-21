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
		Message string `json:"message" form:"message"`
		UserID  string `json:"user_id" form:"user_id"`
	}

	ReceiveMessageResponse struct {
		Category map[string]any `json:"category"`
		Kind     map[string]any `json:"kind"`
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
	if err != nil {
		pkg.ReturnJson(c, http.StatusInternalServerError, err.Error())
		return
	}

	jevResponseKind, err := checkRequestKind(request)
	if err != nil {
		pkg.ReturnJson(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Verify if user want save or get data
	// if user want save, save the data to database
	// if user want get, get the data from database
	// TODO

	response := ReceiveMessageResponse{
		Category: map[string]any{
			"choice":     jevResponse.Answers["classification"].Choice,
			"confidence": fmt.Sprintf("%.2f", jevResponse.Answers["classification"].Confidence*100),
		},
		Kind: map[string]any{
			"score":      jevResponseKind.Answers["adding_or_requiring"].Score,
			"confidence": fmt.Sprintf("%.2f", jevResponseKind.Answers["adding_or_requiring"].Confidence*100),
			"legend":     jevResponseKind.Answers["adding_or_requiring"].Legend,
		},
	}

	// for while, save the response to a file as json
	c.HTML(http.StatusOK, "resultado", response)
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
