package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sylr/cerberus/config"
	"github.com/sylr/cerberus/pkg/slack/actions"

	log "github.com/sirupsen/logrus"
	goslack "github.com/slack-go/slack"
	goslackevents "github.com/slack-go/slack/slackevents"
)

var (
	metricEventsReceivedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "cerberus",
			Subsystem: "slack_events",
			Name:      "received_total",
			Help:      "Number of slack events received",
		},
		[]string{"type"},
	)

	metricEventsUnhandledTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "cerberus",
			Subsystem: "slack_events",
			Name:      "unhandled_total",
			Help:      "Number of unhandled slack events received",
		},
		[]string{"type"},
	)

	metricActionPerformedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "cerberus",
			Subsystem: "actions",
			Name:      "performed_total",
			Help:      "Number of actions performed",
		},
		[]string{"type"},
	)
)

func init() {
	prometheus.MustRegister(metricEventsReceivedTotal)
	prometheus.MustRegister(metricEventsUnhandledTotal)
	prometheus.MustRegister(metricActionPerformedTotal)
}

// Handler ...
type Handler struct {
	Config      *config.Cerberus
	Logger      *log.Logger
	SlackClient *goslack.Client

	AppMentionEventActions     []actions.Actionner
	MessageEventActions        []actions.Actionner
	SubteamUpdatedEventActions []actions.Actionner
}

// NewHandler ...
func NewHandler(conf *config.Cerberus, logger *log.Logger, slackClient *goslack.Client) *Handler {
	h := Handler{
		Config:      conf,
		Logger:      logger,
		SlackClient: slackClient,
	}

	h.AppMentionEventActions = append(h.AppMentionEventActions, actions.NewCerberusMention(conf, logger, slackClient))
	h.MessageEventActions = append(h.MessageEventActions, actions.NewAtChannelMention(conf, logger, slackClient))
	h.SubteamUpdatedEventActions = append(h.SubteamUpdatedEventActions, actions.NewSubteamUpdated(conf, logger, slackClient))

	return &h
}

// ServeHTTP ...
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r.Body)
	js := json.RawMessage(buf.Bytes())
	token := goslackevents.OptionNoVerifyToken()
	eventsAPIEvent, err := goslackevents.ParseEvent(js, token)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		h.Logger.Errorf("%v", err)
		h.Logger.Infof("json=%v", string(js))
		return
	}

	h.Logger.Debugf("eventsAPIEvent.Type=%v", eventsAPIEvent.Type)

	// Callback event
	if eventsAPIEvent.Type == goslackevents.CallbackEvent {
		innerEvent := eventsAPIEvent.InnerEvent
		h.Logger.Debugf("eventsAPIEvent.InnerEvent=%v", innerEvent)

		metricEventsReceivedTotal.WithLabelValues(innerEvent.Type).Inc()

		// Events type
		switch ev := innerEvent.Data.(type) {
		// AppMentionEvent
		case *goslackevents.AppMentionEvent:
			h.Logger.Debugf("eventsAPIEvent.InnerEvent.Data=%v", ev)

			for _, action := range h.AppMentionEventActions {
				actionned, err := action.Action(ev)

				if err != nil {
					h.Logger.Errorf("AppMentionEvent: %s", err)
				}

				if actionned {
					metricActionPerformedTotal.WithLabelValues(fmt.Sprintf("%T", action)).Inc()
				}
			}

		// MessageEvent
		case *goslackevents.MessageEvent:
			h.Logger.Debugf("eventsAPIEvent.InnerEvent.Data=%v", ev)

			for _, action := range h.MessageEventActions {
				actionned, err := action.Action(ev)

				if err != nil {
					h.Logger.Errorf("MessageEvent: %s", err)
				}

				if actionned {
					metricActionPerformedTotal.WithLabelValues(fmt.Sprintf("%T", action)).Inc()
				}
			}

		// SubteamUpdatedEvent
		case *goslack.SubteamUpdatedEvent:
			h.Logger.Debugf("eventsAPIEvent.InnerEvent.Data=%v", ev)

			for _, action := range h.SubteamUpdatedEventActions {
				actionned, err := action.Action(ev)

				if err != nil {
					h.Logger.Errorf("SubteamUpdatedEvent: %s", err)
				}

				if actionned {
					metricActionPerformedTotal.WithLabelValues(fmt.Sprintf("%T", action)).Inc()
				}
			}

		default:
			metricEventsUnhandledTotal.WithLabelValues(innerEvent.Type).Inc()
			h.Logger.Warnf("event inner type not handled: %#v", ev)
		}

		return
	}

	// URL Verification
	if eventsAPIEvent.Type == goslackevents.URLVerification {
		var r *goslackevents.ChallengeResponse
		err := json.Unmarshal(js, &r)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			h.Logger.Errorf("%v", err)
			return
		}

		w.Header().Set("Content-Type", "text")
		w.Write([]byte(r.Challenge))

		return
	}
}
