package main

import (
	"crypto/tls"
	"net/http"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/github"

	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins/lgtm"
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
		BaseServer: tiexternalplugins.NewBaseServer(lgtm.PluginName, &o, lgtm.HelpProvider),
		gc:         githubClient,
	}

	s.SetHandlers(&tiexternalplugins.EventHandlers{
		IssueComment: func(l *logrus.Entry, ic *github.IssueCommentEvent) error {
			return lgtm.HandleIssueCommentEvent(s.gc, ic, s.GetExternalPluginConfig(), ol, l)
		},
		PullRequestReviewComment: func(l *logrus.Entry, rc *github.ReviewCommentEvent) error {
			return lgtm.HandlePullReviewCommentEvent(s.gc, rc, s.GetExternalPluginConfig(), ol, l)
		},
		PullRequestReview: func(l *logrus.Entry, re *github.ReviewEvent) error {
			return lgtm.HandlePullReviewEvent(s.gc, re, s.GetExternalPluginConfig(), ol, l)
		},
		PullRequest: func(l *logrus.Entry, pr *github.PullRequestEvent) error {
			return lgtm.HandlePullRequestEvent(s.gc, pr, s.GetExternalPluginConfig(), l)
		},
	})

	s.Run()
}
