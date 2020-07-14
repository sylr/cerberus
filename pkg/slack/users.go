package slack

import (
	"fmt"
	"time"

	goslack "github.com/slack-go/slack"
	qdcache "github.com/sylr/go-libqd/cache"
)

var (
	userInfoCache = qdcache.GetMeteredCache(2*time.Minute, 2*time.Minute)
)

func GetUserInfo(client *goslack.Client, user string) (*goslack.User, error) {
	cuser, found := userInfoCache.Get(user)

	if found {
		return cuser.(*goslack.User), nil
	}

	u, err := client.GetUserInfo(user)

	if err != nil {
		return nil, fmt.Errorf("slack.GetUserInfo: %w", err)
	}

	userInfoCache.Set(user, u, 0)

	return u, nil
}
