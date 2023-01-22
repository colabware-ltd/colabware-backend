package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type GitHubRepo struct {
	Permissions GitHubRepoPermissions `json:"permissions,omitempty" bson:"permissions,omitempty"`
}

type GitHubRepoPermissions struct {
	Pull bool `json:"pull,omitempty" bson:"pull,omitempty"`
	Push bool `json:"push,omitempty" bson:"push,omitempty"`
	Admin bool `json:"admin,omitempty" bson:"admin,omitempty"`
}

type GitHubFork struct {
	FullName string `json:"full_name,omitempty" bson:"full_name,omitempty"`
}

type GitHubBranch struct {
	Name string `json:"name,omitempty" bson:"name,omitempty"`
}

func RepoBranches(client *http.Client, repoOwner string, repoName string) ([]GitHubBranch, error) {
	var branches []GitHubBranch
	res, err := client.Get("https://api.github.com/repos/" + repoOwner + "/" + repoName + "/branches")
	if err != nil {
		log.Printf("%v", err)
		return nil, fmt.Errorf("%v", err)
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&branches)
	if err != nil {
		log.Printf("%v", err)
		return nil, fmt.Errorf("%v", err)
	}

	return branches, nil
}

func RepoForks(client *http.Client,repoOwner string, repoName string) ([]GitHubFork, error) {
	var forks []GitHubFork

	res, err := client.Get("https://api.github.com/repos/" + repoOwner + "/" + repoName + "/forks")
	if err != nil {
		log.Printf("%v", err)
		return nil, fmt.Errorf("%v", err)
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&forks)
	if err != nil {
		log.Printf("%v", err)
		return nil, fmt.Errorf("%v", err)
	}
	
	return forks, nil
}

func RepoMaintainer(client *http.Client, repoOwner string, repoName string) (bool, error) {
	var repo GitHubRepo

	log.Printf(repoOwner)
	log.Printf(repoName)

	res, err := client.Get("https://api.github.com/repos/" + repoOwner + "/" + repoName)
	if err != nil {
		log.Printf("%v", err)
		return false, fmt.Errorf("%v", err)
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&repo)
	if err != nil {
		log.Printf("%v", err)
		return false, fmt.Errorf("%v", err)
	}

	return repo.Permissions.Admin, nil
}