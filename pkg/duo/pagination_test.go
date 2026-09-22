package duo

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/conductorone/baton-sdk/pkg/uhttp"
)

// clientServing points a client at a stub Duo that always replies with body. Each test
// gets its own server, so the uhttp response cache cannot carry a body between them.
func clientServing(t *testing.T, body string) *Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return NewClient("ikey", "skey", "", server.URL, server.Client())
}

// Duo keeps metadata on the last page - carrying prev_offset and total_objects - and omits
// only next_offset. That is the end of the sync, not a failure.
func TestGetUsersLastPage(t *testing.T) {
	c := clientServing(t, `{"stat":"OK","response":[{"user_id":"u1","username":"ada"}],"metadata":{"prev_offset":2200,"total_objects":2342}}`)

	users, offset, err := c.GetUsers(context.Background(), "2300")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("Expected 1 user, got %d", len(users))
	}
	if offset != "" {
		t.Errorf("Expected an empty offset on the last page, got %q", offset)
	}
}

func TestGetUsersNextOffset(t *testing.T) {
	c := clientServing(t, `{"stat":"OK","response":[{"user_id":"u1"}],"metadata":{"next_offset":100,"total_objects":2342}}`)

	_, offset, err := c.GetUsers(context.Background(), "0")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if offset != "100" {
		t.Errorf("Expected offset %q, got %q", "100", offset)
	}
}

// The case the opt-in exists for. This client always sends limit, which is the condition
// under which Duo returns metadata, so a 200 without it means the block was dropped.
func TestGetUsersMissingMetadata(t *testing.T) {
	c := clientServing(t, `{"stat":"OK","response":[{"user_id":"u1"}]}`)

	_, _, err := c.GetUsers(context.Background(), "0")
	if err == nil {
		t.Fatal("Expected an error when Duo omits metadata")
	}
	if !errors.Is(err, uhttp.ErrMissingPaginationData) {
		t.Fatalf("Expected ErrMissingPaginationData, got %v", err)
	}
}

func TestGetGroupsMissingMetadata(t *testing.T) {
	c := clientServing(t, `{"stat":"OK","response":[{"group_id":"g1"}]}`)

	_, _, err := c.GetGroups(context.Background(), "0")
	if err == nil {
		t.Fatal("Expected an error when Duo omits metadata")
	}
	if !errors.Is(err, uhttp.ErrMissingPaginationData) {
		t.Fatalf("Expected ErrMissingPaginationData, got %v", err)
	}
}

func TestGetAdminsMissingMetadata(t *testing.T) {
	c := clientServing(t, `{"stat":"OK","response":[{"admin_id":"a1"}]}`)

	_, _, err := c.GetAdmins(context.Background(), "0")
	if err == nil {
		t.Fatal("Expected an error when Duo omits metadata")
	}
	if !errors.Is(err, uhttp.ErrMissingPaginationData) {
		t.Fatalf("Expected ErrMissingPaginationData, got %v", err)
	}
}

// Unpaginated reads share the same do() and carry no metadata, so they must not start
// demanding it.
func TestGetAccountDoesNotRequirePagination(t *testing.T) {
	c := clientServing(t, `{"stat":"OK","response":{"name":"Acme"}}`)

	account, err := c.GetAccount(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if account.Name != "Acme" {
		t.Errorf("Expected account name %q, got %q", "Acme", account.Name)
	}
}
