package hamac

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"regexp"
	"strings"
)

// restricted set of algorithms
type Algorithm int

const (
	Sha1 Algorithm = iota
	Sha256
	Sha512
)

// struct to hold validated hmac parameters and an overall enabled flag
type Hmac struct {
	Enabled   bool
	Algorithm Algorithm
	Header    string
	Secret    string
}

// used to check if slice only contains legitimate RFC-style characters
var validHeader = regexp.MustCompile(`(?i)^x-[a-z0-9_-]+$`)

// New hmac will be enabled if all parameters are present and valid
func New(input string) Hmac {
	params := strings.Split(input, ":")
	if len(params) != 3 {
		return Hmac{Enabled: false}
	}

	// got 3 params, let's see if they are usable or not
	alg := params[0]
	header := params[1]
	secret := params[2]

	hmac := Hmac{Enabled: true}

	switch alg {
	case "sha1":
		hmac.Algorithm = Sha1
	case "sha512":
		hmac.Algorithm = Sha512
	default:
		hmac.Algorithm = Sha256
	}

	if validHeader.MatchString(header) {
		hmac.Header = header
	} else {
		hmac.Enabled = false
		return hmac
	}

	if len(secret) > 0 {
		hmac.Secret = secret
	} else {
		hmac.Enabled = false
	}
	return hmac
}

// Sign takes an Hmac (with secret, algorithm, and expected header), and
// a body, and returns an HTTP header, similar to GitHub and GitLab
// HMACs for authenticating the body, via HTTP header envelope.
// see https://tools.ietf.org/html/draft-cavage-http-signatures-10 or a
// later revision for more details on the proposed RFC specification.
func Sign(mac Hmac, body []byte) []byte {

	if !mac.Enabled {
		return []byte("")
	}

	var fn func() hash.Hash
	var alg string

	switch mac.Algorithm {
	case Sha1:
		fn = sha1.New
		alg = "sha1"
	case Sha512:
		fn = sha512.New
		alg = "sha512"
	default:
		fn = sha256.New
		alg = "sha256"
	}

	macFn := hmac.New(fn, []byte(mac.Secret))
	macFn.Write(body)
	hash := hex.EncodeToString(macFn.Sum(nil))

	// build required header by appending strings together
	httpHeader := strings.Join(
		[]string{
			alg,
			"=",
			hash},
		"")

	return []byte(httpHeader)
}
