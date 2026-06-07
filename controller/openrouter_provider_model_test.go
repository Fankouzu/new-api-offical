package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func withOpenRouterProviderRatioConfig(t *testing.T) {
	t.Helper()

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		if key == "ModelRatio" || key == "CompletionRatio" {
			saved[key] = value
		}
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
		model.InvalidatePricingCache()
	})

	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"acme/provider-chat":0.001}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"acme/provider-chat":2}`))
	model.InvalidatePricingCache()
}

func TestListOpenRouterProviderModelsReturnsProviderCatalogShape(t *testing.T) {
	withOpenRouterProviderRatioConfig(t)
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       2001,
		Username: "openrouter-provider-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&model.Model{
		ModelName:   "acme/provider-chat",
		Description: "Provider catalog model",
		Tags:        "or:name=Acme Provider Chat,or:context=64000,or:max_output=4096,or:feature=tools,or:dc=US",
		Status:      1,
	}).Error)
	require.NoError(t, db.Create(&model.Channel{
		Id:     1,
		Type:   constant.ChannelTypeOpenAI,
		Key:    "test-key",
		Status: common.ChannelStatusEnabled,
		Name:   "openai-provider-channel",
		Models: "acme/provider-chat",
		Group:  "default",
	}).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group:     "default",
		Model:     "acme/provider-chat",
		ChannelId: 1,
		Enabled:   true,
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/openrouter/v1/models", nil)

	ListOpenRouterProviderModels(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotContains(t, recorder.Body.String(), `"success"`)

	var payload dto.OpenRouterProviderModelsResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Len(t, payload.Data, 1)
	got := payload.Data[0]
	require.Equal(t, "acme/provider-chat", got.ID)
	require.Equal(t, "Acme Provider Chat", got.Name)
	require.Equal(t, []string{"text"}, got.InputModalities)
	require.Equal(t, []string{"text"}, got.OutputModalities)
	require.Contains(t, got.SupportedFeatures, "tools")
	require.Equal(t, "0.000000002", got.Pricing.Prompt)
	require.Equal(t, "0.000000004", got.Pricing.Completion)
}
