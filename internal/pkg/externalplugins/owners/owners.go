package owners

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	tiexternalplugins "github.com/tidb-community-bots/ti-community-prow/internal/pkg/externalplugins"
	"github.com/tidb-community-bots/ti-community-prow/internal/pkg/ownersclient"
	"k8s.io/test-infra/prow/github"
)

const (
	// SigEndpointFmt specifies a format for sigs URL.
	SigEndpointFmt = "/sigs/%s"
)

const (
	// sigPrefix is a default sig label prefix.
	sigPrefix = "sig/"
	// listOwnersSuccessMessage returns on success.
	listOwnersSuccessMessage = "List all owners success."
	lgtmTwo                  = 2
)

type githubClient interface {
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	ListCollaborators(org, repo string) ([]github.User, error)
}

type Server struct {
	// Client for get sig info.
	Client *http.Client

	TokenGenerator func() []byte
	Gc             githubClient
	ConfigAgent    *tiexternalplugins.ConfigAgent
	Log            *logrus.Entry
}

func (s *Server) listOwnersForNonSig(org string, repo string) (*ownersclient.OwnersResponse, error) {
	collaborators, err := s.Gc.ListCollaborators(org, repo)
	if err != nil {
		s.Log.WithField("org", org).WithField("repo", repo).WithError(err).Error("Failed get collaborators.")
		return nil, err
	}

	var collaboratorsLogin []string
	for _, collaborator := range collaborators {
		collaboratorsLogin = append(collaboratorsLogin, collaborator.Login)
	}

	return &ownersclient.OwnersResponse{
		Data: ownersclient.Owners{
			Approvers: collaboratorsLogin,
			Reviewers: collaboratorsLogin,
			NeedsLgtm: lgtmTwo,
		},
		Message: listOwnersSuccessMessage,
	}, nil
}

func (s *Server) listOwnersForSig(org string, repo string, sigName string,
	config *tiexternalplugins.Configuration) (*ownersclient.OwnersResponse, error) {
	owners := config.OwnersFor(org, repo)

	url := owners.SigEndpoint + fmt.Sprintf(SigEndpointFmt, sigName)
	// Get sig info.
	res, err := s.Client.Get(url)
	if err != nil {
		s.Log.WithField("url", url).WithError(err).Error("Failed get sig info.")
		return nil, err
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != 200 {
		s.Log.WithField("url", url).WithError(err).Error("Failed get sig info.")
		return nil, errors.New("could not get a sig")
	}

	// Unmarshal sig members from body.
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var sigRes SigResponse
	if err := json.Unmarshal(body, &sigRes); err != nil {
		s.Log.WithField("body", body).WithError(err).Error("Failed unmarshal body.")
		return nil, err
	}

	var approvers []string
	var reviewers []string

	sig := sigRes.Data

	for _, leader := range sig.Membership.TechLeaders {
		approvers = append(approvers, leader.GithubName)
		reviewers = append(reviewers, leader.GithubName)
	}

	for _, coLeader := range sig.Membership.CoLeaders {
		approvers = append(approvers, coLeader.GithubName)
		reviewers = append(reviewers, coLeader.GithubName)
	}

	for _, committer := range sig.Membership.Committers {
		approvers = append(approvers, committer.GithubName)
		reviewers = append(reviewers, committer.GithubName)
	}

	for _, reviewer := range sig.Membership.Reviewers {
		reviewers = append(reviewers, reviewer.GithubName)
	}

	return &ownersclient.OwnersResponse{
		Data: ownersclient.Owners{
			Approvers: approvers,
			Reviewers: reviewers,
			NeedsLgtm: sig.NeedsLgtm,
		},
		Message: listOwnersSuccessMessage,
	}, nil
}

// ListOwners returns owners of tidb community PR.
func (s *Server) ListOwners(org string, repo string, number int,
	config *tiexternalplugins.Configuration) (*ownersclient.OwnersResponse, error) {
	// Get pull request.
	pull, err := s.Gc.GetPullRequest(org, repo, number)
	if err != nil {
		s.Log.WithField("pullNumber", number).WithError(err).Error("Failed get pull request.")
		return nil, err
	}

	// Find sig label.
	sigName := GetSigNameByLabel(pull.Labels)

	// When we cannot find a sig label for PR, we will use a collaborators.
	if sigName == "" {
		return s.listOwnersForNonSig(org, repo)
	}

	return s.listOwnersForSig(org, repo, sigName, config)
}

// GetSigNameByLabel returns the name of sig when the label prefix matches.
func GetSigNameByLabel(labels []github.Label) string {
	var sigName string
	for _, label := range labels {
		if strings.HasPrefix(label.Name, sigPrefix) {
			sigName = strings.TrimPrefix(label.Name, sigPrefix)
			return sigName
		}
	}

	return ""
}