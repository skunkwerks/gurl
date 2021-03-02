package hamac_test

import (
	"testing"

	"github.com/skunkwerks/gurl/hamac"
)

func TestFlags(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name  string
		input string
		want  hamac.Hmac
	}{
		{name: "empty string ''", input: "", want: hamac.Hmac{Enabled: false}},
		{name: "insufficient fields", input: "a:h", want: hamac.Hmac{Enabled: false}},
		{name: "missing secret", input: "a:h::", want: hamac.Hmac{Enabled: false}},
		{name: "excessive fields", input: "a:h:s:garbage", want: hamac.Hmac{Enabled: false}},
		{name: "plain example", input: "sha512:x-hub-signature:squirrel",
			want: hamac.Hmac{
				Enabled:   true,
				Algorithm: "sha512",
				Header:    "x-hub-signature",
				Secret:    "squirrel"}},
		{name: "fall-through defaults", input: ":x-lol:squirrel",
			want: hamac.Hmac{
				Enabled:   true,
				Algorithm: "sha256",
				Header:    "x-lol",
				Secret:    "squirrel"}},
	}

	for _, tc := range testCases {
		got := hamac.New(tc.input)
		if tc.want != got {
			t.Errorf("%v: wanted %v, got %v", tc.name, tc.want, got)
		}
	}
}
