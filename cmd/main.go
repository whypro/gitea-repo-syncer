package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/google/go-github/v45/github"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type GiteaRepoSyncer struct {
	githubUser   string
	githubToken  string
	githubClient *github.Client
	giteaUser    string
	giteaClient  *gitea.Client
}

func NewGiteaRepoSyncer(githubUser, githubToken string, giteaUser, giteaToken, giteaURL string) (*GiteaRepoSyncer, error) {
	githubClient := github.NewClient(nil)
	giteaClient, err := gitea.NewClient(giteaURL, gitea.SetToken(giteaToken))
	if err != nil {
		return nil, err
	}

	return &GiteaRepoSyncer{
		githubUser:   githubUser,
		githubToken:  githubToken,
		githubClient: githubClient,
		giteaUser:    giteaUser,
		giteaClient:  giteaClient,
	}, nil
}

func (s *GiteaRepoSyncer) GetStarredRepos() ([]*github.StarredRepository, error) {
	ctx := context.Background()
	page := 1
	perPage := 10
	starredRepositories := make([]*github.StarredRepository, 0)
	for {
		opts := &github.ActivityListStarredOptions{
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		}
		repos, _, err := s.githubClient.Activity.ListStarred(ctx, s.githubUser, opts)
		if err != nil {
			logrus.Errorf("list starred repos of %s error, %v", s.githubUser, err)
			return nil, err
		}
		starredRepositories = append(starredRepositories, repos...)
		if len(repos) < perPage {
			break
		}
		page += 1
		time.Sleep(5 * time.Second)
	}
	return starredRepositories, nil
}

type SyncFn func(*github.Repository) error

func (s *GiteaRepoSyncer) SyncStarredRepos(syncFn SyncFn, ignoreOnError bool) error {
	ctx := context.Background()
	page := 1
	perPage := 10
	for {
		opts := &github.ActivityListStarredOptions{
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: perPage,
			},
		}
		repos, _, err := s.githubClient.Activity.ListStarred(ctx, s.githubUser, opts)
		if err != nil {
			logrus.Errorf("list starred repos of %s error, %v", s.githubUser, err)
			return err
		}
		for _, repo := range repos {
			err := syncFn(repo.GetRepository())
			if err != nil {
				if ignoreOnError {
					logrus.Warnf("sync repo %s error, %v", repo.GetRepository().GetName(), err)
					continue
				} else {
					logrus.Errorf("sync repo %s error, %v", repo.GetRepository().GetName(), err)
					return err
				}
			}
			time.Sleep(5 * time.Second)
		}
		if len(repos) < perPage {
			break
		}
		page += 1
		time.Sleep(5 * time.Second)
	}
	return nil
}

//func (s *GiteaRepoSyncer) SyncRepo(cloneAddr string, syncFn SyncFn) error {
//	ctx := context.Background()
//	s.githubClient.
//
//}

func (s *GiteaRepoSyncer) EnsureOwner(owner *github.User) error {
	if owner.GetType() != "User" {
		return fmt.Errorf("invalid user type, %s", owner.GetType())
	}
	_, resp, err := s.giteaClient.GetUserInfo(owner.GetLogin())
	// logrus.Debugf("err: %v, resp code: %d", err, resp.StatusCode)
	if err == nil {
		logrus.Infof("user %s is already exists in gitea", owner.GetLogin())
		return nil
	}
	if err != nil && (resp == nil || resp.StatusCode != http.StatusNotFound) {
		logrus.Errorf("get user %s info error, %s", owner.GetLogin(), err)
		return err
	}
	ctx := context.Background()
	user, _, err := s.githubClient.Users.Get(ctx, owner.GetLogin())
	if err != nil {
		logrus.Errorf("get user %s from github error, %v", owner.GetLogin(), err)
		return err
	}
	logrus.Debugf("user: %+v", user)
	mustChangePassword := true
	visibility := gitea.VisibleTypePublic
	createUserOpt := gitea.CreateUserOption{
		LoginName:          user.GetLogin(),
		Username:           user.GetLogin(),
		FullName:           user.GetName(),
		Email:              user.GetEmail(),
		Password:           uuid.New().String(),
		MustChangePassword: &mustChangePassword,
		SendNotify:         false,
		Visibility:         &visibility,
	}
	if createUserOpt.Email == "" {
		createUserOpt.Email = fmt.Sprintf("%s@github.com", user.GetLogin())
	}
	_, _, err = s.giteaClient.AdminCreateUser(createUserOpt)
	if err != nil {
		logrus.Errorf("create gitea user %s error, %v", user.GetLogin(), err)
		return err
	}
	logrus.Infof("create gitea user %s success", user.GetLogin())
	editUserOpt := gitea.EditUserOption{
		LoginName: user.GetLogin(),
	}
	blog := user.GetBlog()
	logrus.Debugf("blog: %v", blog)
	if blog != "" {
		editUserOpt.Website = &blog
	}
	location := user.GetLocation()
	if location != "" {
		editUserOpt.Location = &location
	}
	_, err = s.giteaClient.AdminEditUser(user.GetLogin(), editUserOpt)
	if err != nil {
		logrus.Errorf("edit gitea user %s error, %v", user.GetLogin(), err)
		return err
	}
	return nil
}

func (s *GiteaRepoSyncer) EnsureOrg(owner *github.User) error {
	if owner.GetType() != "Organization" {
		return fmt.Errorf("invalid user type, %s", owner.GetType())
	}
	_, resp, err := s.giteaClient.GetOrg(owner.GetLogin())
	// logrus.Debugf("err: %v, resp code: %d", err, resp.StatusCode)
	if err == nil {
		logrus.Infof("org %s is already exists in gitea", owner.GetLogin())
		return nil
	}
	if err != nil && (resp == nil || resp.StatusCode != http.StatusNotFound) {
		logrus.Errorf("get org %s info error, %s", owner.GetLogin(), err)
		return err
	}
	ctx := context.Background()
	org, _, err := s.githubClient.Organizations.Get(ctx, owner.GetLogin())
	if err != nil {
		logrus.Errorf("get org %s from github error, %v", owner.GetLogin(), err)
		return err
	}
	logrus.Debugf("org: %+v", org)
	createOrgOpt := gitea.CreateOrgOption{
		Name:                      owner.GetLogin(),
		FullName:                  org.GetName(),
		Description:               org.GetDescription(),
		Website:                   org.GetBlog(),
		Location:                  org.GetLocation(),
		Visibility:                gitea.VisibleTypePublic,
		RepoAdminChangeTeamAccess: true,
	}
	_, _, err = s.giteaClient.AdminCreateOrg(s.giteaUser, createOrgOpt)
	if err != nil {
		logrus.Errorf("create gitea org %s error, %v", owner.GetLogin(), err)
		return err
	}
	logrus.Infof("create gitea org %s success", owner.GetLogin())
	return nil
}

func (s *GiteaRepoSyncer) MigrateRepo(repoName, ownerName, cloneAddr string) error {
	repo, resp, err := s.giteaClient.GetRepo(ownerName, repoName)
	if err == nil {
		logrus.Debugf("repo: %+v", repo)
		logrus.Infof("repo %s is already exists", repoName)
		return nil
	}
	if err != nil && (resp == nil || resp.StatusCode != http.StatusNotFound) {
		logrus.Errorf("get repo %s error, %s", repoName, err)
		return err
	}
	opt := gitea.MigrateRepoOption{
		RepoName:       repoName,
		RepoOwner:      ownerName,
		CloneAddr:      cloneAddr,
		Service:        gitea.GitServiceGithub,
		AuthToken:      s.githubToken,
		Mirror:         true,
		Private:        false,
		Wiki:           true,
		Milestones:     true,
		Labels:         true,
		Issues:         true,
		PullRequests:   true,
		Releases:       true,
		MirrorInterval: "24h",
	}
	logrus.Infof("opt: %+v", opt)
	_, _, err = s.giteaClient.MigrateRepo(opt)
	if err != nil {
		logrus.Errorf("migrate repo %s error, %v", repoName, err)
		return err
	}
	logrus.Infof("migrate repo %s success", repoName)
	return nil
}

func (s *GiteaRepoSyncer) SyncRepo(repo *github.Repository) error {
	owner := repo.GetOwner()
	if owner == nil {
		logrus.Warnf("repo %s owner is nil", repo.GetName())
		return nil
	}
	logrus.Debugf("owner: %+v", owner)
	switch owner.GetType() {
	case "User":
		err := s.EnsureOwner(owner)
		if err != nil {
			logrus.Errorf("ensure user %s error, %s", owner.GetName(), err)
			return err
		}
	case "Organization":
		err := s.EnsureOrg(owner)
		if err != nil {
			logrus.Errorf("ensure org %s error, %s", owner.GetName(), err)
			return err
		}
	default:
		logrus.Warnf("invalid owner %s type %s", repo.GetName(), owner.GetType())
	}
	err := s.MigrateRepo(repo.GetName(), owner.GetLogin(), repo.GetCloneURL())
	if err != nil {
		logrus.Errorf("migrate repo %s error, %s", repo.GetName(), err)
		return err
	}
	return nil
}

// gitea-repo-syncer
func main() {
	//logrus.SetLevel(logrus.InfoLevel)
	logrus.SetLevel(logrus.DebugLevel)
	githubUser := os.Getenv("GITHUB_USER")
	githubToken := os.Getenv("GITHUB_AUTH_TOKEN")
	giteaUser := os.Getenv("GITEA_USER")
	giteaToken := os.Getenv("GITEA_AUTH_TOKEN")
	giteaURL := os.Getenv("GITEA_URL")
	syncer, err := NewGiteaRepoSyncer(githubUser, githubToken, giteaUser, giteaToken, giteaURL)
	if err != nil {
		panic(err)
	}
	err = syncer.SyncStarredRepos(syncer.SyncRepo, true)
	if err != nil {
		panic(err)
	}
}
