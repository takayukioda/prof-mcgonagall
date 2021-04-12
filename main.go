package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nlopes/slack"
)

const SLACK_DEFAULT_LIMIT = 100
const DEBUG_CHANNEL_ID = "*****"

const month = time.Hour * 24 * 30

func parseSlackTimestamp(ts string) time.Time {
	split := strings.Split(ts, ".")
	s, _ := strconv.ParseInt(split[0], 10, 64)
	ns, _ := strconv.ParseInt(split[1], 10, 64)
	return time.Unix(s, ns)
}

func getAllChannels(api *slack.Client) []slack.Channel {
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

func getLastMessage(api *slack.Client, channel *slack.Channel) (*slack.Message, error) {
	history, err := api.GetChannelHistory(channel.ID, slack.HistoryParameters{
		Count: SLACK_DEFAULT_LIMIT,
	})
	if err != nil {
		fmt.Printf("[%s] Failed to get a history: %s\n", channel.Name, err)
		return nil, err
	}

	var message *slack.Message
	for _, msg := range history.Messages {
		if msg.Type == "message" {
			message = &msg
			break
		}
	}

	return message, nil
}

func substring(text string, limit int) string {
	l := len(text)
	var max int = 140 - 1

	if l < max {
		max = l - 1
	}
	if max < 0 {
		max = 0
	}
	return text[:max]
}

type InactiveChannel struct {
	since   time.Duration
	channel slack.Channel
}

func GetInactivePeriod(api *slack.Client, channel *slack.Channel, ch chan time.Duration) {
	fmt.Printf("Check for #%s\n", channel.Name)
	message, err := getLastMessage(api, channel)
	if err != nil {
		log.Printf("Got an error to retrieve message for #%s\n", channel.Name)
		close(ch)
	}
	since := time.Since(parseSlackTimestamp(message.Timestamp))
	ch <- since
}

func GetAllInactiveChannels(api *slack.Client, channels []slack.Channel, excludes []string) []InactiveChannel {
	var inactives = make([]InactiveChannel, 0)
	ch := make(chan time.Duration, 10)
	for _, channel := range channels {
		for _, ex := range excludes {
			if channel.Name == ex {
				fmt.Printf("Found in the exclude list, skipping: %s(%s)\n", channel.Name, channel.ID)
				continue
			}
		}
		go GetInactivePeriod(api, &channel, ch)
		since, ok := <-ch
		if !ok {
			continue
		}

		if since < month {
			// Channel is active and no need to close
			continue
		}

		inactives = append(inactives, InactiveChannel{
			since:   since,
			channel: channel,
		})
	}
	return inactives
}

func Patrol(api *slack.Client) {
	channels := getAllChannels(api)

	targets := make([]slack.Channel, 0)
	for _, channel := range channels {
		if channel.IsGeneral {
			continue
		}
		if channel.IsShared {
			continue
		}
		if channel.Name == "random" {
			continue
		}
		targets = append(targets, channel)
	}
	fmt.Printf("Found %d channels\n", len(targets))

	excludes := []string{"room-debug"}
	inactives := GetAllInactiveChannels(api, targets, excludes)

	for _, inactive := range inactives {
		days := int(inactive.since.Hours()) / 24

		channel := inactive.channel
		fmt.Printf("🔥 <#%s|%s> (ID:%s) is inactive for %d days\n", channel.ID, channel.Name, channel.ID, days)
	}
}

func Usage() {
	fmt.Fprintf(os.Stderr, "$ %s <command>\n", os.Args[0])
}

func main() {
	if len(os.Args) < 2 {
		Usage()
		log.Fatalln("You need to specify sub command to run")
	}
	api := slack.New(os.Getenv("SLACK_OAUTH_TOKEN"))
	cmd := os.Args[1]
	switch cmd {
	case "patrol":
		Patrol(api)
	default:
		Usage()
		log.Fatalln("No such subcommand have defined: ", cmd)
	}
}
