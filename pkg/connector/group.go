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
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

const memberEntitlement = "member"

type groupResourceType struct {
	resourceType *v2.ResourceType
	client       *duo.Client
}

func (o *groupResourceType) ResourceType(_ context.Context) *v2.ResourceType {
	return o.resourceType
}

func groupBuilder(client *duo.Client) *groupResourceType {
	return &groupResourceType{
		resourceType: resourceTypeGroup,
		client:       client,
	}
}

// Create a new connector resource for a Duo group.
func groupResource(ctx context.Context, group duo.Group, parentResourceID *v2.ResourceId) (*v2.Resource, error) {
	profile := make(map[string]interface{})
	profile["group_id"] = group.GroupID
	profile["group_name"] = group.Name

	groupTrait := []rs.GroupTraitOption{
		rs.WithGroupProfile(profile),
	}

	ret, err := rs.NewGroupResource(
		group.Name,
		resourceTypeGroup,
		group.GroupID,
		groupTrait,
		rs.WithParentResourceID(parentResourceID),
	)

	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (o *groupResourceType) List(ctx context.Context, parentId *v2.ResourceId, token *pagination.Token) ([]*v2.Resource, string, annotations.Annotations, error) {
	if parentId == nil {
		return nil, "", nil, nil
	}

	var pageToken string
	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeGroup.Id})
	if err != nil {
		return nil, "", nil, err
	}

	groups, offset, err := o.client.GetGroups(ctx, bag.PageToken())
	if err != nil {
		return nil, "", nil, err
	}

	if offset != "" {
		pageToken, err = bag.NextToken(offset)
		if err != nil {
			return nil, "", nil, err
		}
	}

	var rv []*v2.Resource
	for _, group := range groups {
		gr, err := groupResource(ctx, group, parentId)
		if err != nil {
			return nil, "", nil, err
		}
		rv = append(rv, gr)
	}

	return rv, pageToken, nil, nil
}

func (o *groupResourceType) Entitlements(ctx context.Context, resource *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	var rv []*v2.Entitlement

	assigmentOptions := []ent.EntitlementOption{
		ent.WithGrantableTo(resourceTypeUser),
		ent.WithDescription(fmt.Sprintf("Member of %s Group in Duo", resource.DisplayName)),
		ent.WithDisplayName(fmt.Sprintf("%s Group %s", resource.DisplayName, memberEntitlement)),
	}

	en := ent.NewAssignmentEntitlement(resource, memberEntitlement, assigmentOptions...)
	rv = append(rv, en)

	return rv, "", nil, nil
}

func (o *groupResourceType) Grants(ctx context.Context, resource *v2.Resource, token *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	var pageToken string
	bag, err := parsePageToken(token.Token, &v2.ResourceId{ResourceType: resourceTypeGroup.Id})
	if err != nil {
		return nil, "", nil, err
	}

	users, offset, err := o.client.GetGroupUsers(ctx, resource.Id.Resource, bag.PageToken())
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
	for _, user := range users {
		userCopy := user
		user, err := o.client.GetUser(ctx, userCopy.UserID)
		if err != nil {
			return nil, "", nil, err
		}
		ur, err := userResource(ctx, &user, resource.Id)
		if err != nil {
			return nil, "", nil, err
		}

		membershipGrant := grant.NewGrant(resource, memberEntitlement, ur.Id)
		rv = append(rv, membershipGrant)
	}

	return rv, pageToken, nil, nil
}
