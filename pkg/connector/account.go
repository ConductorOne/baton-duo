package connector

import (
	"context"
	"fmt"

	"github.com/conductorone/baton-duo/pkg/duo"
	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/annotations"
	"github.com/conductorone/baton-sdk/pkg/pagination"
	rs "github.com/conductorone/baton-sdk/pkg/types/resource"
)

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
			&v2.ChildResourceType{ResourceTypeId: resourceTypeRole.Id},
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

func (o *accountResourceType) Entitlements(_ context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Entitlement, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func (o *accountResourceType) Grants(ctx context.Context, _ *v2.Resource, _ *pagination.Token) ([]*v2.Grant, string, annotations.Annotations, error) {
	return nil, "", nil, nil
}

func accountBuilder(client *duo.Client, integrationKey string) *accountResourceType {
	return &accountResourceType{
		resourceType:   resourceTypeAccount,
		client:         client,
		integrationKey: integrationKey,
	}
}
