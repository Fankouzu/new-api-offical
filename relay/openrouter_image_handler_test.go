package relay

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestImageHelperRunsOpenRouterImageConversionBeforeBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service.InitHttpClient()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"created":1782978351,"data":[{"url":"https://example.com/image.png"}]}`))
	}))
	defer upstream.Close()

	base := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(base)
	recorder := newTestOpenRouterImageRecorder(c.Writer)
	c.Writer = recorder
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", bytes.NewReader([]byte(`{"model":"qwen-image","prompt":"draw"}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set(service.OpenRouterImageResponseConverterContextKey, service.OpenRouterImageResponseConverter(func(context.Context, []byte) ([]byte, error) {
		return nil, errors.New("conversion failed")
	}))
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
	common.SetContextKey(c, constant.ContextKeyChannelId, 1)
	common.SetContextKey(c, constant.ContextKeyChannelBaseUrl, upstream.URL)
	common.SetContextKey(c, constant.ContextKeyChannelKey, "test-key")
	common.SetContextKey(c, constant.ContextKeyOriginalModel, "qwen-image")

	billing := &testBillingSettler{}
	err := ImageHelper(c, &relaycommon.RelayInfo{
		RelayMode:       relayconstant.RelayModeImagesGenerations,
		RelayFormat:     types.RelayFormatOpenAIImage,
		OriginModelName: "qwen-image",
		RequestURLPath:  "/v1/images/generations",
		PriceData: types.PriceData{
			UsePrice:    true,
			OtherRatios: map[string]float64{},
		},
		Request: &dto.ImageRequest{
			Model:  "qwen-image",
			Prompt: "draw",
		},
		Billing: billing,
	})

	require.Error(t, err)
	require.True(t, types.IsSkipRetryError(err))
	require.Equal(t, http.StatusBadGateway, err.StatusCode)
	require.False(t, billing.settled)
	require.Empty(t, recorder.BodyBytes())
}

type testBillingSettler struct {
	settled bool
}

func (s *testBillingSettler) Settle(int) error {
	s.settled = true
	return nil
}

func (s *testBillingSettler) Refund(*gin.Context) {
}

func (s *testBillingSettler) NeedsRefund() bool {
	return true
}

func (s *testBillingSettler) GetPreConsumedQuota() int {
	return 1
}

func (s *testBillingSettler) Reserve(int) error {
	return nil
}

type testOpenRouterImageRecorder struct {
	gin.ResponseWriter
	header http.Header
	body   bytes.Buffer
	status int
	size   int
}

func newTestOpenRouterImageRecorder(writer gin.ResponseWriter) *testOpenRouterImageRecorder {
	return &testOpenRouterImageRecorder{
		ResponseWriter: writer,
		header:         make(http.Header),
	}
}

func (r *testOpenRouterImageRecorder) Header() http.Header {
	return r.header
}

func (r *testOpenRouterImageRecorder) WriteHeader(statusCode int) {
	if r.Written() {
		return
	}
	r.status = statusCode
}

func (r *testOpenRouterImageRecorder) WriteHeaderNow() {
	if r.status == 0 {
		r.status = http.StatusOK
	}
}

func (r *testOpenRouterImageRecorder) Write(data []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.body.Write(data)
	r.size += n
	return n, err
}

func (r *testOpenRouterImageRecorder) WriteString(data string) (int, error) {
	return r.Write([]byte(data))
}

func (r *testOpenRouterImageRecorder) Written() bool {
	return r.size > 0 || r.status != 0
}

func (r *testOpenRouterImageRecorder) Size() int {
	return r.size
}

func (r *testOpenRouterImageRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}
	return r.status
}

func (r *testOpenRouterImageRecorder) BodyBytes() []byte {
	return r.body.Bytes()
}

func (r *testOpenRouterImageRecorder) ReplaceBody(status int, body []byte) {
	r.body.Reset()
	_, _ = r.body.Write(body)
	r.status = status
	r.size = len(body)
}

func (r *testOpenRouterImageRecorder) Reset() {
	for key := range r.header {
		delete(r.header, key)
	}
	r.body.Reset()
	r.status = 0
	r.size = 0
}
