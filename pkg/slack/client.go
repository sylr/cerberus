package slack

import (
	"github.com/sylr/cerberus/config"

	goslack "github.com/slack-go/slack"
)

// NewClient returns a slack.Client
func NewClient(conf *config.Slack) *goslack.Client {
	return goslack.New(
		conf.Token,
		goslack.OptionDebug(conf.Verbose),
	)
}
