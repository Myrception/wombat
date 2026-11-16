package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hashicorp/go-version"
)

// releaseRepo is the GitHub repository whose releases are offered as updates.
// It must match the repository the release workflow publishes to.
const releaseRepo = "Myrception/wombat"

const latestReleaseURL = "https://api.github.com/repos/" + releaseRepo + "/releases/latest"

var errNoUpdate = errors.New("no update available")

type releaseResponse struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

var ghClient *http.Client

func init() {
	ghClient = &http.Client{
		Timeout: 5 * time.Second,
	}
}

func checkForUpdate() (*releaseInfo, error) {
	if isDev {
		return nil, errNoUpdate
	}

	req, err := http.NewRequest("GET", latestReleaseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	resp, err := ghClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %v", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	r := &releaseResponse{}
	if err := json.Unmarshal(raw, r); err != nil {
		return nil, err
	}

	if !isNewerVersion(r.TagName, semver) {
		return nil, errNoUpdate
	}

	return &releaseInfo{
		OldVersion: semver,
		NewVersion: r.TagName,
		URL:        r.HTMLURL,
	}, nil
}

// isNewerVersion reports whether latest is a newer version than current. If
// either cannot be parsed there is no (known) update.
func isNewerVersion(latest, current string) bool {
	vl, el := version.NewVersion(latest)
	vc, ec := version.NewVersion(current)
	return el == nil && ec == nil && vl.GreaterThan(vc)
}
