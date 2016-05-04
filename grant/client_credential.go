package grant

import (
	"net/http"

	"github.com/lyokato/goidc/log"
	oer "github.com/lyokato/goidc/oauth_error"
	"github.com/lyokato/goidc/scope"
	sd "github.com/lyokato/goidc/service_data"
)

const TypeClientCredentials = "client_credentials"

func ClientCredentials() *GrantHandler {
	return &GrantHandler{
		TypeClientCredentials,
		func(r *http.Request, c sd.ClientInterface,
			sdi sd.ServiceDataInterface, logger log.Logger) (*Response, *oer.OAuthError) {

			uid := c.OwnerUserId()
			if uid < 0 {
				return nil, oer.NewOAuthSimpleError(oer.ErrInvalidGrant)
			}

			scp_req := r.FormValue("scope")
			if scp_req != "" && !c.CanUseScope(scp_req) {
				logger.Info(log.TokenEndpointLog(TypeClientCredentials, log.InvalidScope,
					map[string]string{"scope": scp_req, "client_id": c.Id()},
					"requested scope is not allowed to this client"))
				return nil, oer.NewOAuthSimpleError(oer.ErrInvalidScope)
			}

			info, err := sdi.CreateOrUpdateAuthInfo(uid, c.Id(), scp_req, nil)
			if err != nil {
				if err.Type() == sd.ErrFailed {
					return nil, oer.NewOAuthSimpleError(oer.ErrInvalidGrant)
				} else if err.Type() == sd.ErrUnsupported {
					logger.Error(log.TokenEndpointLog(TypeClientCredentials, log.InterfaceUnsupported,
						map[string]string{"method": "CreateOrUpdateAuthInfo"},
						"the method returns 'unsupported' error."))
					return nil, oer.NewOAuthSimpleError(oer.ErrServerError)
				} else {
					return nil, oer.NewOAuthSimpleError(oer.ErrServerError)
				}
			} else {
				if info == nil {
					logger.Error(log.TokenEndpointLog(TypeClientCredentials, log.InterfaceError,
						map[string]string{"method": "CreateOrUpdateAuthInfo"},
						"the method returns (nil, nil)."))
					return nil, oer.NewOAuthSimpleError(oer.ErrServerError)
				}
			}

			token, err := sdi.CreateAccessToken(info,
				scope.IncludeOfflineAccess(info.Scope()))
			if err != nil {
				if err.Type() == sd.ErrFailed {
					return nil, oer.NewOAuthSimpleError(oer.ErrInvalidGrant)
				} else if err.Type() == sd.ErrUnsupported {
					logger.Error(log.TokenEndpointLog(TypeClientCredentials, log.InterfaceUnsupported,
						map[string]string{"method": "CreateAccessToken"},
						"the method returns 'unsupported' error."))
					return nil, oer.NewOAuthSimpleError(oer.ErrServerError)
				} else {
					return nil, oer.NewOAuthSimpleError(oer.ErrServerError)
				}
			} else {
				if token == nil {
					logger.Error(log.TokenEndpointLog(TypeClientCredentials, log.InterfaceError,
						map[string]string{"method": "CreateAccessToken"},
						"the method returns (nil, nil)."))
					return nil, oer.NewOAuthSimpleError(oer.ErrServerError)
				}
			}

			res := NewResponse(token.AccessToken(), token.AccessTokenExpiresIn())

			scp := info.Scope()
			if scp != "" {
				res.Scope = scp
			}
			rt := token.RefreshToken()
			if rt != "" {
				res.RefreshToken = rt
			}
			return res, nil
		},
	}
}
