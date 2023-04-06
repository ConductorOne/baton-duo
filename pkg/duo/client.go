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
)

const (
	paginationLimit   = "100"
	requestFailedStat = "FAIL"
)

type Client struct {
	httpClient     *http.Client
	integrationKey string
	secretKey      string
	baseUrl        string
	host           string
}

func NewClient(integrationKey string, secretKey string, apiHostname string, httpClient *http.Client) *Client {
	baseUrl := fmt.Sprintf("https://%s", apiHostname)
	return &Client{
		integrationKey: integrationKey,
		secretKey:      secretKey,
		baseUrl:        baseUrl,
		host:           apiHostname,
		httpClient:     httpClient,
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

type IntegrationResponse struct {
	ErrorResponse
	Stat     string `json:"stat"`
	Response struct {
		Name           string `json:"name"`
		IntegrationKey string `json:"integration_key"`
	} `json:"response"`
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
		return nil, "", fmt.Errorf("error fetching users: %s", res.Message)
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
		return nil, "", fmt.Errorf("error fetching groups: %s", res.Message)
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
		return nil, "", fmt.Errorf("error fetching group users: %s", res.Message)
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
		return nil, "", fmt.Errorf("error fetching admins: %s", res.Message)
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
		return User{}, fmt.Errorf("error fetching a user: %s", res.Message)
	}

	return res.Response, nil
}

// GetIntegration returns an integration by integration key.
func (c *Client) GetIntegration(ctx context.Context) (IntegrationResponse, error) {
	uri := fmt.Sprintf("/admin/v1/integrations/%s", c.integrationKey)
	adminsUrl := fmt.Sprint(c.baseUrl, uri)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, adminsUrl, nil)
	if err != nil {
		return IntegrationResponse{}, err
	}

	var res IntegrationResponse
	if err := c.doRequest(uri, req, &res, nil); err != nil {
		return IntegrationResponse{}, err
	}

	if res.Stat == requestFailedStat {
		return IntegrationResponse{}, fmt.Errorf("error fetching integration: %s", res.Message)
	}

	return res, nil
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
		return Account{}, fmt.Errorf("error fetching account: %s", res.Message)
	}

	return res.Response, nil
}

func (c *Client) doRequest(uri string, req *http.Request, resType interface{}, params url.Values) error {
	now := time.Now().UTC().Format(time.RFC1123Z)
	signature, err := sign(c.integrationKey, c.secretKey, "GET", c.host, uri, now, params)
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", signature)
	req.Header.Add("Date", now)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&resType); err != nil {
		return err
	}

	return nil
}
