package relay

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	taskkie "github.com/QuantumNous/new-api/relay/channel/task/kie"
	tasksub2apiasync "github.com/QuantumNous/new-api/relay/channel/task/sub2api_async"
	tasktencentvod "github.com/QuantumNous/new-api/relay/channel/task/tencentvod"
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
	if len(adaptor.GetModelList()) == 0 {
		t.Fatal("expected Sub2API-async model list")
	}
}

func TestGetTaskAdaptorReturnsTencentVODAIGCAdaptor(t *testing.T) {
	adaptor := GetTaskAdaptor(constant.TaskPlatform(strconv.Itoa(constant.ChannelTypeTencentVODAIGC)))
	if adaptor == nil {
		t.Fatal("expected Tencent VOD AIGC task adaptor")
	}
	if adaptor.GetChannelName() != tasktencentvod.ChannelName {
		t.Fatalf("channel name = %q", adaptor.GetChannelName())
	}
	if len(adaptor.GetModelList()) == 0 {
		t.Fatal("expected Tencent VOD AIGC model list")
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
