package main

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-sdk/pkg/cli"
	"github.com/spf13/cobra"
)

// config defines the external configuration required for the connector to run.
type config struct {
	cli.BaseConfig `mapstructure:",squash"` // Puts the base config options in the same place as the connector options

	IntegrationKey string `mapstructure:"integration-key"`
	SecretKey      string `mapstructure:"secret-key"`
	ApiHostname    string `mapstructure:"api-hostname"`
}

// validateConfig is run after the configuration is loaded, and should return an error if it isn't valid.
func validateConfig(ctx context.Context, cfg *config) error {
	if cfg.IntegrationKey == "" {
		return fmt.Errorf("integration key is missing")
	}

	if cfg.SecretKey == "" {
		return fmt.Errorf("secret key is missing")
	}

	if cfg.ApiHostname == "" {
		return fmt.Errorf("api host name is missing")
	}

	return nil
}

// cmdFlags sets the cmdFlags required for the connector.
func cmdFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("integration-key", "", "Duo integration key needed to complete the setup to connect to the Duo API. ($BATON_INTEGRATION_KEY)")
	cmd.PersistentFlags().String("secret-key", "", "Duo secret key needed to complete the setup to connect to the Duo API. ($BATON_SECRET_KEY)")
	cmd.PersistentFlags().String("api-hostname", "", "Duo api hostname key needed to complete the setup to connect to the Duo API. ($BATON_API_HOSTNAME)")
}
