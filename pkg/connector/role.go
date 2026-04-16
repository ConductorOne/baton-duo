package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-duo/pkg/duo"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	ent "github.com/conductorone/baton-sdk/pkg/types/entitlement"
	grant "github.com/conductorone/baton-sdk/pkg/types/grant"
	resource "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type roleResourceType struct {
	resourceType *v2.ResourceType
	client       *duo.Client
}

func (o *roleResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// legacyRoleID maps the API's role_id to the hardcoded ID previously used by
// this resource type. These aliases let ConductorOne resolve existing
// references after the migration from the old hardcoded list to API-sourced roles.
var legacyRoleID = map[string]string{
	"owner":               "owner",
	"service_manager":     "administrator",
	"application_manager": "application manager",
	"user_manager":        "user manager",
	"help_desk":           "help desk",
	"billing":             "billing",
	"read_only":           "read-only",
}

// Create a new connector resource for a Duo role.
func roleResource(_ context.Context, role duo.Role, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := map[string]interface{}{
		"role_name": role.Name,
		"role_id":   role.RoleID,
		"is_custom": role.IsCustom,
	}

	roleTraitOptions := []resource.RoleTraitOption{
		resource.WithRoleProfile(profile),
	}

	resourceOpts := []resource.ResourceOption{
		resource.WithParentResourceID(parentResourceID),
	}
	legacyCompatibleResourceID := role.RoleID
	if legacy, ok := legacyRoleID[role.RoleID]; ok {
		// TODO: Use withAliases to migrate to proper IDs once the feature is fully supported.
		legacyCompatibleResourceID = legacy
	}

	ret, err := resource.NewRoleResource(
		role.Name,
		resourceTypeRole,
		legacyCompatibleResourceID,
		roleTraitOptions,
		resourceOpts...,
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *roleResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	if parentId == nil {
		return nil, "", nil, nil
	}

	roles, rl, err := o.client.GetRoles(ctx)
	if err != nil {
		return nil, "", nil, err
	}

	var rv []*v2.Resource
	for _, role := range roles {
		rr, err := roleResource(ctx, role, parentId)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, rr)
	}

	return rv, "", annotations.New(rl), nil
}

func (o *roleResourceType) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement

	assignmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeAdmin),
		ent.WithDescription(fmt.Sprintf("%s Duo role", resource.DisplayName)),
		ent.WithDisplayName(fmt.Sprintf("%s Role %s", resource.DisplayName, memberEntitlement)),
	}

	assignmentEn := ent.NewAssignmentEntitlement(resource, memberEntitlement, assignmentOptions...)
	rv = append(rv, assignmentEn)
	return rv, "", nil, nil
}

func (o *roleResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var pageToken string
	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeAdmin.Id})
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
		if resource.Id.Resource == admin.RoleID {
			ar, err := adminResource(ctx, &admin, resource.Id)
			if err != nil {
				return nil, "", nil, err
			}
			permissionGrant := grant.NewGrant(resource, memberEntitlement, ar.Id)
			rv = append(rv, permissionGrant)
		}
	}

	return rv, pageToken, nil, nil
}

func roleBuilder(client *duo.Client) *roleResourceType {
	return &roleResourceType{
		resourceType: resourceTypeRole,
		client:       client,
	}
}
