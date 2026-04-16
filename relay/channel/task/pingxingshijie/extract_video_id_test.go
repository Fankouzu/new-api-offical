package pingxingshijie

import "testing"

func TestExtractVideoCreateTaskID(t *testing.T) {
	cases := []struct {
		name string
		json string
		want string
	}{
		{"top_id", `{"id":"cgt-a"}`, "cgt-a"},
		{"top_task_id", `{"task_id":"cgt-b"}`, "cgt-b"},
		{"data_id", `{"data":{"id":"cgt-c"}}`, "cgt-c"},
		{"data_task_id", `{"data":{"task_id":"cgt-d"}}`, "cgt-d"},
		{"nested_data", `{"data":{"data":{"id":"cgt-e"}}}`, "cgt-e"},
		{"Result_wrapper", `{"Result":{"id":"cgt-f"}}`, "cgt-f"},
		{"data_string", `{"code":0,"msg":"ok","data":"cgt-20260317165706-4lpxk"}`, "cgt-20260317165706-4lpxk"},
		{"deep_task", `{"response":{"task":{"id":"cgt-deep-20260101120000-abc12"}}}`, "cgt-deep-20260101120000-abc12"},
		{"ignores_msg_ok", `{"code":0,"msg":"ok","data":{"id":"cgt-from-data"}}`, "cgt-from-data"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractVideoCreateTaskID([]byte(tc.json)); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
