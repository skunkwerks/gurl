package hamac_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"testing"

	"github.com/google/go-cmp/cmp"
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
				Algorithm: hamac.Sha512,
				Header:    "x-hub-signature",
				Secret:    "squirrel"}},
		{name: "fall-through defaults", input: ":x-lol:squirrel",
			want: hamac.Hmac{
				Enabled:   true,
				Algorithm: hamac.Sha256,
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

func TestMacs(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		body      []byte
		algorithm func() hash.Hash
		want      string
	}{
		// generate these hashes
		// printf 'content' | openssl dgst -sha256 -hmac squirrel
		{
			name:      "sha256 test",
			algorithm: sha256.New,
			want:      "82134a1023b182184567609ca9c7dd1c3f0c875fbfff9ad876664f78d5ec2f8d",
		},
		{
			name:      "sha512 test",
			algorithm: sha512.New,
			want:      "f0a6e25b31bccdfcf75ab00918838c2fcf7d5c6c498da23fbf09276f375d0d38d4f18c06ffb3f02e6e4123040b2b6845f96b5afc6b071648d5909e33e4bb430f",
		},
	}

	secret := []byte("squirrel")
	body := []byte("content")

	for _, tc := range testCases {

		mac := hmac.New(tc.algorithm, secret)
		mac.Write(body)
		data := mac.Sum(nil)

		// convert []bytes to string to simplify comparing hex strings
		got := hex.EncodeToString(data)

		if !cmp.Equal(tc.want, got) {
			t.Errorf("%v: diff %v", tc.name, cmp.Diff(tc.want, got))
		}
	}
}
