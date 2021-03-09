package main

import (
	"crypto/tls"
	"net/http"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/github"

	tiexternalplugins "github.com/ti-community-infra/tichi/internal/pkg/externalplugins"
	"github.com/ti-community-infra/tichi/internal/pkg/externalplugins/blunderbuss"
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
		BaseServer: tiexternalplugins.NewBaseServer(blunderbuss.PluginName, &o, blunderbuss.HelpProvider),
		gc:         githubClient,
		ol:         ol,
	}

	s.SetHandlers(&tiexternalplugins.EventHandlers{
		IssueComment: func(l *logrus.Entry, ic *github.IssueCommentEvent) error {
			return blunderbuss.HandleIssueCommentEvent(s.gc, ic, s.GetExternalPluginConfig(), ol, l)
		},
		PullRequest: func(l *logrus.Entry, pr *github.PullRequestEvent) error {
			return blunderbuss.HandlePullRequestEvent(s.gc, pr, s.GetExternalPluginConfig(), ol, l)
		},
	})

	s.Run()
}
