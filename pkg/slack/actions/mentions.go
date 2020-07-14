package actions

import (
	"fmt"
	"math/rand"
	"net/url"
	"strings"

	"github.com/sylr/cerberus/config"
	"github.com/sylr/cerberus/pkg/slack"

	log "github.com/sirupsen/logrus"
	goslack "github.com/slack-go/slack"
	goslackevents "github.com/slack-go/slack/slackevents"
)

// -----------------------------------------------------------------------------

// NewCerberusMention returns a new Actionner
func NewCerberusMention(conf *config.Cerberus, logger *log.Logger, client *goslack.Client) Actionner {
	actionner := &CerberusMention{
		config: conf,
		logger: logger,
		client: client,
	}

	return actionner
}

type CerberusMention struct {
	config *config.Cerberus
	logger *log.Logger
	client *goslack.Client
}

func (a *CerberusMention) Action(event interface{}) (bool, error) {
	ev := event.(*goslackevents.AppMentionEvent)

	var msgOptions []goslack.MsgOption

	// Thread
	msgOptions = append(msgOptions, goslack.MsgOptionPostMessageParameters(goslack.PostMessageParameters{
		ThreadTimestamp: ev.TimeStamp,
	}))

	if a.config.CerberusMention == nil || len(a.config.CerberusMention.Messages) == 0 {
		message := "wOOf wOOf"
		msgOptions = append(msgOptions, goslack.MsgOptionText(message, false))
	} else {
		index := rand.Intn(len(a.config.CerberusMention.Messages))
		text := a.config.CerberusMention.Messages[index].Text
		imageURL := a.config.CerberusMention.Messages[index].ImageURL

		if len(imageURL) > 0 {
			_, err := url.ParseRequestURI(imageURL)

			if err != nil {
				a.logger.Errorf("%s is not a valid url", imageURL)
			} else {
				var blockText *goslack.TextBlockObject

				if len(text) > 0 {
					blockText = goslack.NewTextBlockObject(goslack.PlainTextType, text, true, false)
				} else {
					blockText = goslack.NewTextBlockObject(goslack.PlainTextType, "w00f", true, false)
				}

				blockImage := goslack.NewImageBlock(imageURL, "w00f", "", blockText)
				msgOptions = append(msgOptions, goslack.MsgOptionBlocks(blockImage))
			}
		} else if len(text) > 0 {
			msgOptions = append(msgOptions, goslack.MsgOptionText(text, false))
		} else {
			return false, fmt.Errorf("No text or url for reply")
		}
	}

	a.client.PostMessage(ev.Channel)

	_, _, err := a.client.PostMessage(ev.Channel, msgOptions...)

	if err != nil {
		a.logger.Errorf("PostMessage: %s", err)
		return false, err
	}

	return true, nil
}

// -----------------------------------------------------------------------------

// NewAtChannelMention returns a new Actionner
func NewAtChannelMention(conf *config.Cerberus, logger *log.Logger, client *goslack.Client) Actionner {
	actionner := &AtChannelMention{
		config: conf,
		logger: logger,
		client: client,
	}

	return actionner
}

type AtChannelMention struct {
	config *config.Cerberus
	logger *log.Logger
	client *goslack.Client
}

func (a *AtChannelMention) Action(event interface{}) (bool, error) {
	ev := event.(*goslackevents.MessageEvent)

	// Message does not contain @channel mention
	if !strings.Contains(ev.Text, "<!channel>") {
		a.logger.Debugf("Text does not contain <!channel>: %s", ev.Text)
		return false, nil
	}

	ch, err := slack.GetConversationInfo(a.client, ev.Channel)

	if err != nil {
		a.logger.Errorf("%s", err)
		return false, err
	}

	// Channel is not a team channel
	if !strings.HasPrefix(ch.Name, "team-") {
		a.logger.Debugf("#%s is not a team channel", ch.Name)
		return false, nil
	} else {
		a.logger.Debugf("#%s is a team channel", ch.Name)
	}

	user, err := slack.GetUserInfo(a.client, ev.User)

	if err != nil {
		a.logger.Errorf("%s", err)
		return false, err
	}

	// Channel creator
	if ch.Creator == ev.User {
		a.logger.Debugf("@%s is the channel creator", user.Name)
		return false, nil
	}

	// User is Owner or Admin
	if user.IsOwner || user.IsAdmin {
		a.logger.Debugf("@%s is an admin or owner", user.Name)
		return false, nil
	}

	usergroup := strings.ReplaceAll(ch.Name, "-", "")
	group, err := slack.GetUserGroup(a.client, usergroup)

	if err != nil {
		a.logger.Errorf("%s", err)
		return false, err
	}

	if group != nil {
		// User is a member of the team's channel team
		for _, user := range group.Users {
			if user == ev.User {
				a.logger.Debugf("@%s is a member of @%s", ev.User, group.Handle)
				return false, nil
			}
		}
	}

	format := "Hello <@%s> :wave:\nIt seems that you are not a member of <!subteam^%s> therefor you should not mention @channel in <#%s>.\n" +
		"Please edit your message to use <!subteam^%s> to get the team's attention."
	message := fmt.Sprintf(format, ev.User, group.ID, ev.Channel, group.ID)
	channel, _, _, err := a.client.OpenConversation(&goslack.OpenConversationParameters{
		Users: []string{ev.User},
	})

	if err != nil {
		a.logger.Errorf("OpenConversation %s", err)
		return false, err
	}

	_, _, err = a.client.PostMessage(
		channel.ID,
		goslack.MsgOptionText(message, false),
	)

	if err != nil {
		a.logger.Errorf("PostMessage %s", err)
		return false, err
	}

	return true, nil
}
