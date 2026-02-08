package config

import (
	"github.com/conductorone/baton-sdk/pkg/field"
)

var (
	IntegrationKey = field.StringField(
		"integration-key",
		field.WithRequired(true),
		field.WithIsSecret(true),
		field.WithDisplayName("Integration Key"),
		field.WithDescription("Duo integration key needed to complete the setup to connect to the Duo API."),
	)

	SecretKey = field.StringField(
		"secret-key",
		field.WithRequired(true),
		field.WithIsSecret(true),
		field.WithDisplayName("Secret Key"),
		field.WithDescription("Duo secret key needed to complete the setup to connect to the Duo API."),
	)

	ApiHostname = field.StringField(
		"api-hostname",
		field.WithRequired(true),
		field.WithIsSecret(true),
		field.WithDisplayName("API Hostname"),
		field.WithDescription("Duo api hostname key needed to complete the setup to connect to the Duo API."),
	)

	BaseURLField = field.StringField(
		"base-url",
		field.WithDescription("Override the Duo API URL (for testing)"),
	)

	ConfigurationFields = []field.SchemaField{
		IntegrationKey,
		SecretKey,
		ApiHostname,
		BaseURLField,
	}
)

//go:generate go run ./gen
var Config = field.NewConfiguration(
	ConfigurationFields,
	field.WithConnectorDisplayName("Duo"),
	field.WithHelpUrl("/docs/baton/duo"),
	field.WithIconUrl("/static/app-icons/duo.svg"),
)
