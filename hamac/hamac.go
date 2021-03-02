package hamac

import (
	"regexp"
	"strings"
)

// struct to hold validated hmac parameters and an overall enabled flag
type Hmac struct {
	Enabled   bool
	Algorithm string
	Header    string
	Secret    string
}

// used to check if slice only contains legitimate RFC-style characters
var validHeader = regexp.MustCompile(`(?i)^x-[a-z0-9_-]+$`)

// hmac will be enabled if all parameters are present and valid
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
		hmac.Algorithm = alg
	case "sha512":
		hmac.Algorithm = alg
	default:
		hmac.Algorithm = "sha256"
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
