package slack

import (
	"fmt"
	"time"

	goslack "github.com/slack-go/slack"
	qdcache "github.com/sylr/go-libqd/cache"
)

var (
	channelInfoCache = qdcache.GetMeteredCache(2*time.Minute, 2*time.Minute)
)

func GetConversationInfo(client *goslack.Client, channel string) (*goslack.Channel, error) {
	cchan, found := channelInfoCache.Get(channel)

	if found {
		return cchan.(*goslack.Channel), nil
	}

	c, err := client.GetConversationInfo(channel, false)

	if err != nil {
		return nil, fmt.Errorf("slack.GetConversationInfo: %w", err)
	}

	userInfoCache.Set(channel, c, 0)

	return c, nil
}
