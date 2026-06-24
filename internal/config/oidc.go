package config

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/coreos/go-oidc/v3/oidc"
)

func InitOIDC(cfg *Config) (*oidc.IDTokenVerifier, error) {
	provider, err := oidc.NewProvider(context.Background(), cfg.OIDC.Issuer)
	if err != nil {
		return nil, fmt.Errorf("failed to create oidc provider: %w", err)
	}

	verifier := provider.Verifier(&oidc.Config{
		ClientID: cfg.OIDC.ClientID,
	})

	slog.Info("oidc configuration initialized")
	return verifier, nil
}
