package autodelete

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const (
	discordInteractionCreateEvent     = "INTERACTION_CREATE"
	discordInteractionApplicationCmd  = 2
	discordInteractionResponseMessage = 4

	discordAppCommandChatInput       = 1
	discordAppCommandOptionSub       = 1
	discordAppCommandOptionString    = 3
	discordAppCommandOptionInteger   = 4
	discordInteractionFlagEphemeral  = 64
)

type slashApplicationCommand struct {
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	Type        int                          `json:"type,omitempty"`
	Options     []slashApplicationCmdOption `json:"options,omitempty"`
	DMPermission *bool                       `json:"dm_permission,omitempty"`
}

type slashApplicationCmdOption struct {
	Type        int                         `json:"type"`
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Required    bool                        `json:"required,omitempty"`
	Options     []slashApplicationCmdOption `json:"options,omitempty"`
}

type slashInteraction struct {
	ID        string                `json:"id"`
	Type      int                   `json:"type"`
	Data      slashInteractionData  `json:"data"`
	GuildID   string                `json:"guild_id"`
	ChannelID string                `json:"channel_id"`
	Member    *slashInteractionUser `json:"member"`
	User      *discordgo.User       `json:"user"`
	Token     string                `json:"token"`
}

type slashInteractionUser struct {
	User *discordgo.User `json:"user"`
}

type slashInteractionData struct {
	Name    string                   `json:"name"`
	Options []slashInteractionOption `json:"options"`
}

type slashInteractionOption struct {
	Type    int                      `json:"type"`
	Name    string                   `json:"name"`
	Value   interface{}              `json:"value,omitempty"`
	Options []slashInteractionOption `json:"options,omitempty"`
}

type slashInteractionResponse struct {
	Type int                           `json:"type"`
	Data slashInteractionResponseData  `json:"data,omitempty"`
}

type slashInteractionResponseData struct {
	Content string `json:"content,omitempty"`
	Flags   int    `json:"flags,omitempty"`
}

func (b *Bot) slashCommandsEnabled() bool {
	return b.Config.SlashCommands == nil || *b.Config.SlashCommands
}

func (b *Bot) RegisterSlashCommands() error {
	if !b.slashCommandsEnabled() {
		fmt.Println("[slash] slash command registration disabled")
		return nil
	}
	if b.ClientID == "" {
		fmt.Println("[slash] clientid is empty; skipping slash command registration")
		return nil
	}

	noDMs := false
	cmd := slashApplicationCommand{
		Name:        "autodelete",
		Description: "Configure AutoDelete in the current channel.",
		Type:        discordAppCommandChatInput,
		DMPermission: &noDMs,
		Options: []slashApplicationCmdOption{
			{
				Type:        discordAppCommandOptionSub,
				Name:        "help",
				Description: "Show AutoDelete command help.",
			},
			{
				Type:        discordAppCommandOptionSub,
				Name:        "check",
				Description: "Check the AutoDelete settings for this channel.",
			},
			{
				Type:        discordAppCommandOptionSub,
				Name:        "set",
				Description: "Set or disable AutoDelete for this channel.",
				Options: []slashApplicationCmdOption{
					{
						Type:        discordAppCommandOptionString,
						Name:        "duration",
						Description: "Delete messages after this duration, for example 30m or 24h.",
					},
					{
						Type:        discordAppCommandOptionInteger,
						Name:        "count",
						Description: "Delete oldest messages after this many live messages. Use 0 with duration 0s to disable.",
					},
				},
			},
		},
	}

	url := fmt.Sprintf("%sapplications/%s/commands", discordgo.EndpointAPI, b.ClientID)
	_, err := b.s.Request("POST", url, cmd)
	if err != nil {
		return err
	}
	fmt.Println("[slash] registered /autodelete command")
	return nil
}

func (b *Bot) OnRawEvent(s *discordgo.Session, ev *discordgo.Event) {
	if ev == nil || ev.Type != discordInteractionCreateEvent {
		return
	}

	var it slashInteraction
	if err := json.Unmarshal(ev.RawData, &it); err != nil {
		fmt.Println("[slash] could not parse interaction:", err)
		return
	}
	if it.Type != discordInteractionApplicationCmd || it.Data.Name != "autodelete" {
		return
	}

	if err := b.HandleSlashCommand(&it); err != nil {
		fmt.Println("[slash] command error:", err)
		_ = b.respondSlash(&it, "AutoDelete could not handle that command: "+err.Error(), true)
	}
}

func (b *Bot) HandleSlashCommand(it *slashInteraction) error {
	if it.ChannelID == "" || it.GuildID == "" {
		return b.respondSlash(it, "AutoDelete slash commands can only be used inside a server channel.", true)
	}
	if len(it.Data.Options) == 0 {
		return b.respondSlash(it, textHelp, true)
	}

	sub := it.Data.Options[0]
	switch sub.Name {
	case "help":
		return b.respondSlash(it, textHelp, true)
	case "check":
		return b.handleSlashCheck(it)
	case "set":
		return b.handleSlashSet(it, sub.Options)
	default:
		return b.respondSlash(it, "Unknown AutoDelete subcommand.", true)
	}
}

func (b *Bot) handleSlashCheck(it *slashInteraction) error {
	user := it.commandUser()
	if user == nil {
		return b.respondSlash(it, "Could not identify the command user.", true)
	}
	if ok, err := b.userCanManageMessages(user.ID, it.ChannelID); err != nil {
		return b.respondSlash(it, "Could not check your permissions: "+err.Error(), true)
	} else if !ok {
		return b.respondSlash(it, "You must have the Manage Messages permission to check AutoDelete settings.", true)
	}

	msg, err := b.ChannelSettingsMessage(it.ChannelID)
	if err != nil {
		return b.respondSlash(it, fmt.Sprintf("Error checking settings: %v", err), true)
	}
	return b.respondSlash(it, msg, true)
}

func (b *Bot) handleSlashSet(it *slashInteraction, opts []slashInteractionOption) error {
	user := it.commandUser()
	if user == nil {
		return b.respondSlash(it, "Could not identify the command user.", true)
	}

	args, err := slashSetArgs(opts)
	if err != nil {
		return b.respondSlash(it, err.Error(), true)
	}
	if len(args) == 0 {
		return b.respondSlash(it, "Provide a duration, a count, or both. Use count 0 and duration 0s to disable AutoDelete.", true)
	}
	if ok, err := b.userCanManageMessages(user.ID, it.ChannelID); err != nil {
		return b.respondSlash(it, "Could not check your permissions: "+err.Error(), true)
	} else if !ok {
		return b.respondSlash(it, "You must have the Manage Messages permission to change AutoDelete settings.", true)
	}

	if err := b.respondSlash(it, "Applying AutoDelete settings in this channel. The bot will post the result shortly.", true); err != nil {
		return err
	}

	go CommandModify(b, &discordgo.Message{
		ID:        it.ID,
		ChannelID: it.ChannelID,
		GuildID:   it.GuildID,
		Author:    user,
		Content:   "/autodelete set " + strings.Join(args, " "),
	}, args)
	return nil
}

func (it *slashInteraction) commandUser() *discordgo.User {
	if it.Member != nil && it.Member.User != nil {
		return it.Member.User
	}
	return it.User
}

func (b *Bot) userCanManageMessages(userID, channelID string) (bool, error) {
	permissions, err := b.s.UserChannelPermissions(userID, channelID)
	if err != nil {
		return false, err
	}
	return permissions&discordgo.PermissionManageMessages != 0, nil
}

func slashSetArgs(opts []slashInteractionOption) ([]string, error) {
	var args []string
	for _, opt := range opts {
		switch opt.Name {
		case "duration":
			duration, ok := opt.Value.(string)
			if !ok || duration == "" {
				return nil, fmt.Errorf("duration must be a value like 30m or 24h")
			}
			args = append(args, duration)
		case "count":
			count, err := slashIntegerValue(opt.Value)
			if err != nil {
				return nil, fmt.Errorf("count must be a whole number")
			}
			args = append(args, strconv.FormatInt(count, 10))
		}
	}
	return args, nil
}

func slashIntegerValue(v interface{}) (int64, error) {
	switch n := v.(type) {
	case float64:
		return int64(n), nil
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case json.Number:
		return n.Int64()
	case string:
		return strconv.ParseInt(n, 10, 64)
	default:
		return 0, fmt.Errorf("unsupported integer value %T", v)
	}
}

func (b *Bot) respondSlash(it *slashInteraction, content string, ephemeral bool) error {
	flags := 0
	if ephemeral {
		flags = discordInteractionFlagEphemeral
	}
	url := fmt.Sprintf("%sinteractions/%s/%s/callback", discordgo.EndpointAPI, it.ID, it.Token)
	_, err := b.s.Request("POST", url, slashInteractionResponse{
		Type: discordInteractionResponseMessage,
		Data: slashInteractionResponseData{
			Content: content,
			Flags:   flags,
		},
	})
	return err
}
