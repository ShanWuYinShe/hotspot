package textutil

import "testing"

func TestEscapeLikePattern(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"abc", "abc"},
		{"100%", "100\\%"},
		{"a_b", "a\\_b"},
		{"back\\slash", "back\\\\slash"},
		{"50%_off\\", "50\\%\\_off\\\\"},
	}
	for _, c := range cases {
		if got := EscapeLikePattern(c.in); got != c.want {
			t.Errorf("EscapeLikePattern(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
