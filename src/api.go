package eduvpn

import (
	"net/http"
)

func (eduvpn *VPNState) APIAuthenticatedInfo() (string, error) {
	url := eduvpn.Server.Endpoints.API.V3.API + "/info"

	headers := &http.Header{"Authorization": {"Bearer " + eduvpn.Server.OAuth.Token.Access}}
	body, bodyErr := HTTPGetWithOptionalParams(url, &HTTPOptionalParams{Headers: headers})
	if bodyErr != nil {
		return "", bodyErr
	}
	return string(body), nil
}
