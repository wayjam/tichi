package main

import (
	"crypto/tls"
	"net/http"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/commentpruner"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/github"

	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins/merge"
	"github.com/ti-community-infra/tichi/internal/pkg/ownersclient"
)

type svr struct {
	tiexternalplugins.BaseServer

	ol ownersclient.OwnersLoader
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

	// Skip https verify.
	//nolint:gosec
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	ol := &ownersclient.OwnersClient{Client: client}

	s := svr{
		BaseServer: tiexternalplugins.NewBaseServer(merge.PluginName, &o, merge.HelpProvider),
		gc:         githubClient,
	}

	s.SetHandlers(&tiexternalplugins.EventHandlers{
		IssueComment: func(l *logrus.Entry, ic *github.IssueCommentEvent) error {
			// This should be used once per webhook event.
			cp := commentpruner.NewEventClient(
				s.gc, s.Log.WithField("client", "commentpruner"),
				ic.Repo.Owner.Login, ic.Repo.Name, ic.Issue.Number,
			)
			return merge.HandleIssueCommentEvent(s.gc, ic, s.GetExternalPluginConfig(), ol, cp, l)
		},
		PullRequestReviewComment: func(l *logrus.Entry, rc *github.ReviewCommentEvent) error {
			// This should be used once per webhook event.
			cp := commentpruner.NewEventClient(
				s.gc, s.Log.WithField("client", "commentpruner"),
				rc.Repo.Owner.Login, rc.Repo.Name, rc.PullRequest.Number,
			)
			return merge.HandlePullReviewCommentEvent(s.gc, rc, s.GetExternalPluginConfig(), ol, cp, l)
		},
		PullRequest: func(l *logrus.Entry, pr *github.PullRequestEvent) error {
			return merge.HandlePullRequestEvent(s.gc, pr, s.GetExternalPluginConfig(), l)
		},
	})

	s.Run()
}
