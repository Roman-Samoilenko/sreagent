package core

import "testing"

func TestExtractFinalAnswer(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain final answer",
			in:   "some agent error\nFinal Answer: The repo has 3 open issues.",
			want: "The repo has 3 open issues.",
		},
		{
			name: "quoted on both sides",
			in:   `parse error: Final Answer: "wrapped in quotes"`,
			want: "wrapped in quotes",
		},
		{
			name: "no marker present",
			in:   "some unrelated error without the marker",
			want: "",
		},
		{
			name: "marker with trailing whitespace",
			in:   "Final Answer:   spaced out answer   ",
			want: "spaced out answer",
		},
		{
			name: "last occurrence wins",
			in:   "Final Answer: first\nsome more text\nFinal Answer: second",
			want: "second",
		},
		{
			name: "empty string",
			in:   "",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractFinalAnswer(tc.in)
			if got != tc.want {
				t.Errorf("extractFinalAnswer(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
