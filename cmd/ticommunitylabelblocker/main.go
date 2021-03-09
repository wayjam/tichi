package main

import (
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/github"

	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins/labelblocker"
)

type svr struct {
	tiexternalplugins.BaseServer

	gc github.Client
}

func main() {
	o := tiexternalplugins.ServerOptions{}
	o.ParseFromFlags()
	if err := o.Validate(); err != nil {
		logrus.Fatalf("Invalid options: %v", err)
	}

	secretAgent := &secret.Agent{}
	if err := secretAgent.Start([]string{o.Github.TokenPath, o.WebhookSecretFile}); err != nil {
		logrus.WithError(err).Fatal("Error starting secrets agent.")
	}

	githubClient, err := o.Github.GitHubClient(secretAgent, o.DryRun)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting GitHub client.")
	}
	githubClient.Throttle(360, 360)

	s := svr{
		BaseServer: tiexternalplugins.NewBaseServer(labelblocker.PluginName, &o, labelblocker.HelpProvider),
		gc:         githubClient,
	}

	s.SetHandlers(&tiexternalplugins.EventHandlers{
		PullRequest: func(l *logrus.Entry, pr *github.PullRequestEvent) error {
			return labelblocker.HandlePullRequestEvent(s.gc, pr, s.GetExternalPluginConfig(), l)
		},
		Issues: func(l *logrus.Entry, ie *github.IssueEvent) error {
			return labelblocker.HandleIssueEvent(s.gc, ie, s.GetExternalPluginConfig(), l)
		},
	})

	s.Run()
}
