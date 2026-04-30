package pingxingshijie

import "testing"

func TestModelListMatchesDocumentedPingXingShiJieModels(t *testing.T) {
	want := []string{
		"pingxingshijie-asset",
		"doubao-seedance-1-0-pro-fast-251015",
		"doubao-seedance-1-5-pro-251215",
		"doubao-seedance-2-0-fast-260128",
		"doubao-seedance-2-0-260128",
		"doubao-seedream-5-0-260128",
		"doubao-seedream-4-5-251128",
		"doubao-seedream-4-0-250828",
	}

	if len(ModelList) != len(want) {
		t.Fatalf("ModelList length = %d, want %d: %#v", len(ModelList), len(want), ModelList)
	}
	for i, model := range want {
		if ModelList[i] != model {
			t.Fatalf("ModelList[%d] = %q, want %q; full list: %#v", i, ModelList[i], model, ModelList)
		}
	}
}
