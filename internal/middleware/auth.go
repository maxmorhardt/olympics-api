package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/maxmorhardt/olympics-api/internal/model"
	"github.com/maxmorhardt/olympics-api/internal/util"
)

const authErrorMessage = "Authentication required. Please log in to continue"

var errClaimsParse = errors.New("claims parse failed")

type TokenVerifier interface {
	Verify(ctx context.Context, token string) (*model.Claims, error)
}

type oidcTokenVerifier struct {
	verifier *oidc.IDTokenVerifier
}

func NewOIDCTokenVerifier(verifier *oidc.IDTokenVerifier) TokenVerifier {
	return &oidcTokenVerifier{verifier: verifier}
}

func (v *oidcTokenVerifier) Verify(ctx context.Context, token string) (*model.Claims, error) {
	idToken, err := v.verifier.Verify(ctx, token)
	if err != nil {
		return nil, err
	}

	claims := &model.Claims{}
	if err := idToken.Claims(claims); err != nil {
		return nil, fmt.Errorf("%w: %w", errClaimsParse, err)
	}

	return claims, nil
}

func AuthMiddleware(verifier TokenVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := util.LoggerFromGinContext(c)

		// extract bearer token from the authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			log.Warn("missing authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, authErrorMessage, c))
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == "" {
			log.Warn("empty bearer token")
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, authErrorMessage, c))
			return
		}

		// verify token and extract claims via the injected verifier
		claims, err := verifier.Verify(c.Request.Context(), token)
		if err != nil {
			log.Warn("failed to verify token", "error", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, model.NewAPIError(http.StatusUnauthorized, authErrorMessage, c))
			return
		}

		// store user and claims in context, and tag the logger with the user
		util.SetGinContextValue(c, model.UserKey, claims.Username)
		util.SetGinContextValue(c, model.ClaimsKey, claims)
		util.SetGinContextValue(c, model.LoggerKey, log.With("user", claims.Username))

		c.Next()
	}
}
