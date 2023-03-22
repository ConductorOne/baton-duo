package connector

import (
	"context"
	"fmt"

	"github.com/ConductorOne/baton-duo/pkg/duo"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	grant "github.com/conductorone/baton-sdk/pkg/types/grant"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
)

var roles = map[string]string{
	"Owner":               "owner",
	"Administrator":       "administrator",
	"Application Manager": "application manager",
	"User Manager":        "user manager",
	"Help Desk":           "help desk",
	"Billing":             "billing",
	"Phishing Manager":    "phishing manager",
	"Read-only":           "readonly",
}

type accountResourceType struct {
	resourceType   *v2.ResourceType
	client         *duo.Client
	integrationKey string
}

func (o *accountResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// Create a new connector resource for a Duo account.
func accountResource(ctx context.Context, account duo.Account, integrationKey string) (*v2.Resource, error) {
	accountOptions := []rs.ResourceOption{
		rs.WithAnnotation(
			&v2.ChildResourceType{ResourceTypeId: resourceTypeUser.Id},
			&v2.ChildResourceType{ResourceTypeId: resourceTypeGroup.Id},
			&v2.ChildResourceType{ResourceTypeId: resourceTypeAdmin.Id},
		),
	}
	ret, err := rs.NewResource(
		account.Name,
		resourceTypeAccount,
		integrationKey,
		accountOptions...,
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *accountResourceType) List(ctx context.Context, _ *v2.ResourceId, _ *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	account, err := o.client.GetAccount(ctx)
	if err != nil {
		return nil, "", nil, fmt.Errorf("duo-connector: failed to list an account: %w", err)
	}

	var rv []*v2.Resource
	ar, err := accountResource(ctx, account, o.integrationKey)
	if err != nil {
		return nil, "", nil, err
	}
	rv = append(rv, ar)

	return rv, "", nil, nil
}

func (o *accountResourceType) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	for _, role := range roles {
		permissionOptions := []ent.EntitlementOption{
			ent.WithGrantableTo(resourceTypeAdmin),
			ent.WithDescription(fmt.Sprintf("Role in %s Duo account", resource.DisplayName)),
			ent.WithDisplayName(fmt.Sprintf("%s Account %s", resource.DisplayName, role)),
		}

		permissionEn := ent.NewPermissionEntitlement(resource, role, permissionOptions...)
		rv = append(rv, permissionEn)
	}
	return rv, "", nil, nil
}

func (o *accountResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var pageToken string
	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeAccount.Id})
	if err != nil {
		return nil, "", nil, err
	}
	admins, offset, err := o.client.GetAdmins(ctx, bag.PageToken())
	if err != nil {
		return nil, "", nil, err
	}
	if offset != "" {
		pageToken, err = bag.NextToken(offset)
		if err != nil {
			return nil, "", nil, err
		}
	}

	var rv []*v2.Grant
	for _, admin := range admins {
		roleName, ok := roles[admin.Role]
		if !ok {
			ctxzap.Extract(ctx).Warn("Unknown Duo Role Name, skipping",
				zap.String("role_name", admin.Role),
				zap.String("user", admin.Name),
			)
			continue
		}
		adminCopy := admin
		ar, err := adminResource(ctx, &adminCopy, resource.Id)
		if err != nil {
			return nil, "", nil, err
		}

		permissionGrant := grant.NewGrant(resource, roleName, ar.Id)
		rv = append(rv, permissionGrant)
	}
	return rv, pageToken, nil, nil
}

func accountBuilder(client *duo.Client, integrationKey string) *accountResourceType {
	return &accountResourceType{
		resourceType:   resourceTypeAccount,
		client:         client,
		integrationKey: integrationKey,
	}
}
