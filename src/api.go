package eduvpn

import (
	"net/http"
)

func (eduvpn *VPNState) APIAuthenticatedGet(endpoint string) (string, error) {
	url := eduvpn.Server.Endpoints.API.V3.API + endpoint

	// Ensure we have non-expired tokens
	oauthErr := eduvpn.EnsureTokensOAuth()

	if oauthErr != nil {
		return "", oauthErr
	}

	headers := &http.Header{"Authorization": {"Bearer " + eduvpn.Server.OAuth.Token.Access}}
	body, bodyErr := HTTPGetWithOptionalParams(url, &HTTPOptionalParams{Headers: headers})
	if bodyErr != nil {
		return "", bodyErr
	}
	return string(body), nil
}
