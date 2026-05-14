package relay

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	taskkie "github.com/QuantumNous/new-api/relay/channel/task/kie"
	tasksub2apiasync "github.com/QuantumNous/new-api/relay/channel/task/sub2api_async"
)

func TestGetTaskAdaptorReturnsKieAdaptor(t *testing.T) {
	adaptor := GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeKieAI)))
	if adaptor == nil {
		t.Fatal("expected KieAI task adaptor")
	}
	if adaptor.GetChannelName() != taskkie.ChannelName {
		t.Fatalf("channel name = %q", adaptor.GetChannelName())
	}
	if len(adaptor.GetModelList()) == 0 {
		t.Fatal("expected default Kie model list")
	}
}

func TestGetTaskAdaptorReturnsSub2APIAsyncAdaptor(t *testing.T) {
	adaptor := GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeSub2APIAsync)))
	if adaptor == nil {
		t.Fatal("expected Sub2API-async task adaptor")
	}
	if adaptor.GetChannelName() != tasksub2apiasync.ChannelName {
		t.Fatalf("channel name = %q", adaptor.GetChannelName())
	}
	if len(adaptor.GetModelList()) != 2 {
		t.Fatalf("expected two Sub2API-async models, got %d", len(adaptor.GetModelList()))
	}
}

func TestKieFallbackTaskModelsAreDetected(t *testing.T) {
	platform := constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeKieAI))
	if !isKieFallbackTaskModel("59_generate", platform, constant.TaskActionGenerate) {
		t.Fatal("expected numeric task fallback model to be treated as Kie default")
	}
	if !isKieFallbackTaskModel("dall-e", platform, constant.TaskActionGenerate) {
		t.Fatal("expected image fallback model to be treated as Kie default")
	}
	if isKieFallbackTaskModel(taskkie.ModelSeedance2, platform, constant.TaskActionGenerate) {
		t.Fatal("expected concrete Kie model to stay unchanged")
	}
}
