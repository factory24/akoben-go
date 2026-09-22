package connectivity

import (
	"fmt"
	"os"
	"strings"
)

// env reads one setting, e.g. env("CELLULAR.API_KEY"). The name is AKOBEN.<key>;
// the underscored spelling exists because some consumers cannot inject dots,
// and the CONNECTIVITY prefix is the old name, still read so nothing deployed
// breaks.
func env(key string) string {
	underscored := strings.ReplaceAll(key, ".", "_")
	for _, name := range []string{"AKOBEN." + key, "AKOBEN_" + underscored, "CONNECTIVITY." + key, "CONNECTIVITY_" + underscored} {
		if v := os.Getenv(name); v != "" {
			return v
		}
	}
	return ""
}

// NewFromEnv builds a Client from environment configuration:
//
//	AKOBEN.BASE_URL       API root, e.g. https://api.example.com/v1 (required)
//	AKOBEN.API_TOKEN      static bearer token, or
//	AKOBEN.TOKEN_URL      OpenID token endpoint for client credentials
//	AKOBEN.CLIENT_ID      with
//	AKOBEN.CLIENT_SECRET
//	AKOBEN.BUSINESS_ID    only for staff principals acting for a business
//
// Underscored spellings (AKOBEN_BASE_URL, ...) are accepted too, and so is the
// old CONNECTIVITY prefix.
func NewFromEnv() (*Client, error) {
	baseURL := env("BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("connectivity: AKOBEN.BASE_URL is not set")
	}

	var tokens TokenSource
	if t := env("API_TOKEN"); t != "" {
		tokens = StaticToken(t)
	} else {
		tokenURL := env("TOKEN_URL")
		clientID := env("CLIENT_ID")
		clientSecret := env("CLIENT_SECRET")
		if tokenURL == "" || clientID == "" {
			return nil, fmt.Errorf("connectivity: set AKOBEN.API_TOKEN, or AKOBEN.TOKEN_URL and AKOBEN.CLIENT_ID/CLIENT_SECRET")
		}
		tokens = NewClientCredentialsTokenSource(tokenURL, clientID, clientSecret)
	}

	var opts []Option
	if biz := env("BUSINESS_ID"); biz != "" {
		opts = append(opts, WithBusinessID(biz))
	}
	return New(baseURL, tokens, opts...), nil
}
