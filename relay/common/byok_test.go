package common

import "testing"

func TestIsPersonalByokChannel(t *testing.T) {
	tests := []struct {
		name        string
		userId      int
		usingGroup  string
		channelName string
		want        bool
	}{
		{name: "my-keys openai", userId: 7, usingGroup: "u7", channelName: "u7-openai", want: true},
		{name: "my-keys silicon", userId: 12, usingGroup: "u12", channelName: "u12-silicon", want: true},
		{name: "wrong group", userId: 7, usingGroup: "default", channelName: "u7-openai", want: false},
		{name: "shared name in personal group", userId: 7, usingGroup: "u7", channelName: "shared-openai", want: false},
		{name: "other user prefix", userId: 7, usingGroup: "u7", channelName: "u8-openai", want: false},
		{name: "empty provider", userId: 7, usingGroup: "u7", channelName: "u7-", want: false},
		{name: "zero user", userId: 0, usingGroup: "u0", channelName: "u0-openai", want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsPersonalByokChannel(tc.userId, tc.usingGroup, tc.channelName)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
