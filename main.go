package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

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
	StartPage     int
	PageSize      int
}

func SyncGithubStarredRepos(o *SyncGithubStarredReposOption) error {
	syncer, err := NewGiteaRepoSyncer(
		o.GiteaOption.ServerURL, o.GiteaOption.User, o.GiteaOption.AuthToken,
		o.GithubOption.User, o.GithubOption.AuthToken,
		o.StartPage, o.PageSize,
	)
	if err != nil {
		return err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	stopCh := make(chan struct{})
	go func() {
		<-sigCh
		logrus.Infof("received shutdown signal")
		close(stopCh)
	}()

	err = syncer.Run(stopCh)
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
	cmd.Flags().IntVar(&o.StartPage, "start-page", 0, "github starred list start page")
	cmd.Flags().IntVar(&o.PageSize, "page-size", 10, "github starred list page size")

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
	logrus.SetFormatter(formatter)
	logrus.SetReportCaller(true)
	logrus.SetOutput(os.Stdout)
	err := cmd.Execute()
	if err != nil {
		fmt.Println(err)
	}
}

var formatter = &logrus.TextFormatter{
	TimestampFormat:        "2006-01-02 15:04:05", // the "time" field configuratiom
	FullTimestamp:          true,
	DisableLevelTruncation: true, // log level field configuration
	CallerPrettyfier: func(f *runtime.Frame) (string, string) {
		return "", fmt.Sprintf("%s:%d", formatFilePath(f.File), f.Line)
	},
}

func formatFilePath(path string) string {
	arr := strings.Split(path, "/")
	return arr[len(arr)-1]
}
