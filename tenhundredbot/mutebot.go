package tenhundredbot

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"os"
	"strings"
	"sync"
	"tenhundredmuter/tenhundredbot/wordMap"
)

// TenHundredBot is used to restrict people to using only the words in the WordsFile field of struct TenHundredBotConfig.
type TenHundredBot struct {
	session         *discordgo.Session
	wordStore       *wordMap.WordMap
	botToken        string
	commandPrefix   string
	serverID        string
	mutedChannelID  string
	mutedUsers      []string
	AfterUserUpdate func()
	mutex           sync.Mutex
}

// TenHundredBotConfig is used with NewTenHundredBot.
type TenHundredBotConfig struct {
	// The only words a muted user is allowed to say.
	WordsFile string `json:"wordsFile"`
	// Command prefix this bot listens to.
	CommandPrefix string `json:"commandPrefix"`
	// BotToken provided by Discord.
	BotToken string `json:"botToken"`
	// ServerID this specific bot operates on.
	ServerID string `json:"serverID"`
	// Muted User IDs that can only talk with the words in WordsFile.
	MutedUsers []string `json:"mutedUsers"`
	// Channel where everyone is restricted to words in WordsFile.
	MutedChannelID string `json:"mutedChannelID"`
	// Called after every change to MutedUsers.
	AfterUserUpdate func()
}

// NewTenHundredBot returns a TenHundredBot.
func NewTenHundredBot(config TenHundredBotConfig) (th *TenHundredBot) {
	ws := getWordStore(config.WordsFile)
	th = &TenHundredBot{
		session:        nil,
		wordStore:      ws,
		botToken:       config.BotToken,
		commandPrefix:  config.CommandPrefix,
		serverID:       config.ServerID,
		mutedChannelID: config.MutedChannelID,
		mutedUsers:     config.MutedUsers,
		mutex:          sync.Mutex{},
		AfterUserUpdate: func() {

		},
	}
	return th
}

// Serve connects the bot to Discord to operate with the configFile passed to the NewTenHundredBot function.
func (th *TenHundredBot) Serve(ctx context.Context) {
	dg, err := discordgo.New("Bot " + th.botToken)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}
	defer dg.Close()

	dg.AddHandler(func(sess *discordgo.Session, m *discordgo.MessageCreate) {
		th.HandlerMessageCreate(sess, m)
	})

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	fmt.Println("Serving")
	<-ctx.Done()
}

// MutedUsers returns the serverID associated with this bot.
func (th *TenHundredBot) ServerID() string {
	return th.serverID
}

// MutedUsers returns a string array of muted user IDs.
func (th *TenHundredBot) MutedUsers() []string {
	return th.mutedUsers
}

// MutedChannelID gets the channel ID where everyone can only use the words in WordsFile.
func (th *TenHundredBot) MutedChannelID() string {
	return th.mutedChannelID
}

// CommandPrefix gets the command prefix this bot responds to.
func (th *TenHundredBot) CommandPrefix() string {
	return th.commandPrefix
}

// SetAfterUpdateFunc sets the update function ran after the slice of muted users updates.
func (th *TenHundredBot) SetAfterUpdateFunc(update func()) {
	th.AfterUserUpdate = update
}

func userExistsInGuild(s *discordgo.Session, guildID, userID string) bool {
	_, err := s.GuildMember(guildID, userID)
	if err != nil {
		fmt.Println(err)
		return false
	}

	return true
}

func inSlice(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// todo: combat against message edits
func (th *TenHundredBot) HandlerMessageCreate(sess *discordgo.Session, msgEv *discordgo.MessageCreate) { //todo: consider moving all mutex to HandlerMessageCreate
	defer th.mutex.Unlock()
	th.mutex.Lock()
	th.session = sess                                                                                      // sess is used later in other pointer receiver functions
	if msgEv.Author.ID == th.session.State.User.ID || msgEv.GuildID == "" || msgEv.GuildID != th.serverID{ // is bot's own message || is a PM/DM || is not of this guild
		return
	}

	if hasPrefix(th, msgEv.Content) && userCanAddBots(sess, msgEv.Author.ID, msgEv.ChannelID) {
		th.processCommands(msgEv)
		return
	}
	th.decideMessageRemoval(msgEv)
}

func userCanAddBots(sess *discordgo.Session, userID, channelID string) bool {
	aperms, err := sess.UserChannelPermissions(userID, channelID)
	if err != nil {
		log.Println("userCanAddBots", err)
	}
	return aperms&discordgo.PermissionManageServer != 0
}

func (th *TenHundredBot) decideMessageRemoval(msgEv *discordgo.MessageCreate) {

	if inSlice(th.mutedUsers, msgEv.Author.ID) || msgEv.ChannelID == th.mutedChannelID {
		badWords := badWords(msgEv, th.wordStore)
		if !(len(badWords) > 0) {
			return
		}
		th.session.ChannelMessageDelete(msgEv.ChannelID, msgEv.ID)

		notice := "You can only talk with the ten hundred most used words now. \nhttps://github.com/f0nkey/tenhundred\nThese words are not simple: "
		if msgEv.ChannelID == th.mutedChannelID {
			notice = "You can only talk with the ten hundred most used words in this channel. \nhttps://github.com/f0nkey/tenhundred\nThese words are not simple: "
		}

		for i, badWord := range badWords {
			notice += badWord
			if i != len(badWords)-1 {
				notice += ", "
			}
		}
		notice += "\nPlease try again."
		if userCanAddBots(th.session, msgEv.Author.ID, msgEv.ChannelID) && !(msgEv.ChannelID == th.mutedChannelID) {
			notice += "\nPaste `" + th.commandPrefix + " unmute " + msgEv.Author.ID + "` in the server to unmute yourself. [This line is displayed to users who can add bots only]"
		}

		sendPrivateMessage(th.session, msgEv.Author.ID, notice)
	}
}

// todo: allow punctuation, strip punctuation when testing
// todo: check if player target is actually a playerID
func (th *TenHundredBot) processCommands(msgEv *discordgo.MessageCreate) {
	msg := strings.Split(msgEv.Content, " ")
	if !(len(msg) >= 2) {
		return
	}
	cmd := msg[1]

	if cmd == "help" {
		sendPMHelp(th.session, msgEv.Author.ID, th.commandPrefix)
		return
	}

	defer th.RunAfterUserUpdate() // the rest of the commands will update some value that needs to be written
	if cmd == "set" {
		if th.mutedChannelID == msgEv.ChannelID {
			th.session.ChannelMessageSend(msgEv.ChannelID, "This area is already only allows simple words.")
			return
		}
		th.mutedChannelID = msgEv.ChannelID
		th.session.ChannelMessageSend(msgEv.ChannelID, "All people are only allowed to talk with simple words in this area now.")
		return
	}

	if cmd == "rem" {
		if th.mutedChannelID == "" {
			th.session.ChannelMessageSend(msgEv.ChannelID, "There is no place to remove simple talk policing.")
			return
		}
		th.mutedChannelID = ""
		th.session.ChannelMessageSend(msgEv.ChannelID, "Removed simple talk policing in this area.")
		return
	}

	if len(msg) >= 3 {
		thirdArgument := parseUserID(msg[2]) // userID for mute, unmute todo: check if user exists on server

		if cmd == "prefix" {
			th.commandPrefix = thirdArgument
			sendPrivateMessage(th.session, msgEv.Author.ID, "This Bot will now respond to **"+th.commandPrefix+"**")
			return
		}

		if cmd == "mute" {
			th.muteProcedure(thirdArgument, msgEv)
			return
		}

		if cmd == "unmute" {
			th.unmuteProcedure(thirdArgument, msgEv)
			return
		}
	}
	sendPrivateMessage(th.session, msgEv.Author.ID, "Invalid command. Try "+th.commandPrefix+" mute @USER")
	th.session.ChannelMessageDelete(msgEv.ChannelID, msgEv.ID)
	return
}

func (th *TenHundredBot) unmuteProcedure(targetUser string, msgEv *discordgo.MessageCreate) {
	if !userExistsInGuild(th.session, msgEv.GuildID, targetUser) {
		sendPrivateMessage(th.session, msgEv.Author.ID, "User does not exist in this server.")
		return
	}

	alreadyUnmuted := th.unmuteUser(th.session, targetUser)
	if alreadyUnmuted {
		sendPrivateMessage(th.session, msgEv.Author.ID, "That user is not muted.")
		return
	}
	th.session.ChannelMessageSend(msgEv.ChannelID, "<@"+targetUser+"> can talk with any words now.")
	return
}

func (th *TenHundredBot) RunAfterUserUpdate() {
	th.AfterUserUpdate()
}

func (th *TenHundredBot) muteProcedure(targetUser string, msgEv *discordgo.MessageCreate) {
	if !userExistsInGuild(th.session, msgEv.GuildID, targetUser) {
		sendPrivateMessage(th.session, msgEv.Author.ID, "User does not exist in this server.")
		return
	}

	alreadyMuted := th.muteUser(targetUser)
	if alreadyMuted {
		sendPrivateMessage(th.session, msgEv.Author.ID, "That user is already muted.")
		return
	}
	th.session.ChannelMessageSend(msgEv.ChannelID, "<@"+targetUser+"> can only talk with the ten hundred most used words now (simple words).")
	return
}

func (th *TenHundredBot) muteUser(targetUser string) (alreadyMuted bool) {
	if inSlice(th.mutedUsers, targetUser) {
		return true
	}
	th.mutedUsers = append(th.mutedUsers, targetUser)
	return false
}

func (th *TenHundredBot) unmuteUser(s *discordgo.Session, targetUser string) (alreadyUnmuted bool) {
	userIndex := -1
	for i, name := range th.mutedUsers {
		if targetUser == name {
			userIndex = i
		}
	}
	if userIndex == -1 {
		return true
	}
	th.mutedUsers = append(th.mutedUsers[:userIndex], th.mutedUsers[userIndex+1:]...)
	return false
}

func hasPrefix(th *TenHundredBot, msg string) bool {
	botMention := "<@!"+th.session.State.User.ID+">"
	prefixIsMention :=  len(msg) > len(botMention) && msg[0:len(botMention)] == botMention

	prefixIsCmdPrefix := len(msg) > len(th.commandPrefix + " ") && msg[0:len(th.commandPrefix)+1]  == th.commandPrefix + " " //adding a space; if !t, then it would detect !th
	return prefixIsCmdPrefix || prefixIsMention
}

func sendPrivateMessage(s *discordgo.Session, userID, msg string) {
	chann, err := s.UserChannelCreate(userID) // uses existing channel if its already created
	if err != nil && strings.Contains(err.Error(), "Cannot send messages to this user") {
		return
	}
	if err != nil {
		log.Println("sendPrivateMessage:", err)
		return
	}
	s.ChannelMessageSend(chann.ID, msg)
}

func sendPMHelp(session *discordgo.Session, userID string, cmdPrefix string) {
	line0 := fmt.Sprintln("Visit https://github.com/f0nkey/tenhundred for more help.")
	line1 := fmt.Sprintf("Commands (**%v** can be replaced with **@TenHundredBot**):\n", cmdPrefix)
	cmd1 := fmt.Sprintf("**%v set (yourNewPrefix)** - Only allows the 1000 most common words in the channel this is ran in.\n", cmdPrefix)
	cmd2 := fmt.Sprintf("**%v rem (yourNewPrefix)** - Removes the restriction to the currently set channel.\n", cmdPrefix)
	cmd3 := fmt.Sprintf("**%v unmute (@User)** - Unmutes a user\n", cmdPrefix)
	cmd4 := fmt.Sprintf("**%v mute (@User)** - Restricts a user to using only the 1000 most common words.\n", cmdPrefix)
	cmd5 := fmt.Sprintf("**%v prefix (yourNewPrefix)** - Changes the prefix this bot responds to. Currently set to **%v **\n", cmdPrefix, cmdPrefix)

	sendPrivateMessage(session, userID, line0+line1+cmd1+cmd2+cmd3+cmd4+cmd5)
}

func parseUserID(s string) string {
	mentionPrefix := "<@!"
	if len(s) > len(mentionPrefix) && s[:len(mentionPrefix)] == mentionPrefix { // they sent a mention
		return s[len(mentionPrefix) : len(s)-1]
	}
	return s // they sent a userID
}

func badWords(msgEv *discordgo.MessageCreate, wordStore *wordMap.WordMap) []string {
	words := strings.Split(msgEv.Content, " ")
	badWords := []string{}
	for _, word := range words {
		if !wordStore.Exists(word) {
			badWords = append(badWords, word)
		}
	}
	return badWords
}

func getWordStore(fileName string) *wordMap.WordMap {
	f, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	ws, err := wordMap.NewWordMap(f)
	err = f.Close()
	if err != nil {
		log.Fatal(err)
	}
	return ws
}

// todo: add a max limit to mutedUsers and others
// todo: do not tell user their prefix command is of the not allowed words
// todo: don't let people mute the bot itself