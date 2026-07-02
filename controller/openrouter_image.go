package controller

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func RelayOpenRouterImage(c *gin.Context) {
	var req service.OpenRouterImageRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		writeOpenRouterImageError(c, http.StatusBadRequest, err)
		return
	}

	imageReq, mode, err := service.ConvertOpenRouterImageRequest(req)
	if err != nil {
		writeOpenRouterImageError(c, http.StatusBadRequest, err)
		return
	}
	body, err := common.Marshal(imageReq)
	if err != nil {
		writeOpenRouterImageError(c, http.StatusBadRequest, err)
		return
	}
	storage, err := common.CreateBodyStorage(body)
	if err != nil {
		writeOpenRouterImageError(c, http.StatusBadRequest, err)
		return
	}
	oldStorage, _ := c.Get(common.KeyBodyStorage)
	if oldBodyStorage, ok := oldStorage.(common.BodyStorage); ok && oldBodyStorage != nil {
		_ = oldBodyStorage.Close()
	}
	c.Set(common.KeyBodyStorage, storage)
	c.Request.Body = io.NopCloser(common.ReaderOnly(storage))
	c.Request.ContentLength = int64(len(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.Header.Set("Content-Length", fmt.Sprintf("%d", len(body)))

	originalPath := c.Request.URL.Path
	originalRawPath := c.Request.URL.RawPath
	relayPath := "/v1/images/generations"
	relayMode := relayconstant.RelayModeImagesGenerations
	if mode == service.OpenRouterImageEditMode {
		relayPath = "/v1/images/edits"
		relayMode = relayconstant.RelayModeImagesEdits
	}
	originalRelayMode, hadOriginalRelayMode := c.Get("relay_mode")
	c.Set("relay_mode", relayMode)
	c.Request.URL.Path = relayPath
	c.Request.URL.RawPath = ""
	defer func() {
		c.Request.URL.Path = originalPath
		c.Request.URL.RawPath = originalRawPath
		if hadOriginalRelayMode {
			c.Set("relay_mode", originalRelayMode)
		}
	}()

	originalWriter := c.Writer
	recorder := newOpenRouterImageResponseRecorder(originalWriter)
	c.Writer = recorder
	Relay(c, types.RelayFormatOpenAIImage)
	c.Writer = originalWriter

	status := recorder.status
	if status == 0 {
		status = http.StatusOK
	}
	for key := range originalWriter.Header() {
		delete(originalWriter.Header(), key)
	}
	copyOpenRouterImageHeaders(recorder.header, originalWriter.Header())

	if status >= http.StatusBadRequest {
		c.Writer.WriteHeader(status)
		_, _ = c.Writer.Write(recorder.body.Bytes())
		return
	}

	converted, err := service.ConvertOpenRouterImageResponse(c.Request.Context(), recorder.body.Bytes(), service.DefaultOpenRouterImageConvertOptions())
	if err != nil {
		writeOpenRouterImageErrorToWriter(c, http.StatusBadGateway, err)
		return
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", fmt.Sprintf("%d", len(converted)))
	c.Writer.WriteHeader(status)
	_, _ = c.Writer.Write(converted)
}

type openRouterImageResponseRecorder struct {
	gin.ResponseWriter
	header http.Header
	body   bytes.Buffer
	status int
	size   int
}

func newOpenRouterImageResponseRecorder(writer gin.ResponseWriter) *openRouterImageResponseRecorder {
	return &openRouterImageResponseRecorder{
		ResponseWriter: writer,
		header:         make(http.Header),
	}
}

func (r *openRouterImageResponseRecorder) Header() http.Header {
	return r.header
}

func (r *openRouterImageResponseRecorder) WriteHeader(statusCode int) {
	if r.Written() {
		return
	}
	r.status = statusCode
}

func (r *openRouterImageResponseRecorder) WriteHeaderNow() {
	if r.status == 0 {
		r.status = http.StatusOK
	}
}

func (r *openRouterImageResponseRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.body.Write(data)
	r.size += n
	return n, err
}

func (r *openRouterImageResponseRecorder) WriteString(data string) (int, error) {
	return r.Write([]byte(data))
}

func (r *openRouterImageResponseRecorder) Written() bool {
	return r.size > 0 || r.status != 0
}

func (r *openRouterImageResponseRecorder) Size() int {
	return r.size
}

func (r *openRouterImageResponseRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func (r *openRouterImageResponseRecorder) Flush() {
}

func copyOpenRouterImageHeaders(source, target http.Header) {
	for key, values := range source {
		if strings.EqualFold(key, "Content-Length") {
			continue
		}
		for _, value := range values {
			target.Add(key, value)
		}
	}
}

func writeOpenRouterImageError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{
		"error": types.OpenAIError{
			Message: err.Error(),
			Type:    "invalid_request_error",
			Code:    types.ErrorCodeInvalidRequest,
		},
	})
}

func writeOpenRouterImageErrorToWriter(c *gin.Context, status int, err error) {
	for key := range c.Writer.Header() {
		delete(c.Writer.Header(), key)
	}
	writeOpenRouterImageError(c, status, err)
}
