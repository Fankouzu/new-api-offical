package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func TestWriteTaskDataContentDecodesOpenAIImageBase64(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	body, err := common.Marshal(map[string]any{
		"created": 123,
		"data": []any{
			map[string]any{"b64_json": "aGVsbG8="},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = writeTaskDataContent(c, &model.Task{Data: body})
	if err != nil {
		t.Fatal(err)
	}

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("content-type = %q, want image/png", got)
	}
	if got := recorder.Body.String(); got != "hello" {
		t.Fatalf("body = %q, want hello", got)
	}
}
