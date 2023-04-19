package connector

import (
	"context"
	"fmt"

	"github.com/ConductorOne/baton-duo/pkg/duo"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/connectorbuilder"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
)

var (
	resourceTypeUser = &v2.ResourceType{
		Id:          "user",
		DisplayName: "User",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
	}
	resourceTypeGroup = &v2.ResourceType{
		Id:          "group",
		DisplayName: "Group",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_GROUP,
		},
	}
	resourceTypeAdmin = &v2.ResourceType{
		Id:          "admin",
		DisplayName: "Admin",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_USER,
		},
	}
	resourceTypeAccount = &v2.ResourceType{
		Id:          "account",
		DisplayName: "Account",
	}
	resourceTypeRole = &v2.ResourceType{
		Id:          "role",
		DisplayName: "Role",
		Traits: []v2.ResourceType_Trait{
			v2.ResourceType_TRAIT_ROLE,
		},
	}
)

type Duo struct {
	client         *duo.Client
	integrationKey string
}

func (d *Duo) ResourceSyncers(ctx context.Context) []connectorbuilder.ResourceSyncer {
	return []connectorbuilder.ResourceSyncer{
		userBuilder(d.client),
		groupBuilder(d.client),
		adminBuilder(d.client),
		accountBuilder(d.client, d.integrationKey),
		roleBuilder(d.client),
	}
}

// Metadata returns metadata about the connector.
func (d *Duo) Metadata(ctx context.Context) (*v2.ConnectorMetadata, error) {
	return &v2.ConnectorMetadata{
		DisplayName: "Duo",
	}, nil
}

// Validate hits the Duo API to validate API credentials.
func (d *Duo) Validate(ctx context.Context) (annotations.Annotations, error) {
	_, err := d.client.GetIntegration(ctx)
	if err != nil {
		return nil, fmt.Errorf("error fetching integration by credentials: %w", err)
	}
	return nil, nil
}

// New returns the Duo connector.
func New(ctx context.Context, integrationKey string, secretKey string, apiHostname string) (*Duo, error) {
	httpClient, err := uhttp.NewClient(ctx, uhttp.WithLogger(true, ctxzap.Extract(ctx)))
	if err != nil {
		return nil, err
	}

	return &Duo{
		client:         duo.NewClient(integrationKey, secretKey, apiHostname, httpClient),
		integrationKey: integrationKey,
	}, nil
}
