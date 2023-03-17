package connector

import (
	"context"
	"fmt"
	"strings"

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

type adminResourceType struct {
	resourceType *v2.ResourceType
	client       *duo.Client
}

func (o *adminResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// Create a new connector resource for a Duo admin.
func adminResource(ctx context.Context, admin *duo.Admin, _ *v2.ResourceId) (*v2.Resource, error) {
	names := strings.SplitN(admin.Name, " ", 2)
	var firstName, lastName string
	switch len(names) {
	case 1:
		firstName = names[0]
	case 2:
		firstName = names[0]
		lastName = names[1]
	}
	profile := map[string]interface{}{
		"first_name": firstName,
		"last_name":  lastName,
		"login":      admin.Email,
		"user_id":    admin.AdminID,
		"role":       admin.Role,
	}

	adminTraitOptions := []rs.UserTraitOption{
		rs.WithUserProfile(profile),
		rs.WithEmail(admin.Email, true),
	}

	ret, err := rs.NewUserResource(
		admin.Name,
		resourceTypeAdmin,
		admin.AdminID,
		adminTraitOptions,
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *adminResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	var pageToken string
	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeAdmin.Id})
	if err != nil {
		return nil, "", nil, err
	}
	admins, offset, err := o.client.GetAdmins(ctx, bag.PageToken())
	if err != nil {
		return nil, "", nil, fmt.Errorf("duo-connector: failed to list admins: %w", err)
	}

	if offset != "" {
		pageToken, err = bag.NextToken(offset)
		if err != nil {
			return nil, "", nil, err
		}
	}

	var rv []*v2.Resource
	for _, admin := range admins {
		adminCopy := admin
		ar, err := adminResource(ctx, &adminCopy, parentId)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, ar)
	}

	return rv, pageToken, nil, nil
}

func (o *adminResourceType) Entitlements(_ context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement
	for _, role := range roles {
		permissionOptions := []ent.EntitlementOption{
			ent.WithGrantableTo(resourceTypeAdmin),
			ent.WithDescription(fmt.Sprintln("Admin role")),
			ent.WithDisplayName(fmt.Sprintf("role %s", role)),
		}

		permissionEn := ent.NewPermissionEntitlement(resource, role, permissionOptions...)
		rv = append(rv, permissionEn)
	}
	return rv, "", nil, nil
}

func (o *adminResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
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
		roleName, ok := roles[admin.Role]
		if !ok {
			ctxzap.Extract(ctx).Warn("Unknown Duo Role Name, skipping",
				zap.String("role_name", admin.Role),
				zap.String("user", admin.Name),
			)
			continue
		}
		adminCopy := admin
		ur, err := adminResource(ctx, &adminCopy, resource.Id)
		if err != nil {
			return nil, "", nil, err
		}

		permissionGrant := grant.NewGrant(resource, roleName, ur.Id)
		rv = append(rv, permissionGrant)
	}
	return rv, pageToken, nil, nil
}

func adminBuilder(client *duo.Client) *adminResourceType {
	return &adminResourceType{
		resourceType: resourceTypeAdmin,
		client:       client,
	}
}
