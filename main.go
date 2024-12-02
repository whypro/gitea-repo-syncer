package main

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type GiteaOption struct {
	User      string
	AuthToken string
	ServerURL string
}

type GithubOption struct {
	User      string
	AuthToken string
}

type SyncGithubStarredReposOption struct {
	GiteaOption
	GithubOption
	IgnoreOnError bool
}

func SyncGithubStarredRepos(o *SyncGithubStarredReposOption) error {
	syncer, err := NewGiteaRepoSyncer(
		o.GiteaOption.ServerURL, o.GiteaOption.User, o.GiteaOption.AuthToken,
		o.GithubOption.User, o.GithubOption.AuthToken,
	)
	if err != nil {
		return err
	}
	err = syncer.SyncGithubStarredRepos()
	if err != nil {
		return err
	}
	return nil
}

func NewSyncGithubStarredRepos() *cobra.Command {
	o := &SyncGithubStarredReposOption{}
	cmd := &cobra.Command{
		Use: "sync-github-starred-repos",
		Run: func(cmd *cobra.Command, args []string) {
			err := SyncGithubStarredRepos(o)
			if err != nil {
				fmt.Printf("%v", err)
			}
		},
	}

	cmd.Flags().StringVar(&o.GiteaOption.ServerURL, "gitea-server-url", os.Getenv("GITEA_SERVER_URL"), "Gitea server url")
	cmd.Flags().StringVar(&o.GiteaOption.User, "gitea-user", os.Getenv("GITEA_USER"), "Gitea user name")
	cmd.Flags().StringVar(&o.GiteaOption.AuthToken, "gitea-auth-token", os.Getenv("GITEA_AUTH_TOKEN"), "Gitea auth token")
	cmd.Flags().StringVar(&o.GithubOption.User, "github-user", os.Getenv("GITHUB_USER"), "Github user name")
	cmd.Flags().StringVar(&o.GithubOption.AuthToken, "github-auth-token", os.Getenv("GITHUB_AUTH_TOKEN"), "Github auth token")

	cmd.Flags().BoolVar(&o.IgnoreOnError, "ignore-on-error", false, "Ingore on error occured")

	return cmd
}

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "gitea-repo-syncer",
	}
	cmd.AddCommand(NewSyncGithubStarredRepos())
	return cmd
}

func main() {
	cmd := NewRootCmd()
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetOutput(os.Stdout)
	err := cmd.Execute()
	if err != nil {
		fmt.Println(err)
	}
}
