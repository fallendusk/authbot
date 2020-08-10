package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/vharitonsky/iniflags"
)

type Config struct {
	// Token Discord API Bot Token
	Token string
	// Prefix for commands
	Prefix string
	// DefaultRole String containing the role name to put authetnicated users in
	DefaultRole string
	// BotCommand Name of the command to use for authenticaton function
	BotCommand string
	// WelcomeEnabled disable or enable new member welcome message
	WelcomeEnabled bool
	// WelcomeMessage Message to announce when a new user joins the server
	WelcomeMessage string
	// WelcomeChannel Channel ID of channel to send welcome messages to
	WelcomeChannel string
}

var config = Config{}

func init() {
	flag.StringVar(&config.Token, "token", "", "Bot Token")
	flag.StringVar(&config.Prefix, "prefix", "!", "Command prefix")
	flag.StringVar(&config.DefaultRole, "role", "Members", "Name of the role to place authenticated members in")
	flag.StringVar(&config.BotCommand, "cmd", "iam", "Bot command name")
	flag.BoolVar(&config.WelcomeEnabled, "welcome", false, "Enable new member welcome message")
	flag.StringVar(&config.WelcomeMessage, "welcomemsg", "Welcome {name}! Please set your in-game character with !iam server firstname lastname", "Message to send when a new user joins")
	flag.StringVar(&config.WelcomeChannel, "welcomechannel", "", "Channel ID of channel to send welcome messages to (Required if welcome is enabled)")
	iniflags.Parse()
}

func main() {
	// Create new discord session using token provided on command line via -token
	if config.Token == "" {
		fmt.Println("Missing bot token. Please specify via -token")
		os.Exit(1)
	}
	// Check that a channel and welcome message are set if welcome is enabled
	if config.WelcomeEnabled && config.WelcomeChannel == "" && config.WelcomeMessage == "" {
		fmt.Println("New user welcome is enabled, but welcome channel or welcome message is not set. Please check configuration!")
		os.Exit(1)
	}

	dg, err := discordgo.New("Bot " + config.Token)
	// Exit if we fail to connect for whatever reason
	if err != nil {
		fmt.Println("Error creating discord session: ", err)
		os.Exit(1)
	}

	// Register handlers for discord events
	dg.AddHandler(messageCreate)

	// Connect to discord
	err = dg.Open()
	if err != nil {
		fmt.Println("Error connecting to discord websocket: ", err)
		os.Exit(1)
	}

	// Set bot status
	err = dg.UpdateStatus(0, "github.com/fallendusk/authbot")
	if err != nil {
		fmt.Println("Failed to set activity", err)
	}

	fmt.Println("Bot connected to Discord! Press Ctrl-C to shutdown")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

func embedSuccess(content string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{},
		Color:       1022555,
		Title:       "Success!",
		Description: content}

	return embed
}

func embedFailure(content string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{},
		Color:       16711684,
		Title:       "Failure!",
		Description: content}

	return embed
}

func guildMemberAdd(s *discordgo.Session, g *discordgo.GuildMemberAdd) {

}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// ignore messages we send
	if m.Author.ID == s.State.User.ID {
		return
	}

	// if a message length is greater than 1 character and the prefix matches our set prefix,
	// split the command from the prefix and store the rest as arguments to be passed on to
	// the appropiate handler function
	if len(m.Content) > 1 && strings.HasPrefix(m.Content, config.Prefix) {
		messageArray := strings.Split(m.Content, " ")
		command := strings.ToLower(strings.TrimPrefix(messageArray[0], config.Prefix))
		args := messageArray[1:]

		switch command {
		case config.BotCommand:
			authHandler(s, m, args)
		}
	}
}

func authHandler(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
	// Bomb out if there's less than 3 arguments
	if len(args) < 3 {
		s.ChannelMessageSendEmbed(m.ChannelID, embedFailure("Missing argument. Please use !iam servername firstname lastname"))
		return
	}

	// Build character name
	characterFirstName := strings.Title(args[1])
	characterLastName := strings.Title(args[2])
	characterName := characterFirstName + " " + characterLastName

	// Attempt to change nickname
	err := s.GuildMemberNickname(m.GuildID, m.Author.ID, characterName)
	if err != nil {
		fmt.Println("Failed to change nickname for ", m.Author.Username)
		fmt.Println(err)
		//return
	}

	// Add default role to member
	dguild, err := s.Guild(m.GuildID)
	if err != nil {
		fmt.Println(err)
		return
	}

	role := getGuildRoleByName(config.DefaultRole, dguild)
	err = s.GuildMemberRoleAdd(m.GuildID, m.Author.ID, role)
	if err != nil {
		msg := "Failed to add role " + config.DefaultRole + " to " + m.Author.Username
		fmt.Println(msg)
		fmt.Println(err)
		s.ChannelMessageSendEmbed(m.ChannelID, embedFailure(msg))
		return
	}

	// Success! Let the user know
	s.ChannelMessageSendEmbed(m.ChannelID, embedSuccess("<@"+m.Author.ID+"> authenticated as **"+characterName+"**"))
}

func getGuildRoleByName(name string, guild *discordgo.Guild) string {
	for _, role := range guild.Roles {
		if role.Name == name {
			return role.ID
		}
	}
	return ""
}
