package webapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/rezadesky/highrisebot/models"
)

const DefaultBaseURL = "https://webapi.highrise.game"

// PaginationOptions provides cursor parameters for webapi queries.
type PaginationOptions struct {
	StartsAfter string
	EndsBefore  string
	Limit       int
}

// Client interacts with the Highrise REST Web API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new Highrise Web API REST client.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// UserProfile represents the detailed profile returned by the Web API.
type UserProfile struct {
	User        models.User `json:"user"`
	Bio         string      `json:"bio,omitempty"`
	JoinedAt    string      `json:"joined_at,omitempty"`
	NumFollowers int        `json:"num_followers,omitempty"`
	NumFollowing int        `json:"num_following,omitempty"`
}

// UsersResponse represents paginated response for users search.
type UsersResponse struct {
	Users []models.User `json:"users"`
	Total int           `json:"total,omitempty"`
}

// RoomsResponse represents paginated response for rooms list.
type RoomsResponse struct {
	Rooms []models.RoomInfo `json:"rooms"`
	Total int               `json:"total,omitempty"`
}

// ItemsResponse represents list of items.
type ItemsResponse struct {
	Items []models.Item `json:"items"`
	Total int           `json:"total,omitempty"`
}

// GetUser fetches a specific user profile by user ID.
func (c *Client) GetUser(ctx context.Context, userID string) (*UserProfile, error) {
	endpoint := fmt.Sprintf("%s/users/%s", c.BaseURL, url.PathEscape(userID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webapi: request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("webapi: status %d: %s", res.StatusCode, string(body))
	}

	var profile UserProfile
	if err := json.NewDecoder(res.Body).Decode(&profile); err != nil {
		return nil, fmt.Errorf("webapi: decode failed: %w", err)
	}
	return &profile, nil
}

// SearchUsers queries users with optional username search and pagination.
func (c *Client) SearchUsers(ctx context.Context, username string, opts *PaginationOptions) (*UsersResponse, error) {
	u, err := url.Parse(fmt.Sprintf("%s/users", c.BaseURL))
	if err != nil {
		return nil, err
	}

	q := u.Query()
	if username != "" {
		q.Set("username", username)
	}
	applyPagination(q, opts)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("webapi: status %d: %s", res.StatusCode, string(body))
	}

	var usersRes UsersResponse
	if err := json.NewDecoder(res.Body).Decode(&usersRes); err != nil {
		return nil, err
	}
	return &usersRes, nil
}

// GetRooms lists public rooms with pagination.
func (c *Client) GetRooms(ctx context.Context, opts *PaginationOptions) (*RoomsResponse, error) {
	u, err := url.Parse(fmt.Sprintf("%s/rooms", c.BaseURL))
	if err != nil {
		return nil, err
	}

	q := u.Query()
	applyPagination(q, opts)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("webapi: status %d: %s", res.StatusCode, string(body))
	}

	var roomsRes RoomsResponse
	if err := json.NewDecoder(res.Body).Decode(&roomsRes); err != nil {
		return nil, err
	}
	return &roomsRes, nil
}

// GetItem retrieves item details by item ID.
func (c *Client) GetItem(ctx context.Context, itemID string) (*models.Item, error) {
	endpoint := fmt.Sprintf("%s/items/%s", c.BaseURL, url.PathEscape(itemID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("webapi: status %d: %s", res.StatusCode, string(body))
	}

	var item models.Item
	if err := json.NewDecoder(res.Body).Decode(&item); err != nil {
		return nil, err
	}
	return &item, nil
}

func applyPagination(q url.Values, opts *PaginationOptions) {
	if opts == nil {
		return
	}
	if opts.StartsAfter != "" {
		q.Set("starts_after", opts.StartsAfter)
	}
	if opts.EndsBefore != "" {
		q.Set("ends_before", opts.EndsBefore)
	}
	if opts.Limit > 0 {
		q.Set("limit", strconv.Itoa(opts.Limit))
	}
}
