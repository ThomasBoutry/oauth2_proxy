package providers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/bitly/go-simplejson"
	"github.com/pusher/oauth2_proxy/api"
)

// AzureProvider represents an Azure based Identity Provider
type AzureProvider struct {
	*ProviderData
	Tenant string
	DomainHint string
}

// NewAzureProvider initiates a new AzureProvider
func NewAzureProvider(p *ProviderData) *AzureProvider {
	p.ProviderName = "Azure"

	if p.ProfileURL == nil || p.ProfileURL.String() == "" {
		p.ProfileURL = &url.URL{
			Scheme:   "https",
			Host:     "graph.windows.net",
			Path:     "/me",
			RawQuery: "api-version=1.6",
		}
	}
	if p.ProtectedResource == nil || p.ProtectedResource.String() == "" {
		p.ProtectedResource = &url.URL{
			Scheme: "https",
			Host:   "graph.windows.net",
		}
	}
	if p.Scope == "" {
		p.Scope = "openid"
	}

	return &AzureProvider{ProviderData: p}
}

// Configure defaults the AzureProvider configuration options
func (p *AzureProvider) Configure(tenant string, domainHint string) {
	p.DomainHint = domainHint
	p.Tenant = tenant
	if tenant == "" {
		p.Tenant = "common"
	}

	if p.LoginURL == nil || p.LoginURL.String() == "" {
		p.LoginURL = &url.URL{
			Scheme: "https",
			Host:   "login.microsoftonline.com",
			Path:   "/" + p.Tenant + "/oauth2/authorize"}
	}
	if p.RedeemURL == nil || p.RedeemURL.String() == "" {
		p.RedeemURL = &url.URL{
			Scheme: "https",
			Host:   "login.microsoftonline.com",
			Path:   "/" + p.Tenant + "/oauth2/token",
		}
	}
}

func getAzureHeader(accessToken string) http.Header {
	header := make(http.Header)
	header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	return header
}

func getEmailFromJSON(json *simplejson.Json) (string, error) {
	var email string
	var err error

	email, err = json.Get("mail").String()

	if err != nil || email == "" {
		otherMails, otherMailsErr := json.Get("otherMails").Array()
		if len(otherMails) > 0 {
			email = otherMails[0].(string)
		}
		err = otherMailsErr
	}

	return email, err
}

// GetEmailAddress returns the Account email address
func (p *AzureProvider) GetEmailAddress(s *SessionState) (string, error) {
	var email string
	var err error

	if s.AccessToken == "" {
		return "", errors.New("missing access token")
	}
	req, err := http.NewRequest("GET", p.ProfileURL.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header = getAzureHeader(s.AccessToken)

	json, err := api.Request(req)

	if err != nil {
		return "", err
	}

	email, err = getEmailFromJSON(json)

	if err == nil && email != "" {
		return email, err
	}

	email, err = json.Get("userPrincipalName").String()

	if err != nil {
		log.Printf("failed making request %s", err)
		return "", err
	}

	if email == "" {
		log.Printf("failed to get email address")
		return "", err
	}

	return email, err
}

// GetLoginURL with typical oauth parameters
func (p *AzureProvider)  GetLoginURL(redirectURI, state string) string {
	var a url.URL
	a = *p.LoginURL
	params, _ := url.ParseQuery(a.RawQuery)
	params.Set("redirect_uri", redirectURI)
	params.Set("approval_prompt", p.ApprovalPrompt)
	if p.DomainHint != "" {
		params.Add("domain_hint", p.DomainHint)
	}
	params.Add("scope", p.Scope)
	params.Set("client_id", p.ClientID)
	params.Set("response_type", "code")
	params.Add("state", state)
	a.RawQuery = params.Encode()
	return a.String()
}


