package volcengine

import (
	"testing"

	channelconstant "github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
)

func TestPingXingShiJieImageGenerationUsesDocumentedV2Endpoint(t *testing.T) {
	a := &Adaptor{}
	info := &relaycommon.RelayInfo{
		RelayMode: relayconstant.RelayModeImagesGenerations,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    channelconstant.ChannelTypePingXingShiJie,
			ChannelBaseUrl: "https://api.pingxingshijie.cn",
		},
	}

	got, err := a.GetRequestURL(info)
	if err != nil {
		t.Fatal(err)
	}
	const want = "https://api.pingxingshijie.cn/v2/image/generations"
	if got != want {
		t.Fatalf("GetRequestURL() = %q, want %q", got, want)
	}
}
