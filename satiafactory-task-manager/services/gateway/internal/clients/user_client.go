package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

type UserClient struct {
	baseURL string
	client  *http.Client
}

func NewUserClient(baseURL string) *UserClient {
	return &UserClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  struct {
		ID       int64  `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"user"`
}

func (c *UserClient) Register(req RegisterRequest) (*AuthResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	log.Printf("Register request to %s: %s", c.baseURL+"/api/register", string(body))

	resp, err := c.client.Post(c.baseURL+"/api/register", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("Register connection error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("Register response status: %s", resp.Status)

	if resp.StatusCode != http.StatusCreated {
		// Читаем тело ответа для диагностики
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		log.Printf("Register error body: %s", buf.String())
		return nil, fmt.Errorf("register failed: %s", resp.Status)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}
	return &authResp, nil
}

func (c *UserClient) Login(req LoginRequest) (*AuthResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	log.Printf("Login request to %s: %s", c.baseURL+"/api/login", string(body))

	resp, err := c.client.Post(c.baseURL+"/api/login", "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("Login connection error: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("Login response status: %s", resp.Status)

	if resp.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		log.Printf("Login error body: %s", buf.String())
		return nil, fmt.Errorf("login failed: %s", resp.Status)
	}

	var authResp AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return nil, err
	}
	return &authResp, nil
}

type UserBrief struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type UserSearchRow struct {
	ID         int64  `json:"id"`
	Username   string `json:"username"`
	IsFavorite bool   `json:"is_favorite"`
}

func (c *UserClient) authGet(token, path string, dest any) error {
	req, _ := http.NewRequest("GET", c.baseURL+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c *UserClient) authPost(token, path string) error {
	req, _ := http.NewRequest("POST", c.baseURL+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: %s", resp.Status)
	}
	return nil
}

func (c *UserClient) authDelete(token, path string) error {
	req, _ := http.NewRequest("DELETE", c.baseURL+path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: %s", resp.Status)
	}
	return nil
}

func (c *UserClient) SearchUsers(token, query string) ([]UserSearchRow, error) {
	var users []UserSearchRow
	err := c.authGet(token, "/api/users/search?q="+url.QueryEscape(query), &users)
	return users, err
}

func (c *UserClient) ListFavorites(token string) ([]UserBrief, error) {
	var users []UserBrief
	err := c.authGet(token, "/api/users/favorites", &users)
	return users, err
}

func (c *UserClient) AddFavorite(token string, userID int64) error {
	return c.authPost(token, fmt.Sprintf("/api/users/favorites/%d", userID))
}

func (c *UserClient) RemoveFavorite(token string, userID int64) error {
	return c.authDelete(token, fmt.Sprintf("/api/users/favorites/%d", userID))
}

func (c *UserClient) ListUsers() ([]UserBrief, error) {
	resp, err := c.client.Get(c.baseURL + "/api/users")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list users failed: %s", resp.Status)
	}
	var users []UserBrief
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}
	return users, nil
}
