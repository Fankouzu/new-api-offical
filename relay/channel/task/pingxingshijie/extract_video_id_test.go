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
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractVideoCreateTaskID([]byte(tc.json)); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}
