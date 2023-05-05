package connector

import (
	"context"
	"fmt"
	"strings"

	"github.com/conductorone/baton-duo/pkg/duo"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

type adminResourceType struct {
	resourceType *v2.ResourceType
	client       *duo.Client
}

func (o *adminResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

// Create a new connector resource for a Duo admin.
func adminResource(ctx context.Context, admin *duo.Admin, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
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
		rs.WithParentResourceID(parentResourceID),
	)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *adminResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	if parentId == nil {
		return nil, "", nil, nil
	}

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
	return nil, "", nil, nil
}

func (o *adminResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func adminBuilder(client *duo.Client) *adminResourceType {
	return &adminResourceType{
		resourceType: resourceTypeAdmin,
		client:       client,
	}
}
