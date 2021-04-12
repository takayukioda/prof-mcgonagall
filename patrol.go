package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

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
	var debugChannel slack.Channel

	api := slack.New(os.Getenv("SLACK_OAUTH_TOKEN"))
	//api := slack.New(os.Getenv("SLACK_OAUTH_TOKEN"), slack.OptionDebug(true))

	channels := getAllConversations(api)
	fmt.Printf("Found %d channels\n", len(channels))

	ylist := make([]slack.Channel, 0)
	rlist := make([]slack.Channel, 0)

	for _, channel := range channels {
		if channel.IsShared {
			continue
		}
		if channel.Name == "room-debug" {
			debugChannel = channel
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

		if inactive < month {
			continue
		}

		if inactive > 3*month {
			rlist = append(rlist, channel)
			continue
		}

		ylist = append(ylist, channel)
	}

	var ys []string = make([]string, 0, len(ylist))
	for _, y := range ylist {
		ys = append(ys, fmt.Sprintf("<#%s|%s> (%s)", y.ID, y.Name, y.ID))
	}
	api.PostMessage(
		debugChannel.ID,
		slack.MsgOptionText(
			fmt.Sprintf(`Students!!
Here, I have a list. A list of channels that are inactive for more than a month.
Keep in your mind that if those channels are continuously inactive, I will archive those channels.

Here are those channels.
%s`, strings.Join(ys, "\n")),
			false,
		),
	)

	var rs []string = make([]string, 0, len(rlist))
	for _, r := range rlist {
		rs = append(rs, fmt.Sprintf("<#%s|%s> (%s)", r.ID, r.Name, r.ID))
	}
	api.PostMessage(
		debugChannel.ID,
		slack.MsgOptionText(
			fmt.Sprintf(`Everyone!!
Today, I have something to share. I will share channels, that will be archived.
These channels were abandoned, kept inactive for more than a quarter.

Here are those channels.
%s`, strings.Join(rs, "\n")),
			false,
		),
	)
}
