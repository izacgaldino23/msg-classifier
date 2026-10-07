package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"msg-classifier/internal/jevq"
	"msg-classifier/internal/repository"
	"msg-classifier/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusFor(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"upstream", fmt.Errorf("failed to call jev: %w", jevq.ErrUpstream), http.StatusBadGateway},
		{"invalid filter", fmt.Errorf("%w: contact filter %q", services.ErrInvalidFilter, "x"), http.StatusBadRequest},
		{"invalid data", fmt.Errorf("%w: amount %q", services.ErrInvalidData, "x"), http.StatusBadRequest},
		{"not found", fmt.Errorf("failed to load contact: %w", repository.ErrNotFound), http.StatusNotFound},
		{"anything else", errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, StatusFor(tt.err))
		})
	}
}

func TestRenderErrorBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	RenderError(c, http.StatusBadGateway, "failed to call jev")

	assert.Equal(t, http.StatusBadGateway, recorder.Code)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &decoded), "body must be JSON")
	assert.Equal(t, "failed to call jev", decoded["error"])
	assert.Len(t, decoded, 1, `only the "error" key`)
	assert.Equal(t, "application/json; charset=utf-8", recorder.Header().Get("Content-Type"))
}
