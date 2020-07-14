package actions

import (
	"github.com/sylr/cerberus/config"
	"github.com/sylr/cerberus/pkg/slack"

	log "github.com/sirupsen/logrus"
	goslack "github.com/slack-go/slack"
)

// NewSubteamUpdated returns a new Actionner
func NewSubteamUpdated(conf *config.Cerberus, logger *log.Logger, client *goslack.Client) Actionner {
	actionner := &SubteamUpdated{
		config: conf,
		logger: logger,
		client: client,
	}

	return actionner
}

type SubteamUpdated struct {
	config *config.Cerberus
	logger *log.Logger
	client *goslack.Client
}

func (a *SubteamUpdated) Action(event interface{}) (bool, error) {
	ev := event.(*goslack.SubteamUpdatedEvent)

	slack.InvalidateGroupCache(ev.Subteam.Name)

	return true, nil
}
