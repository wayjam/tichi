package externalplugins

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/github"
)

func logEventHandleNotFound(l *logrus.Entry, eventType EventType) error {
	l.Debugf("received an event of type %q but implementation not found", eventType)
	return ErrUndefinedEventHandler
}

// DemuxEvent dispatches the provided payload to the handler.
func DemuxEvent(log *logrus.Entry, handlers *EventHandlers, eventType EventType, eventGUID string, payload []byte) error {
	l := log.WithFields(
		logrus.Fields{
			"event-type":     eventType,
			github.EventGUID: eventGUID,
		},
	)

	switch eventType {
	case IssuesEvent:
		var i github.IssueEvent
		if err := json.Unmarshal(payload, &i); err != nil {
			return err
		}
		i.GUID = eventGUID
		if handlers.Issues != nil {
			go func() {
				if err := handlers.Issues(l, &i); err != nil {
					l.WithField("event-type", eventType).WithError(err).Info("Error handling event.")
				}
			}()
		} else {
			return logEventHandleNotFound(log, eventType)
		}
	case IssueCommentEvent:
		var ic github.IssueCommentEvent
		if err := json.Unmarshal(payload, &ic); err != nil {
			return err
		}
		ic.GUID = eventGUID
		if handlers.IssueComment != nil {
			go func() {
				if err := handlers.IssueComment(l, &ic); err != nil {
					l.WithField("event-type", eventType).WithError(err).Info("Error handling event.")
				}
			}()
		} else {
			return logEventHandleNotFound(log, eventType)
		}
	case PullRequestEvent:
		var pr github.PullRequestEvent
		if err := json.Unmarshal(payload, &pr); err != nil {
			return err
		}
		pr.GUID = eventGUID
		if handlers.PullRequest != nil {
			go func() {
				if err := handlers.PullRequest(l, &pr); err != nil {
					l.WithField("event-type", eventType).WithError(err).Info("Error handling event.")
				}
			}()
		} else {
			return logEventHandleNotFound(log, eventType)
		}
	case PullRequestReviewEvent:
		var re github.ReviewEvent
		if err := json.Unmarshal(payload, &re); err != nil {
			return err
		}
		re.GUID = eventGUID
		if handlers.PullRequestReview != nil {
			go func() {
				if err := handlers.PullRequestReview(l, &re); err != nil {
					l.WithField("event-type", eventType).WithError(err).Info("Error handling event.")
				}
			}()
		} else {
			return logEventHandleNotFound(log, eventType)
		}
	case PullRequestReviewCommentEvent:
		var rce github.ReviewCommentEvent
		if err := json.Unmarshal(payload, &rce); err != nil {
			return err
		}
		rce.GUID = eventGUID
		if handlers.PullRequestReviewComment != nil {
			go func() {
				if err := handlers.PullRequestReviewComment(l, &rce); err != nil {
					l.WithField("event-type", eventType).WithError(err).Info("Error handling event.")
				}
			}()
		} else {
			return logEventHandleNotFound(log, eventType)
		}
	case PushEvent:
		var pe github.PushEvent
		if err := json.Unmarshal(payload, &pe); err != nil {
			return err
		}
		pe.GUID = eventGUID
		if handlers.Push != nil {
			go func() {
				if err := handlers.Push(l, &pe); err != nil {
					l.WithField("event-type", eventType).WithError(err).Info("Error handling event.")
				}
			}()
		} else {
			return logEventHandleNotFound(log, eventType)
		}
	case StatusEvent:
		var se github.StatusEvent
		if err := json.Unmarshal(payload, &se); err != nil {
			return err
		}
		se.GUID = eventGUID
		if handlers.Status != nil {
			go func() {
				if err := handlers.Status(l, &se); err != nil {
					l.WithField("event-type", eventType).WithError(err).Info("Error handling event.")
				}
			}()
		} else {
			return logEventHandleNotFound(log, eventType)
		}
	default:
		log.Debugf("received an event of type %q but didn't ask for it", eventType)
		return ErrUnknownEventType
	}

	return nil
}
