package slack

import (
	"fmt"
	"time"

	goslack "github.com/slack-go/slack"
	qdcache "github.com/sylr/go-libqd/cache"
)

var (
	userGroupsCache       = qdcache.GetMeteredCache(5*time.Minute, 5*time.Minute)
	userGroupMembersCache = qdcache.GetMeteredCache(2*time.Minute, 2*time.Minute)
)

func InvalidateGroupCache(usergroup string) {
	userGroupsCache.Flush()
	userGroupMembersCache.Delete(usergroup)
}

func GetUserGroup(client *goslack.Client, usergroup string) (*goslack.UserGroup, error) {
	groups, err := GetUserGroups(client)

	if err != nil {
		return nil, fmt.Errorf("slack.GetUserGroup: %w", err)
	}

	for _, group := range groups {
		if group.Handle == usergroup {
			return &group, nil
		}
	}

	return nil, nil
}

func GetUserGroups(client *goslack.Client) ([]goslack.UserGroup, error) {
	groups, found := userGroupsCache.Get("groups")

	if found {
		return groups.([]goslack.UserGroup), nil
	}

	groups, err := client.GetUserGroups(goslack.GetUserGroupsOptionIncludeUsers(true))

	if err != nil {
		return nil, fmt.Errorf("slack.GetUserGroups: %w", err)
	}

	userGroupsCache.Set("groups", groups, 0)

	return groups.([]goslack.UserGroup), nil
}

func GetUserGroupMembers(client *goslack.Client, usergroup string) ([]string, error) {
	cmembers, found := userGroupMembersCache.Get(usergroup)

	if found {
		return cmembers.([]string), nil
	}

	members, err := client.GetUserGroupMembers(usergroup)

	if err != nil {
		return nil, fmt.Errorf("slack.GetUserGroupMembers: %w", err)
	}

	userGroupMembersCache.Set(usergroup, members, 0)

	return members, nil
}
