package xiaohongshu

import "testing"

func TestIsPlaceholderNickname(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "empty", in: "", want: true},
		{name: "spaces", in: "   ", want: true},
		{name: "wo", in: "我", want: true},
		{name: "wode", in: "我的", want: true},
		{name: "me", in: "me", want: true},
		{name: "valid", in: "ZIIKOO TALK", want: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := isPlaceholderNickname(tc.in)
			if got != tc.want {
				t.Fatalf("isPlaceholderNickname(%q)=%v want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestSelectPreferredNickname(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		base    string
		profile string
		title   string
		want    string
	}{
		{
			name:    "use base when valid",
			base:    "Alice",
			profile: "Bob",
			title:   "Bob - 小红书",
			want:    "Alice",
		},
		{
			name:    "fallback to profile when base placeholder",
			base:    "我",
			profile: "ZIIKOO TALK",
			title:   "ZIIKOO TALK - 小红书",
			want:    "ZIIKOO TALK",
		},
		{
			name:    "fallback to title when profile empty",
			base:    "我的",
			profile: "",
			title:   "ZIIKOO TALK - 小红书",
			want:    "ZIIKOO TALK",
		},
		{
			name:    "empty when no valid source",
			base:    "我",
			profile: "",
			title:   "",
			want:    "",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := selectPreferredNickname(tc.base, tc.profile, tc.title)
			if got != tc.want {
				t.Fatalf("selectPreferredNickname(%q,%q,%q)=%q want %q", tc.base, tc.profile, tc.title, got, tc.want)
			}
		})
	}
}
