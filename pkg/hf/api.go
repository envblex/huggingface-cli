package hf

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type WhoamiInfo struct {
	Name string   `json:"name"`
	Orgs []string `json:"orgs"`
	Role string   `json:"role"` // "read", "write", etc.
}

type rawWhoamiResponse struct {
	Name string `json:"name"`
	Orgs []struct {
		Name string `json:"name"`
	} `json:"orgs"`
	Auth struct {
		AccessToken struct {
			Role string `json:"role"`
		} `json:"accessToken"`
	} `json:"auth"`
}

// Whoami calls /api/whoami-v2 and returns user and organization details.
func (c *Client) Whoami(token string) (*WhoamiInfo, error) {
	resp, err := c.DoRequest(context.Background(), "GET", "/api/whoami-v2", nil, token)
	if err != nil {
		return nil, fmt.Errorf("failed to call whoami: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("whoami failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var raw rawWhoamiResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to decode whoami response: %w", err)
	}

	info := &WhoamiInfo{
		Name: raw.Name,
		Role: raw.Auth.AccessToken.Role,
	}
	for _, o := range raw.Orgs {
		info.Orgs = append(info.Orgs, o.Name)
	}
	return info, nil
}

type RepoSibling struct {
	RFilename string `json:"rfilename"`
}

type RepoInfo struct {
	ID       string        `json:"id"`
	SHA      string        `json:"sha"`
	Siblings []RepoSibling `json:"siblings"`
	Private  bool          `json:"private"`
}

// RepoInfo queries repository information for a model, dataset, or space.
func (c *Client) RepoInfo(repoID, repoType, revision, token string) (*RepoInfo, error) {
	if repoType == "" {
		repoType = RepoTypeModel
	}

	path := fmt.Sprintf("/api/%ss/%s", repoType, repoID)
	if revision != "" {
		path = fmt.Sprintf("/api/%ss/%s/revision/%s", repoType, repoID, url.PathEscape(revision))
	}

	resp, err := c.DoRequest(context.Background(), "GET", path, nil, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("repo_info failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var info RepoInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode repo info: %w", err)
	}
	return &info, nil
}

type CreateRepoOptions struct {
	RepoID       string
	RepoType     string
	Private      bool
	ExistOK      bool
	SpaceSDK     string
	Organization string
	Token        string
}

// CreateRepo creates a repository on the Hub.
func (c *Client) CreateRepo(opts CreateRepoOptions) (string, error) {
	repoType := opts.RepoType
	if repoType == "" {
		repoType = RepoTypeModel
	}

	org := opts.Organization
	name := opts.RepoID
	if strings.Contains(opts.RepoID, "/") {
		parts := strings.SplitN(opts.RepoID, "/", 2)
		org = parts[0]
		name = parts[1]
	}

	payload := map[string]any{
		"name":    name,
		"private": opts.Private,
	}
	if org != "" {
		payload["organization"] = org
	}
	if repoType != RepoTypeModel {
		payload["type"] = repoType
	}
	if repoType == RepoTypeSpace && opts.SpaceSDK != "" {
		payload["sdk"] = opts.SpaceSDK
	}

	resp, err := c.DoRequest(context.Background(), "POST", "/api/repos/create", payload, opts.Token)
	if err != nil {
		return "", fmt.Errorf("failed to create repo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict && opts.ExistOK {
		// Repo exists and ExistOK
		if org != "" {
			return fmt.Sprintf("%s/%s/%s", c.Endpoint, org, name), nil
		}
		return fmt.Sprintf("%s/%s", c.Endpoint, name), nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("create repo failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var res struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil || res.URL == "" {
		if org != "" {
			return fmt.Sprintf("%s/%s/%s", c.Endpoint, org, name), nil
		}
		return fmt.Sprintf("%s/%s", c.Endpoint, name), nil
	}
	return res.URL, nil
}

// CreateBranch creates a branch on the Hub starting from specified revision or HEAD.
func (c *Client) CreateBranch(repoID, repoType, branch, revision, token string, existOK bool) error {
	if repoType == "" {
		repoType = RepoTypeModel
	}

	path := fmt.Sprintf("/api/%ss/%s/branch/%s", repoType, repoID, url.PathEscape(branch))
	payload := map[string]any{}
	if revision != "" {
		payload["startingPoint"] = revision
	}

	resp, err := c.DoRequest(context.Background(), "POST", path, payload, token)
	if err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusConflict && existOK {
		return nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create branch failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}
