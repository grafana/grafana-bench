package validate

import (
	"fmt"

	"github.com/grafana/grafana-bench/pkg/notifier"
	"github.com/slack-go/slack"
)

type ChannelStatus struct {
	ID   string
	Name string
	Err  error
}

// CheckPermissions checks whether grafana-bench bot has permissions to post in the channel that
// the mapping is pointing to
func CheckPermissions(codeownersMapPath string, slackToken string) ([]ChannelStatus, error) {
	codeownersMap, err := notifier.NewCodeownersMapping(codeownersMapPath)
	if err != nil {
		return nil, err
	}

	client := slack.New(slackToken)
	// for each item in the mapping and check whether the slack bot is a member of that channel
	var channelStatuses []ChannelStatus
	for _, m := range codeownersMap {
		channelStatuses = append(channelStatuses, isMember(client, m))
	}

	return channelStatuses, nil
}

// Checks if slack client has permission to post to a given channel
func isMember(client *slack.Client, channelID string) ChannelStatus {
	// Get conversation
	channel, err := client.GetConversationInfo(&slack.GetConversationInfoInput{
		ChannelID: channelID,
	})

	channelStatus := ChannelStatus{
		ID: channelID,
	}

	if err != nil {
		if slackErr, ok := err.(*slack.SlackErrorResponse); ok {
			switch slackErr.Err {
			case "channel_not_found":
				channelStatus.Err = fmt.Errorf("channel not found or bot doesn't have access")
			case "not_in_channel":
				channelStatus.Err = fmt.Errorf("bot is not a member of this channel")
			default:
				channelStatus.Err = fmt.Errorf("error accessing channel: %s", slackErr.Err)
			}
		} else {
			channelStatus.Err = err
		}
		return channelStatus
	}

	channelStatus.Name = channel.Name

	// Check if bot is a member
	if !channel.IsMember {
		channelStatus.Err = fmt.Errorf("bot is not a member of channel")
	}

	return channelStatus
}
