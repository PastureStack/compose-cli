package main

import "testing"

func TestTraditionalChineseOperatorMessage(t *testing.T) {
	if got := operatorMessage("zh-TW", "ready"); got != "PastureStack Compose 命令列工具已就緒" {
		t.Fatalf("unexpected zh-TW message: %q", got)
	}
}

func TestRequestsVersion(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "long flag", args: []string{"compose-executor", "--version"}, want: true},
		{name: "short flag", args: []string{"compose-executor", "-v"}, want: true},
		{name: "normal start", args: []string{"compose-executor"}, want: false},
		{name: "unknown flag", args: []string{"compose-executor", "--help"}, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := requestsVersion(test.args); got != test.want {
				t.Fatalf("requestsVersion(%q) = %t, want %t", test.args, got, test.want)
			}
		})
	}
}
