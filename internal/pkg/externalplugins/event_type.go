package externalplugins

import (
	"errors"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/github"
)

// EventType is an alias of string which describe the GitHub webhook event.
type EventType = string

// Event type constants.
const (
	IssuesEvent       EventType = "issues"
	IssueCommentEvent EventType = "issue_comment"

	PullRequestEvent              EventType = "pull_request"
	PullRequestReviewEvent        EventType = "pull_request_review"
	PullRequestReviewCommentEvent EventType = "pull_request_review_comment"

	PushEvent EventType = "push"

	StatusEvent EventType = "status"
)

var (
	ErrUndefinedEventHandler = errors.New("undefined event handler")
	ErrUnknownEventType      = errors.New("unknown event type")
)

type IssuesEventHandler func(l *logrus.Entry, i *github.IssueEvent) error
type IssueCommentEventHandler func(l *logrus.Entry, ic *github.IssueCommentEvent) error
type PullRequestEventHandler func(l *logrus.Entry, pr *github.PullRequestEvent) error
type PullRequestReviewEventHandler func(l *logrus.Entry, re *github.ReviewEvent) error
type PullRequestReviewCommentEventHandler func(l *logrus.Entry, rce *github.ReviewCommentEvent) error
type PushEventHandler func(l *logrus.Entry, pe *github.PushEvent) error
type StatusEventHandler func(l *logrus.Entry, se *github.StatusEvent) error

// Handlers contains different type of event handler.
type EventHandlers struct {
	Issues                   IssuesEventHandler
	IssueComment             IssueCommentEventHandler
	PullRequest              PullRequestEventHandler
	PullRequestReview        PullRequestReviewEventHandler
	PullRequestReviewComment PullRequestReviewCommentEventHandler
	Push                     PushEventHandler
	Status                   StatusEventHandler
}

func (eh *EventHandlers) SetIssuesHandler(f IssuesEventHandler) {
	eh.Issues = f
}
