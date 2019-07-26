package main

import (
	"fmt"
	"log"
	"os"

	"github.com/nlopes/slack"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("You need to specify channel ID to close")
	}

	api := slack.New(os.Getenv("SLACK_OAUTH_TOKEN"))

	id := os.Args[1]
	channel, err := api.GetConversationInfo(id, false)
	if err != nil {
		log.Fatalf("Seems like you have specified wrong channel ID\nID: %s\nError: %s\n", id, err)
	}
	if channel.IsShared {
		log.Printf("I guess I shouldn't close this shared channel... <#%s|%s>\n", channel.ID, channel.Name)
		os.Exit(2)
	}
	if channel.IsArchived {
		log.Printf("Channel has been archived already <#%s|%s>\n", channel.ID, channel.Name)
		os.Exit(2)
	}

	fmt.Printf("Archiving channel <#%s|%s>...\n", channel.ID, channel.Name)
	_, _, err = api.PostMessage(
		channel.ID,
		slack.MsgOptionText("ごきげんよう、マクゴナガルです。こちらの部屋は四半期以上の間動きがありませんので、一度閉じさせていただきます。\nもしまた部屋を開きたい場合は管理部まで申請をお願いします。", false),
	)
	if err != nil {
		log.Fatalf("Couldn't post a notification message, aborting\nError: %s", err)
	}

	err = api.ArchiveConversation(channel.ID)
	if err != nil {
		log.Fatalf("Some thing wrong have happened on archiving a channel\nError: %s\n", err)
	}
}
