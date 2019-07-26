package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hako/durafmt"
	"github.com/nlopes/slack"
)

var month = time.Hour * 24 * 30

func parseSlackTimestamp(ts string) time.Time {
	split := strings.Split(ts, ".")
	s, _ := strconv.ParseInt(split[0], 10, 64)
	ns, _ := strconv.ParseInt(split[1], 10, 64)
	return time.Unix(s, ns)
}

func getAllConversations(api *slack.Client) []slack.Channel {

	params := &slack.GetConversationsParameters{
		ExcludeArchived: "true",
		Limit:           100,
		Types:           []string{"public_channel"},
	}

	var channels []slack.Channel
	var next *string

	for next == nil || *next != "" {
		convs, cursor, err := api.GetConversations(params)
		if err != nil {
			fmt.Printf("%s\n", err)
		}
		next = &cursor

		params.Cursor = *next
		channels = append(channels, convs...)
	}
	return channels
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("You need to specify one or more channel ID or name to check")
	}
	checklist := os.Args[1:]

	api := slack.New(os.Getenv("SLACK_OAUTH_TOKEN"))
	//api := slack.New(os.Getenv("SLACK_OAUTH_TOKEN"), slack.OptionDebug(true))

	channels := getAllConversations(api)
	fmt.Printf("Found %d channels\n", len(channels))

	for _, channel := range channels {
		needCheck := false
		for _, check := range checklist {
			if channel.ID == check || channel.Name == check {
				needCheck = true
			}
		}
		if !needCheck {
			continue
		}

		history, err := api.GetChannelHistory(channel.ID, slack.HistoryParameters{
			Count: 1,
		})
		if err != nil {
			fmt.Printf("[%s] Failed to get a history: %s\n", channel.Name, err)
		}
		lastMessage := history.Messages[0]
		last := parseSlackTimestamp(lastMessage.Timestamp)
		inactive := time.Since(last)

		fmt.Printf(
			"Channel <#%s|%s> is inactive for %s\n",
			channel.ID,
			channel.Name,
			durafmt.ParseShort(inactive).String(),
		)
		fmt.Printf(
			"Last activity is %#v",
			lastMessage,
		)
	}
}
