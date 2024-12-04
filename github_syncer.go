package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/google/go-github/v45/github"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type GiteaRepoSyncer struct {
	giteaUser   string
	giteaClient *gitea.Client

	githubUser   string
	githubToken  string
	githubClient *github.Client

	userCache           map[string]int64
	githubRepoCh        chan *github.Repository
	githubListStartPage int
	githubListPageSize  int
}

type GiteaUser struct {
	LoginName   string
	Username    string
	FullName    string
	Email       string
	Description string
	Website     string
	Location    string
}

type GiteaOrg struct {
	Name        string
	FullName    string
	Description string
	Website     string
	Location    string
}

func NewGiteaRepoSyncer(
	giteaServerURL, giteaUser, giteaToken string,
	githubUser, githubToken string,
	startPage, pageSize int,
) (*GiteaRepoSyncer, error) {
	giteaClient, err := gitea.NewClient(giteaServerURL, gitea.SetToken(giteaToken))
	if err != nil {
		logrus.Errorf("create gitea client error: %v", err)
		return nil, err
	}
	githubClient := github.NewClient(nil)

	return &GiteaRepoSyncer{
		giteaUser:           giteaUser,
		giteaClient:         giteaClient,
		githubUser:          githubUser,
		githubToken:         githubToken,
		githubClient:        githubClient,
		userCache:           make(map[string]int64),
		githubRepoCh:        make(chan *github.Repository, 5),
		githubListStartPage: startPage,
		githubListPageSize:  pageSize,
	}, nil
}

func ConvertGithubUserToGiteaUser(user *github.User) *GiteaUser {
	giteaUser := &GiteaUser{
		LoginName:   user.GetLogin(),
		Username:    user.GetLogin(),
		FullName:    user.GetName(),
		Email:       user.GetEmail(),
		Description: user.GetBio(),
		Website:     user.GetBlog(),
		Location:    user.GetLocation(),
	}
	if user.GetEmail() == "" {
		email := fmt.Sprintf("%s@github.com", user.GetLogin())
		logrus.Warnf("user %s email is empty, set to %s", user.GetLogin(), email)
		giteaUser.Email = email
	}
	return giteaUser
}

func (s *GiteaRepoSyncer) CreateGiteaUser(user *GiteaUser) error {
	logrus.Debugf("creating gitea user: %+v", user)
	mustChangePassword := true
	visibility := gitea.VisibleTypePublic
	createUserOpt := gitea.CreateUserOption{
		LoginName:          user.LoginName,
		Username:           user.Username,
		FullName:           user.FullName,
		Email:              user.Email,
		Password:           uuid.New().String(),
		MustChangePassword: &mustChangePassword,
		SendNotify:         false,
		Visibility:         &visibility,
	}
	newUser, _, err := s.giteaClient.AdminCreateUser(createUserOpt)
	if err != nil {
		logrus.Errorf("create gitea user %+v error, %v", user, err)
		return err
	}
	s.userCache[user.LoginName] = newUser.ID
	logrus.Infof("create gitea user %+v success", user)
	return s.UpdateGiteaUser(user)
}

func (s *GiteaRepoSyncer) UpdateGiteaUser(user *GiteaUser) error {
	logrus.Debugf("updating gitea user: %+v", user)
	editUserOpt := gitea.EditUserOption{
		LoginName: user.LoginName,
	}
	if user.Description != "" {
		editUserOpt.Description = &user.Description
	}
	if user.Website != "" {
		editUserOpt.Website = &user.Website
	}
	if user.Location != "" {
		editUserOpt.Location = &user.Location
	}
	_, err := s.giteaClient.AdminEditUser(user.LoginName, editUserOpt)
	if err != nil {
		logrus.Errorf("edit gitea user %+v error, %v", user, err)
		return err
	}
	return nil
}

func ConvertGithubOrgToGiteaOrg(org *github.Organization) *GiteaOrg {
	giteaOrg := &GiteaOrg{
		Name:        org.GetLogin(),
		FullName:    org.GetName(),
		Description: org.GetDescription(),
		Website:     org.GetBlog(),
		Location:    org.GetLocation(),
	}
	if org.GetBlog() != "" && !strings.HasPrefix(org.GetBlog(), "http") {
		website := fmt.Sprintf("http://%s", org.GetBlog())
		logrus.Warnf("user %s website is invalid, set to %s", org.GetLogin(), website)
		giteaOrg.Website = website
	}
	return giteaOrg
}

func (s *GiteaRepoSyncer) CreateGiteaOrg(org *GiteaOrg) error {
	logrus.Debugf("creating gitea org: %+v", org)
	createOrgOpt := gitea.CreateOrgOption{
		Name:                      org.Name,
		FullName:                  org.FullName,
		Description:               org.Description,
		Website:                   org.Website,
		Location:                  org.Location,
		Visibility:                gitea.VisibleTypePublic,
		RepoAdminChangeTeamAccess: true,
	}
	newOrg, _, err := s.giteaClient.AdminCreateOrg(s.giteaUser, createOrgOpt)
	if err != nil {
		logrus.Errorf("create gitea org %+v error, %v", org, err)
		return err
	}
	s.userCache[org.Name] = newOrg.ID
	logrus.Infof("create gitea org %+v success", org)
	return nil
}

func (s *GiteaRepoSyncer) UpdateGiteaOrg(org *GiteaOrg) error {
	logrus.Debugf("updating gitea org: %+v", org)
	editOrgOpt := gitea.EditOrgOption{
		FullName:    org.FullName,
		Description: org.Description,
		Website:     org.Website,
		Location:    org.Location,
		Visibility:  gitea.VisibleTypePublic,
	}
	_, err := s.giteaClient.EditOrg(org.Name, editOrgOpt)
	if err != nil {
		logrus.Errorf("edit gitea org %+v error, %v", org, err)
		return err
	}
	logrus.Infof("edit gitea org %+v success", org)
	return nil
}

func (s *GiteaRepoSyncer) isGiteaUserOrOrgExists(name string) bool {
	_, exists := s.userCache[name]
	return exists
}

func (s *GiteaRepoSyncer) EnsureGithubUser(user *github.User) error {
	switch user.GetType() {
	case "User":
		if s.isGiteaUserOrOrgExists(user.GetLogin()) {
			logrus.Infof("user %s is already exists in gitea", user.GetLogin())
			return nil
		}
		giteaUser, resp, err := s.giteaClient.GetUserInfo(user.GetLogin())
		// logrus.Debugf("err: %v, resp code: %d", err, resp.StatusCode)
		if err == nil {
			logrus.Infof("user %s is already exists in gitea", user.GetLogin())
			s.userCache[user.GetLogin()] = giteaUser.ID
			return nil
		}
		if err != nil && (resp == nil || resp.StatusCode != http.StatusNotFound) {
			logrus.Errorf("get user %s error, %s", user.GetLogin(), err)
			return err
		}
		newGiteaUser := ConvertGithubUserToGiteaUser(user)
		err = s.CreateGiteaUser(newGiteaUser)
		if err != nil {
			logrus.Errorf("ensure user %s error, %s", user.GetName(), err)
			return err
		}
	case "Organization":
		if s.isGiteaUserOrOrgExists(user.GetLogin()) {
			logrus.Infof("org %s is already exists in gitea", user.GetLogin())
			return nil
		}
		giteaOrg, resp, err := s.giteaClient.GetOrg(user.GetLogin())
		// logrus.Debugf("err: %v, resp code: %d", err, resp.StatusCode)
		if err == nil {
			logrus.Infof("org %s is already exists in gitea", user.GetLogin())
			s.userCache[user.GetLogin()] = giteaOrg.ID
			return nil
		}
		if err != nil && (resp == nil || resp.StatusCode != http.StatusNotFound) {
			logrus.Errorf("get org %s error, %s", user.GetLogin(), err)
			return err
		}
		ctx := context.Background()
		org, _, err := s.githubClient.Organizations.Get(ctx, user.GetLogin())
		if err != nil {
			logrus.Errorf("get org %s from github error, %v", user.GetLogin(), err)
			return err
		}
		newGiteaOrg := ConvertGithubOrgToGiteaOrg(org)
		err = s.CreateGiteaOrg(newGiteaOrg)
		if err != nil {
			logrus.Errorf("ensure org %s error, %s", user.GetName(), err)
			return err
		}
	default:
		logrus.Warnf("invalid user type %s, user: %s", user.GetType(), user.GetName())
	}
	return nil
}

func (s *GiteaRepoSyncer) syncGithubRepo(repo *github.Repository) error {
	owner := repo.GetOwner()
	if owner == nil {
		logrus.Warnf("repo %s owner is nil", repo.GetName())
		return nil
	}
	logrus.Debugf("owner: %+v", owner)
	err := s.EnsureGithubUser(owner)
	if err != nil {
		logrus.Errorf("sync github owner %s error: %v", owner.GetName(), err)
		return err
	}
	err = s.createGiteaMirrorRepo(owner.GetLogin(), repo.GetName(), repo.GetCloneURL())
	if err != nil {
		logrus.Errorf("create gitea mirror repo %s error, %s", repo.GetName(), err)
		return err
	}
	return nil
}

func isGiteaRepoMigrateFailed(repo *gitea.Repository) bool {
	return repo.Empty && repo.Mirror
}

func (s *GiteaRepoSyncer) createGiteaMirrorRepo(ownerName, repoName, cloneAddr string) error {
	repo, resp, err := s.giteaClient.GetRepo(ownerName, repoName)
	if err == nil && !isGiteaRepoMigrateFailed(repo) {
		logrus.Debugf("get gitea repo: %+v", repo)
		logrus.Infof("repo %s is already exists", repoName)
		return nil
	}
	if err != nil && (resp == nil || resp.StatusCode != http.StatusNotFound) {
		logrus.Errorf("get repo %s error, %s", repoName, err)
		return err
	}
	if err == nil && isGiteaRepoMigrateFailed(repo) {
		logrus.Warnf("repo %s/%s is migrate failed, try deleting", ownerName, repoName)
		_, err := s.giteaClient.DeleteRepo(ownerName, repoName)
		if err != nil {
			return err
		}
		logrus.Infof("delete gitea repo %s/%s successfully", ownerName, repoName)
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
		MirrorInterval: "168h", // 7d
	}
	logrus.Infof("migrate repo opt: %+v", opt)
	_, _, err = s.giteaClient.MigrateRepo(opt)
	if err != nil {
		logrus.Errorf("migrate repo %s error, %v", repoName, err)
		return err
	}
	logrus.Infof("migrate repo %s success", repoName)
	return nil
}

func (s *GiteaRepoSyncer) listGithubStarredRepos(stopCh <-chan struct{}) error {
	ctx := context.Background()
	const pageSize int = 10
	const pageRequestInterval = 10 * time.Second
	opts := &github.ActivityListStarredOptions{
		ListOptions: github.ListOptions{
			Page:    s.githubListStartPage,
			PerPage: pageSize,
		},
	}
	// starredRepositories := make([]*github.StarredRepository, 0)
	for {
		select {
		case <-stopCh:
			logrus.Infof("stop list")
			return nil
		default:
			logrus.Debugf("list github starred repos, page %d", opts.Page)
			repos, resp, err := s.githubClient.Activity.ListStarred(ctx, s.githubUser, opts)
			if err != nil {
				logrus.Errorf("list starred repos for %s error, %v", s.githubUser, err)
				return err
			}
			// starredRepositories = append(starredRepositories, repos...)
			for _, repo := range repos {
				select {
				case <-stopCh:
					logrus.Infof("stop enqueue")
					return nil
				case s.githubRepoCh <- repo.GetRepository():
					logrus.Infof("enqueue github repo %s/%s", repo.GetRepository().GetOwner().GetLogin(), repo.GetRepository().GetName())
				}
			}
			if resp.NextPage == 0 {
				return nil
			}
			opts.Page = resp.NextPage
			time.Sleep(pageRequestInterval)
		}
	}
}

func (s *GiteaRepoSyncer) process(stopCh <-chan struct{}) {
	for {
		select {
		case repo := <-s.githubRepoCh:
			err := s.syncGithubRepo(repo)
			if err != nil {
				logrus.Errorf("sync github repo %s error: %v", repo.GetName(), err)
				continue
			}
		case <-stopCh:
			logrus.Infof("stop process")
			return
		}
	}
}

func (s *GiteaRepoSyncer) Run(stopCh <-chan struct{}) error {
	go s.process(stopCh)
	go s.listGithubStarredRepos(stopCh)

	<-stopCh
	return nil
}
