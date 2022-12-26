package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func main() {
	if err := run(); err != nil {
		log.Println(err)

		os.Exit(1)
	}
}

func getUpstreamURL(ctx context.Context) (string, error) {
	resp, err := http.Get("https://www.kernel.org/releases.json")
	if err != nil {
		return "", err
	}
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		return "", fmt.Errorf("unexpected HTTP status code: got %d, want %d", got, want)
	}
	var releases struct {
		LatestStable struct {
			Version string `json:"version"`
		} `json:"latest_stable"`
		Releases []struct {
			Version string `json:"version"`
			Source  string `json:"source"`
		} `json:"releases"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", err
	}
	for _, release := range releases.Releases {
		if release.Version != releases.LatestStable.Version {
			continue
		}
		return release.Version, nil
	}
	return "", fmt.Errorf("malformed releases.json: latest stable release %q not found in releases list", releases.LatestStable.Version)
}

func run() error {
	updateVersion, err := getUpstreamURL(context.Background())
	if err != nil {
		return err
	}

	tagName := fmt.Sprintf("v%v", updateVersion)

	log.Println("latest version:", tagName)

	latestSha, err := githubCommitSha(tagName)
	if err != nil {
		return err
	}
	log.Println("latest commit:", latestSha)

	currentSha, err := submoduleSha("linux-sources")
	if err != nil {
		return err
	}
	log.Println("submodule commit:", currentSha)

	if latestSha == currentSha {
		log.Println("already up to date")
		return nil
	}
	fmt.Println(latestSha)

	return nil
}

func githubCommitSha(tagName string) (string, error) {
	cmd := exec.Command("git", "ls-remote", "-t", "origin", tagName)
	cmd.Dir = "linux-sources"
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// split first column
	return strings.TrimSpace(strings.Split(string(out), "\t")[0]), nil
}

func submoduleSha(submodule string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = "linux-sources"
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
