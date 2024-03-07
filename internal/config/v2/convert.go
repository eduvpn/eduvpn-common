package v2

import (
	"time"

	"github.com/eduvpn/eduvpn-common/internal/config/v1"
	"github.com/eduvpn/eduvpn-common/types/server"
)

func v1AuthTime(st time.Time, ost time.Time) time.Time {
	// OAuth start time can be zero
	if ost.IsZero() {
		return st
	}
	return ost
}

func convertV1Server(list v1.InstituteServers, iscurrent bool, t server.Type) (map[ServerKey]*Server, *ServerKey) {
	ret := make(map[ServerKey]*Server)
	var lc *ServerKey
	for k, v := range list.Map {
		key := ServerKey{
			T:  t,
			ID: k,
		}
		if iscurrent && k == list.CurrentURL {
			lc = &key
		}
		prfs := v.Base.Profiles.Public()
		prfs.Current = v.Base.Profiles.Current
		ret[key] = &Server{
			Profiles:          prfs,
			LastAuthorizeTime: v1AuthTime(v.Base.StartTime, v.Base.StartTimeOAuth),
			ExpireTime:        v.Base.ExpireTime,
		}
	}
	return ret, lc
}

// FromV1 converts a version 1 state struct into a v2 one
func FromV1(ver1 *v1.V1) *V2 {
	gsrvs := ver1.Servers

	var lc *ServerKey
	cust, glc := convertV1Server(gsrvs.Custom, gsrvs.IsType == v1.CustomServerType, server.TypeCustom)
	if lc == nil {
		lc = glc
	}
	res, glc := convertV1Server(gsrvs.Institute, gsrvs.IsType == v1.InstituteAccessServerType, server.TypeInstituteAccess)
	if lc == nil {
		lc = glc
	}

	for k, v := range cust {
		res[k] = v
	}
	sec := gsrvs.SecureInternetHome
	// if the home organization ID is filled we have secure internet present
	if sec.HomeOrganizationID == "" {
		return &V2{
			Discovery:  ver1.Discovery,
			List:       res,
			LastChosen: lc,
		}
	}
	v, ok := sec.BaseMap[sec.CurrentLocation]
	if v != nil && ok {
		t := ServerKey{
			T:  server.TypeSecureInternet,
			ID: sec.HomeOrganizationID,
		}
		if gsrvs.IsType == v1.SecureInternetServerType {
			lc = &t
		}
		prfs := v.Profiles.Public()
		prfs.Current = v.Profiles.Current
		res[t] = &Server{
			CountryCode:       sec.CurrentLocation,
			Profiles:          prfs,
			LastAuthorizeTime: v1AuthTime(v.StartTime, v.StartTimeOAuth),
			ExpireTime:        v.ExpireTime,
		}
	}
	return &V2{
		Discovery:  ver1.Discovery,
		List:       res,
		LastChosen: lc,
	}
}
