package eduvpn

import (
	"encoding/json"
)

type Server struct {
	BaseURL string
	Endpoints *ServerEndpoints
	OAuth *OAuth
}

type ServerEndpointList struct {
	API           string `json:"api_endpoint"`
	Authorization string `json:"authorization_endpoint"`
	Token         string `json:"token_endpoint"`
}

// Struct that defines the json format for /.well-known/vpn-user-portal"
type ServerEndpoints struct {
	API struct {
		V2 ServerEndpointList `json:"http://eduvpn.org/api#2"`
		V3 ServerEndpointList `json:"http://eduvpn.org/api#3"`
	} `json:"api"`
	V string `json:"v"`
}


func (server *Server) Initialize(url string) error {
	server.BaseURL = url
	endpointsErr := server.GetEndpoints()
	if endpointsErr != nil {
		return endpointsErr
	}
	return nil
}


func (server *Server) GetEndpoints() error {
	url := server.BaseURL + "/.well-known/vpn-user-portal"
	body, bodyErr := HTTPGet(url)

	if bodyErr != nil {
		return bodyErr
	}

	endpoints := &ServerEndpoints{}
	jsonErr := json.Unmarshal(body, &endpoints)

	if jsonErr != nil {
		return jsonErr
	}

	server.Endpoints = endpoints

	return nil
}
