package assertion

import (
	"fmt"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/lyokato/goidc/log"
	oer "github.com/lyokato/goidc/oauth_error"
	sd "github.com/lyokato/goidc/service_data"
)

func HandleAssertionError(a string, t *jwt.Token, jwt_err error,
	gt string, c sd.ClientInterface, sdi sd.ServiceDataInterface,
	logger log.Logger) *oer.OAuthError {

	if jwt_err != nil {

		ve := jwt_err.(*jwt.ValidationError)

		if ve.Errors&jwt.ValidationErrorUnverifiable == jwt.ValidationErrorUnverifiable {

			// - invalid alg
			// - no key func
			// - key func returns err
			if inner, ok := ve.Inner.(*oer.OAuthError); ok {

				logger.Debug(log.TokenEndpointLog(gt,
					log.AssertionConditionMismatch,
					map[string]string{"assertion": a},
					"'assertion' unverifiable"))

				return inner
			} else {

				return oer.NewOAuthError(oer.ErrInvalidGrant,
					"assertion unverifiable")
			}
		}

		if ve.Errors&jwt.ValidationErrorMalformed == jwt.ValidationErrorMalformed {

			logger.Debug(log.TokenEndpointLog(gt,
				log.AssertionConditionMismatch,
				map[string]string{"assertion": a, "client_id": c.Id()},
				"invalid 'assertion' format"))

			return oer.NewOAuthError(oer.ErrInvalidGrant,
				"invalid assertion format")
		}

		if ve.Errors&jwt.ValidationErrorSignatureInvalid == jwt.ValidationErrorSignatureInvalid {

			logger.Info(log.TokenEndpointLog(gt,
				log.AssertionConditionMismatch,
				map[string]string{"assertion": a, "client_id": c.Id()},
				"invalid 'assertion' signature"))

			return oer.NewOAuthError(oer.ErrInvalidGrant,
				"invalid assertion signature")

		}

		if ve.Errors&jwt.ValidationErrorExpired == jwt.ValidationErrorExpired {

			logger.Info(log.TokenEndpointLog(gt,
				log.AssertionConditionMismatch,
				map[string]string{"assertion": a, "client_id": c.Id()},
				"assertion expired"))

			return oer.NewOAuthError(oer.ErrInvalidGrant,
				"assertion expired")
		}

		if ve.Errors&jwt.ValidationErrorNotValidYet == jwt.ValidationErrorNotValidYet {

			logger.Info(log.TokenEndpointLog(gt,
				log.AssertionConditionMismatch,
				map[string]string{"assertion": a, "client_id": c.Id()},
				"assertion not valid yet"))

			return oer.NewOAuthError(oer.ErrInvalidGrant,
				"assertion not valid yet")
		}

		// unknown error type
		logger.Warn(log.TokenEndpointLog(gt,
			log.AssertionConditionMismatch,
			map[string]string{"assertion": a, "client_id": c.Id()},
			"unknown 'assertion' validation failure"))

		return oer.NewOAuthError(oer.ErrInvalidGrant,
			"invalid assertion")
	}

	if !t.Valid {

		// must not come here
		logger.Warn(log.TokenEndpointLog(gt,
			log.AssertionConditionMismatch,
			map[string]string{"assertion": a, "client_id": c.Id()},
			"invalid 'assertion' signature"))

		return oer.NewOAuthError(oer.ErrInvalidGrant,
			"invalid assertion signature")
	}

	// MUST(exp) error
	// MAY(iat) reject if too far past
	// MAY(jti)

	aud, ok := t.Claims["aud"].(string)
	if !ok {

		logger.Debug(log.TokenEndpointLog(gt,
			log.MissingParam,
			map[string]string{"param": "aud", "client_id": c.Id()},
			"'aud' not found in assertion"))

		return oer.NewOAuthError(oer.ErrInvalidRequest,
			"'aud' parameter not found in assertion")
	}

	service := sdi.Issuer()
	if service == "" {

		logger.Error(log.TokenEndpointLog(gt,
			log.InterfaceUnsupported,
			map[string]string{"method": "Issure"},
			"the method returns 'unsupported' error."))

		return oer.NewOAuthSimpleError(oer.ErrServerError)
	}

	if aud != service {

		logger.Info(log.TokenEndpointLog(gt,
			log.AssertionConditionMismatch,
			map[string]string{"assertion": a, "client_id": c.Id()},
			"invalid 'aud'"))

		return oer.NewOAuthError(oer.ErrInvalidGrant,
			fmt.Sprintf("invalid 'aud' parameter '%s' in assertion", aud))
	}

	return nil
}
