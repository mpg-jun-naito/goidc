package grant

import (
	"fmt"
	"net/http"

	"github.com/lyokato/goidc/id_token"

	"github.com/lyokato/goidc/scope"

	"github.com/lyokato/goidc/log"
	oer "github.com/lyokato/goidc/oauth_error"
	"github.com/lyokato/goidc/pkce"
	sd "github.com/lyokato/goidc/service_data"
)

const TypeAuthorizationCode = "authorization_code"

func AuthorizationCode() *GrantHandler {
	return &GrantHandler{
		TypeAuthorizationCode,
		func(r *http.Request, c sd.ClientInterface,
			sdi sd.ServiceDataInterface, logger log.Logger) (*Response, *oer.OAuthError) {

			uri := r.FormValue("redirect_uri")
			if uri == "" {
				return nil, oer.NewOAuthError(oer.ErrInvalidRequest,
					"missing 'redirect_uri' parameter")
			}
			code := r.FormValue("code")
			if code == "" {
				return nil, oer.NewOAuthError(oer.ErrInvalidRequest,
					"missing 'code' parameter")
			}
			info, err := sdi.FindAuthInfoByCode(code)
			if err != nil {
				if err.Type() == sd.ErrFailed {
					return nil, oer.NewOAuthSimpleError(oer.ErrInvalidGrant)
				} else if err.Type() == sd.ErrUnsupported {
					logger.Warnf("[goidc.TokenEndpoint:%s] <ServerError:InterfaceUnsupported:%s>: the method returns 'unsupported' error.",
						TypeAuthorizationCode, "FindAuthInfoByCode")
					return nil, oer.NewOAuthSimpleError(oer.ErrServerError)
				} else {
					return nil, oer.NewOAuthSimpleError(oer.ErrServerError)
				}
			} else {
				if info == nil {
					logger.Warnf("[goidc.TokenEndpoint:%s] <ServerError:InterfaceError:%s>: the method returns (nil, nil).",
						TypeAuthorizationCode, "FindAuthInfoByCode")
					return nil, oer.NewOAuthSimpleError(oer.ErrServerError)
				}
			}
			if info.ClientId() != c.Id() {
				logger.Infof("[goidc.TokenEndpoint:%s] <AuthInfoConditionMismatch:%s>: 'client_id' mismatch.",
					TypeAuthorizationCode, c.Id())
				return nil, oer.NewOAuthSimpleError(oer.ErrInvalidGrant)
			}
			if info.RedirectURI() != uri {
				logger.Infof("[goidc.TokenEndpoint:%s] <AuthInfoConditionMismatch:%s>: 'redirect_uri' mismatch.",
					TypeAuthorizationCode, c.Id())
				return nil, oer.NewOAuthError(oer.ErrInvalidGrant,
					fmt.Sprintf("indicated 'redirect_uri' (%s) is not allowed for this client", uri))
			}

			// RFC7636: OAuth PKCE Extension
			// https://tools.ietf.org/html/rfc7636
			cv := info.CodeVerifier()
			if cv != "" {
				cm := r.FormValue("code_challenge_method")
				if cm == "" {
					return nil, oer.NewOAuthError(oer.ErrInvalidRequest,
						"missing 'code_challenge_method' parameter")
				}
				cc := r.FormValue("code_challenge")
				if cc == "" {
					return nil, oer.NewOAuthError(oer.ErrInvalidRequest,
						"missing 'code_challenge' parameter")
				}
				verifier, err := pkce.FindVerifierByMethod(cm)
				if err != nil {
					return nil, oer.NewOAuthError(oer.ErrInvalidRequest,
						fmt.Sprintf("unsupported 'code_challenge_method': '%s'", cm))
				}
				if !verifier.Verify(cc, cv) {
					return nil, oer.NewOAuthError(oer.ErrInvalidGrant,
						fmt.Sprintf("invalid 'code_challenge': '%s'", cc))
				}
			}

			token, err := sdi.CreateAccessToken(info,
				scope.IncludeOfflineAccess(info.Scope()))
			if err != nil {
				if err.Type() == sd.ErrFailed {
					return nil, oer.NewOAuthSimpleError(oer.ErrInvalidGrant)
				} else if err.Type() == sd.ErrUnsupported {
					logger.Warnf("[goidc.TokenEndpoint:%s] <ServerError:InterfaceUnsupported:%s>: the method returns 'unsupported' error.",
						TypeAuthorizationCode, "CreateAccessToken")
					return nil, oer.NewOAuthSimpleError(oer.ErrServerError)
				} else {
					return nil, oer.NewOAuthSimpleError(oer.ErrServerError)
				}
			} else {
				if token == nil {
					logger.Warnf("[goidc.TokenEndpoint:%s] <ServerError:InterfaceError:%s>: the method returns (nil, nil).",
						TypeAuthorizationCode, "CreateAccessToken")
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

			if scope.IncludeOpenID(scp) {
				idt, err := id_token.Gen(c.IdTokenAlg(), c.IdTokenKey(), c.IdTokenKeyId(), sdi.Issure(),
					info.ClientId(), info.Subject(), info.Nonce(), info.IDTokenExpiresIn(), info.AuthorizedAt())
				if err != nil {
					return nil, oer.NewOAuthSimpleError(oer.ErrServerError)
				} else {
					res.IdToken = idt
				}
			}
			return res, nil
		},
	}
}
