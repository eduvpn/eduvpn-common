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

func convertV1Server(list v1.InstituteServers, iscurrent bool, t server.Type) (map[ServerType]*Server, *ServerType) {
	ret := make(map[ServerType]*Server)
	var lc *ServerType
	for k, v := range list.Map {
		key := ServerType{
			T:  t,
			ID: k,
		}
		if iscurrent && k == list.CurrentURL {
			lc = &key
		}
		prfs := v.Profiles.Public()
		prfs.Current = v.Profiles.Current
		ret[key] = &Server{
			Profiles:          prfs,
			LastAuthorizeTime: v1AuthTime(v.Base.StartTime, v.Base.StartTimeOAuth),
			ExpireTime:        v.Base.ExpireTime,
		}
	}
	return ret, lc
}

func FromV1(ver1 *v1.V1) *V2 {
	gsrvs := ver1.Servers

	var lc *ServerType
	cust, glc := convertV1Server(gsrvs.Custom, gsrvs.IsType == server.TypeCustom, server.TypeCustom)
	if lc == nil {
		lc = glc
	}
	res, glc := convertV1Server(gsrvs.Institute, gsrvs.IsType == server.TypeInstituteAccess, server.TypeInstituteAccess)
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
		t := ServerType{
			T:  server.TypeSecureInternet,
			ID: sec.HomeOrganizationID,
		}
		if gsrvs.IsType == server.TypeSecureInternet {
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
