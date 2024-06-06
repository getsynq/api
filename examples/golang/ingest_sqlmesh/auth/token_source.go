package auth

import (
	"context"
	"fmt"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

type TokenSource interface {
	oauth2.TokenSource
	credentials.PerRPCCredentials
}

func LongLivedTokenSource(longLivedToken string, apiEndpoint string) (TokenSource, error) {
	if apiEndpoint == "" {
		apiEndpoint = "developer.synq.io"
	}

	initialToken, err := obtainToken(apiEndpoint, longLivedToken)
	if err != nil {
		return nil, err
	}

	return oauth.TokenSource{TokenSource: oauth2.ReuseTokenSource(initialToken, &tokenSource{apiEndpoint: apiEndpoint, longLivedToken: longLivedToken})}, nil
}

type tokenSource struct {
	longLivedToken string
	apiEndpoint    string
}

func (t *tokenSource) Token() (*oauth2.Token, error) {
	return obtainToken(t.apiEndpoint, t.longLivedToken)
}

func obtainToken(apiEndpoint string, longLivedToken string) (*oauth2.Token, error) {
	tokenURL := fmt.Sprintf("https://%s/oauth2/token", apiEndpoint)
	conf := oauth2.Config{
		Endpoint: oauth2.Endpoint{
			TokenURL:  tokenURL,
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}
	return conf.PasswordCredentialsToken(context.Background(), "synq", longLivedToken)
}
