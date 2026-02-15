package duo

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	v2 "github.com/conductorone/baton-sdk/pb/c1/connector/v2"
	"github.com/conductorone/baton-sdk/pkg/uhttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	paginationLimit   = "100"
	requestFailedStat = "FAIL"
)

type Client struct {
	wrapper        *uhttp.BaseHttpClient
	integrationKey string
	secretKey      string
	baseUrl        string
	host           string
}

func NewClient(integrationKey string, secretKey string, apiHostname string, baseURL string, httpClient *http.Client) *Client {
	effectiveBaseURL := baseURL
	effectiveHost := apiHostname
	if effectiveBaseURL == "" {
		effectiveBaseURL = fmt.Sprintf("https://%s", apiHostname)
	} else if u, err := url.Parse(effectiveBaseURL); err == nil && u.Host != "" {
		effectiveHost = u.Host
	}
	return &Client{
		integrationKey: integrationKey,
		secretKey:      secretKey,
		baseUrl:        effectiveBaseURL,
		host:           effectiveHost,
		wrapper:        uhttp.NewBaseHttpClient(httpClient),
	}
}

type ListResultMetadata struct {
	NextOffset   json.Number `json:"next_offset"`
	PrevOffset   json.Number `json:"prev_offset"`
	TotalObjects json.Number `json:"total_objects"`
}

type ErrorResponse struct {
	Code          int64  `json:"code,omitempty"`
	Message       string `json:"message,omitempty"`
	MessageDetail string `json:"message_detail,omitempty"`
}

type UsersResponse struct {
	ErrorResponse
	Metadata ListResultMetadata `json:"metadata"`
	Response []User             `json:"response"`
	Stat     string             `json:"stat"`
}

type GroupsResponse struct {
	ErrorResponse
	Metadata ListResultMetadata `json:"metadata"`
	Stat     string             `json:"stat"`
	Response []Group            `json:"response,omitempty"`
}

type GroupUsersResponse struct {
	ErrorResponse
	Metadata ListResultMetadata `json:"metadata"`
	Stat     string             `json:"stat"`
	Response []User             `json:"response"`
}

type AdminsResponse struct {
	ErrorResponse
	Metadata ListResultMetadata `json:"metadata"`
	Stat     string             `json:"stat"`
	Response []Admin            `json:"response"`
}

type UserResponse struct {
	ErrorResponse
	Stat     string `json:"stat"`
	Response User   `json:"response"`
}

type AccountResponse struct {
	ErrorResponse
	Stat     string  `json:"stat"`
	Response Account `json:"response"`
}

type RolesResponse struct {
	ErrorResponse
	Stat     string `json:"stat"`
	Response []Role `json:"response"`
}

func duoToGRPCErrorCode(duoCode int64) codes.Code {
	// Extract the first 3 digits from the left, Duo sends a 5 digit code for errors
	httpCode := duoCode
	for httpCode >= 1000 {
		httpCode /= 10
	}
	switch httpCode {
	case http.StatusBadRequest:
		return codes.InvalidArgument
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		return codes.Unavailable
	case http.StatusUnauthorized:
		return codes.Unauthenticated
	case http.StatusForbidden:
		return codes.PermissionDenied
	case http.StatusNotFound:
		return codes.NotFound
	case http.StatusNotImplemented:
		return codes.Unimplemented
	case http.StatusInternalServerError:
		return codes.Internal
	}

	if httpCode >= 500 && httpCode <= 599 {
		return codes.Unavailable
	}

	if httpCode < 200 || httpCode >= 300 {
		return codes.Unknown
	}

	return codes.Unknown
}

func wrapError(errResp ErrorResponse, message string) error {
	grpcErrorCode := duoToGRPCErrorCode(errResp.Code)
	status := status.New(grpcErrorCode, errResp.Message)
	return fmt.Errorf("duo-connector: %s: %w", message, status.Err())
}

// returns query params with pagination options.
func paginationQuery(offset string) url.Values {
	q := url.Values{}

	if offset == "" {
		offset = "0"
	}

	q.Set("offset", offset)
	q.Set("limit", paginationLimit)
	return q
}

// found in duo go examples library - needed for signing requests.
func canonParams(params url.Values) string {
	for key, val := range params {
		sort.Strings(val)
		params[key] = val
	}
	orderedParams := params.Encode()
	// duo needs %XX escaping
	return strings.NewReplacer("+", "%20").Replace(orderedParams)
}

// found in duo go examples library - needed for signing requests.
func canonicalize(
	method string,
	host string,
	uri string,
	params url.Values,
	date string,
) string {
	var canon [5]string
	canon[0] = date
	canon[1] = strings.ToUpper(method)
	canon[2] = strings.ToLower(host)
	canon[3] = uri
	canon[4] = canonParams(params)
	return strings.Join(canon[:], "\n")
}

// found in duo go examples library - needed for signing requests.
func sign(ikey string,
	skey string,
	method string,
	host string,
	uri string,
	date string,
	params url.Values) (string, error) {
	canon := canonicalize(method, host, uri, params, date)
	mac := hmac.New(sha512.New, []byte(skey))
	_, err := mac.Write([]byte(canon))
	if err != nil {
		return "", err
	}
	sig := hex.EncodeToString(mac.Sum(nil))
	auth := ikey + ":" + sig
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth)), nil
}

// GetUsers returns all users.
func (c *Client) GetUsers(ctx context.Context, offset string) ([]User, string, error) {
	uri := "/admin/v1/users"
	usersUrl := fmt.Sprint(c.baseUrl, uri)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usersUrl, nil)
	if err != nil {
		return nil, "", err
	}

	params := paginationQuery(offset)
	req.URL.RawQuery = params.Encode()

	var res UsersResponse
	if err := c.doRequest(uri, req, &res, params); err != nil {
		return nil, "", err
	}

	if res.Stat == requestFailedStat {
		return nil, "", wrapError(res.ErrorResponse, "error fetching users")
	}

	if (res.Metadata != ListResultMetadata{}) {
		return res.Response, res.Metadata.NextOffset.String(), nil
	}

	return res.Response, "", nil
}

// GetGroups returns all groups.
func (c *Client) GetGroups(ctx context.Context, offset string) ([]Group, string, error) {
	uri := "/admin/v1/groups"
	usersUrl := fmt.Sprint(c.baseUrl, uri)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usersUrl, nil)
	if err != nil {
		return nil, "", err
	}

	params := paginationQuery(offset)
	req.URL.RawQuery = params.Encode()

	var res GroupsResponse
	if err := c.doRequest(uri, req, &res, params); err != nil {
		return nil, "", err
	}

	if res.Stat == requestFailedStat {
		return nil, "", wrapError(res.ErrorResponse, "error fetching groups")
	}

	if (res.Metadata != ListResultMetadata{}) {
		return res.Response, res.Metadata.NextOffset.String(), nil
	}

	return res.Response, "", nil
}

// GetGroupUsers returns all users in a group.
func (c *Client) GetGroupUsers(ctx context.Context, groupId string, offset string) ([]User, string, error) {
	uri := fmt.Sprintf("/admin/v2/groups/%s/users", groupId)
	usersUrl := fmt.Sprint(c.baseUrl, uri)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, usersUrl, nil)
	if err != nil {
		return nil, "", err
	}

	params := paginationQuery(offset)
	req.URL.RawQuery = params.Encode()

	var res GroupUsersResponse
	if err := c.doRequest(uri, req, &res, params); err != nil {
		return nil, "", err
	}

	if res.Stat == requestFailedStat {
		return nil, "", wrapError(res.ErrorResponse, "error fetching group users")
	}

	if (res.Metadata != ListResultMetadata{}) {
		return res.Response, res.Metadata.NextOffset.String(), nil
	}

	return res.Response, "", nil
}

// GetAdmins returns all admins.
func (c *Client) GetAdmins(ctx context.Context, offset string) ([]Admin, string, error) {
	uri := "/admin/v1/admins"
	adminsUrl := fmt.Sprint(c.baseUrl, uri)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, adminsUrl, nil)
	if err != nil {
		return nil, "", err
	}

	params := paginationQuery(offset)
	req.URL.RawQuery = params.Encode()

	var res AdminsResponse
	if err := c.doRequest(uri, req, &res, params); err != nil {
		return nil, "", err
	}

	if res.Stat == requestFailedStat {
		return nil, "", wrapError(res.ErrorResponse, "error fetching admins")
	}

	if (res.Metadata != ListResultMetadata{}) {
		return res.Response, res.Metadata.NextOffset.String(), nil
	}

	return res.Response, "", nil
}

// GetUser returns a user by ID.
func (c *Client) GetUser(ctx context.Context, userId string) (User, error) {
	uri := fmt.Sprintf("/admin/v1/users/%s", userId)
	adminsUrl := fmt.Sprint(c.baseUrl, uri)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, adminsUrl, nil)
	if err != nil {
		return User{}, err
	}

	var res UserResponse
	if err := c.doRequest(uri, req, &res, nil); err != nil {
		return User{}, err
	}

	if res.Stat == requestFailedStat {
		return User{}, wrapError(res.ErrorResponse, "error fetching a user")
	}

	return res.Response, nil
}

// GetAccount returns account info.
func (c *Client) GetAccount(ctx context.Context) (Account, error) {
	uri := "/admin/v1/settings"
	accountUrl := fmt.Sprint(c.baseUrl, uri)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, accountUrl, nil)
	if err != nil {
		return Account{}, err
	}

	var res AccountResponse
	if err := c.doRequest(uri, req, &res, nil); err != nil {
		return Account{}, err
	}

	if res.Stat == requestFailedStat {
		return Account{}, wrapError(res.ErrorResponse, "error fetching account")
	}

	return res.Response, nil
}

// GetRoles returns all admin roles along with rate limit data from the response headers.
func (c *Client) GetRoles(ctx context.Context) ([]Role, *v2.RateLimitDescription, error) {
	uri := "/admin/v1/admin_roles"
	rolesUrl, err := url.JoinPath(c.baseUrl, uri)
	if err != nil {
		return nil, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rolesUrl, nil)
	if err != nil {
		return nil, nil, err
	}

	var res RolesResponse
	rl, err := c.do(uri, req, &res, nil)
	if err != nil {
		return nil, nil, err
	}

	if res.Stat == requestFailedStat {
		return nil, rl, wrapError(res.ErrorResponse, "error fetching roles")
	}

	return res.Response, rl, nil
}

// AddUserToGroup adds a user to a group.
func (c *Client) AddUserToGroup(ctx context.Context, groupId, userId string) error {
	uri := fmt.Sprint("/admin/v1/users/", userId, "/groups")
	addUserUrl := fmt.Sprint(c.baseUrl, uri)
	data := url.Values{}
	data.Set("group_id", groupId)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, addUserUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return err
	}

	var res struct {
		Stat string `json:"stat"`
		ErrorResponse
	}

	if err := c.doRequest(uri, req, &res, data); err != nil {
		return err
	}

	if res.Stat == requestFailedStat {
		return wrapError(res.ErrorResponse, "error adding user to group")
	}

	return nil
}

// RemoveUserFromGroup removes a user from a group.
func (c *Client) RemoveUserFromGroup(ctx context.Context, groupId, userId string) error {
	uri := fmt.Sprint("/admin/v1/users/", userId, "/groups/", groupId)
	removeUserUrl := fmt.Sprint(c.baseUrl, uri)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, removeUserUrl, nil)
	if err != nil {
		return err
	}

	var res struct {
		Stat string `json:"stat"`
		ErrorResponse
	}

	if err := c.doRequest(uri, req, &res, nil); err != nil {
		return err
	}

	if res.Stat == requestFailedStat {
		return wrapError(res.ErrorResponse, "error removing user from group")
	}

	return nil
}

// do executes the request, decodes the JSON response body into resType, and returns
// any rate limit information parsed from the response headers.
func (c *Client) do(uri string, req *http.Request, resType interface{}, params url.Values) (*v2.RateLimitDescription, error) {
	now := time.Now().UTC().Format(time.RFC1123Z)
	signature, err := sign(c.integrationKey, c.secretKey, req.Method, c.host, uri, now, params)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", signature)
	req.Header.Add("Date", now)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	var rl v2.RateLimitDescription
	// #nosec G704 -- URL is config baseUrl + fixed path pattern, not user-controlled
	resp, err := c.wrapper.Do(req,
		uhttp.WithJSONResponse(resType),
		uhttp.WithRatelimitData(&rl),
	)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return &rl, err
	}

	return &rl, nil
}

// doRequest executes the request and decodes the response, discarding rate limit data.
func (c *Client) doRequest(uri string, req *http.Request, resType interface{}, params url.Values) error {
	_, err := c.do(uri, req, resType, params)
	return err
}
