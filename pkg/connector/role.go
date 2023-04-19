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
	resource "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type roleResourceType struct {
	resourceType *v2.ResourceType
	client       *duo.Client
}

func (o *roleResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

const (
	owner              = "owner"
	administrator      = "administrator"
	applicationManager = "application manager"
	userManager        = "user manager"
	helpDesk           = "help desk"
	billing            = "billing"
	phishingManager    = "phishing manager"
	readOnly           = "readonly"
)

var roles = []string{
	owner,
	administrator,
	applicationManager,
	userManager,
	helpDesk,
	billing,
	phishingManager,
	readOnly,
}

// Create a new connector resource for a Duo role.
func roleResource(ctx context.Context, role string, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	roleDisplayName := titleCaser.String(role)
	profile := map[string]interface{}{
		"role_name": roleDisplayName,
		"role_id":   role,
	}

	roleTraitOptions := []resource.RoleTraitOption{
		resource.WithRoleProfile(profile),
	}

	ret, err := resource.NewRoleResource(
		roleDisplayName,
		resourceTypeRole,
		role,
		roleTraitOptions,
		resource.WithParentResourceID(parentResourceID),
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

	var rv []*v2.Resource
	for _, role := range roles {
		rr, err := roleResource(ctx, role, parentId)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, rr)
	}

	return rv, "", nil, nil
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
		if resource.DisplayName == admin.Role {
			adminCopy := admin
			ar, err := adminResource(ctx, &adminCopy, resource.Id)
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
