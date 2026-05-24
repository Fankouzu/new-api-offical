package common

import (
	"testing"

	"github.com/QuantumNous/new-api/constant"
)

func TestChannelType2APITypeMapsSub2APIAsyncToOpenAI(t *testing.T) {
	apiType, ok := ChannelType2APIType(constant.ChannelTypeSub2APIAsync)
	if !ok {
		t.Fatal("expected Sub2API-async channel type to be known")
	}
	if apiType != constant.APITypeOpenAI {
		t.Fatalf("api type = %d, want %d", apiType, constant.APITypeOpenAI)
	}
}
