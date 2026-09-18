package connectivity

import (
	"fmt"
	"os"
)

// envAny returns the first set variable among names. Dotted names exist
// because some consumers inject configuration with dots in the key.
func envAny(names ...string) string {
	for _, n := range names {
		if v := os.Getenv(n); v != "" {
			return v
		}
	}
	return ""
}

// NewFromEnv builds a Client from environment configuration:
//
//	CONNECTIVITY.BASE_URL       API root, e.g. https://api.example.com/v1 (required)
//	CONNECTIVITY.API_TOKEN      static bearer token, or
//	CONNECTIVITY.TOKEN_URL      OpenID token endpoint for client credentials
//	CONNECTIVITY.CLIENT_ID      with
//	CONNECTIVITY.CLIENT_SECRET
//	CONNECTIVITY.BUSINESS_ID    only for staff principals acting for a business
//
// Underscored spellings (CONNECTIVITY_BASE_URL, ...) are accepted too.
func NewFromEnv() (*Client, error) {
	baseURL := envAny("CONNECTIVITY.BASE_URL", "CONNECTIVITY_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("connectivity: CONNECTIVITY.BASE_URL is not set")
	}

	var tokens TokenSource
	if t := envAny("CONNECTIVITY.API_TOKEN", "CONNECTIVITY_API_TOKEN"); t != "" {
		tokens = StaticToken(t)
	} else {
		tokenURL := envAny("CONNECTIVITY.TOKEN_URL", "CONNECTIVITY_TOKEN_URL")
		clientID := envAny("CONNECTIVITY.CLIENT_ID", "CONNECTIVITY_CLIENT_ID")
		clientSecret := envAny("CONNECTIVITY.CLIENT_SECRET", "CONNECTIVITY_CLIENT_SECRET")
		if tokenURL == "" || clientID == "" {
			return nil, fmt.Errorf("connectivity: set CONNECTIVITY.API_TOKEN, or CONNECTIVITY.TOKEN_URL and CONNECTIVITY.CLIENT_ID/CLIENT_SECRET")
		}
		tokens = NewClientCredentialsTokenSource(tokenURL, clientID, clientSecret)
	}

	var opts []Option
	if biz := envAny("CONNECTIVITY.BUSINESS_ID", "CONNECTIVITY_BUSINESS_ID"); biz != "" {
		opts = append(opts, WithBusinessID(biz))
	}
	return New(baseURL, tokens, opts...), nil
}
