package muteBot

import (
	"context"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
	"os"
	"strings"
	"sync"
	"tenhundredmuter/muteBot/wordMap"
)

// MuteBot is used to restrict people to using only the words in the WordsFile field of struct MuteBotConfig.
type MuteBot struct {
	session         *discordgo.Session
	wordStore       *wordMap.WordMap
	botToken        string
	commandPrefix   string
	guildID         string
	mutedChannelID  string
	mutedUsers      []string
	AfterUserUpdate func()
	// todo: consider having one mutex for the whole struct
	muMutedUsers      sync.Mutex
	muAfterUserUpdate sync.Mutex
	muCommandPrefix   sync.Mutex
	muMutedChannelID  sync.Mutex
}

// MuteBotConfig is used with NewMuteBot.
type MuteBotConfig struct {
	// The only words a muted user is allowed to say.
	WordsFile string `json:"wordsFile"`
	// Command prefix this bot listens to.
	CommandPrefix string `json:"commandPrefix"`
	// BotToken provided by Discord.
	BotToken string `json:"botToken"`
	// ServerID this specific bot operates on.
	ServerID string `json:"guildID"`
	// Muted User IDs that can only talk with the words in WordsFile.
	MutedUsers []string `json:"mutedUsers"`
	// Channel where everyone is restricted to words in WordsFile.
	MutedChannelID string `json:"mutedChannelID"`
	// Called after every change to MutedUsers.
	AfterUserUpdate func()
}

// NewMuteBot returns a MuteBot.
func NewMuteBot(config MuteBotConfig) (mb *MuteBot) {
	ws := getWordStore(config.WordsFile)
	mb = &MuteBot{
		session:          nil,
		wordStore:        ws,
		botToken:         config.BotToken,
		commandPrefix:    config.CommandPrefix,
		guildID:          config.ServerID,
		mutedChannelID:   config.MutedChannelID,
		mutedUsers:       config.MutedUsers,
		muMutedUsers:     sync.Mutex{},
		muMutedChannelID: sync.Mutex{},
		AfterUserUpdate: func() {

		},
	}
	return mb
}

// Serve connects the bot to Discord to operate with the configFile passed to the NewMuteBot function.
func (mb *MuteBot) Serve(ctx context.Context) {
	dg, err := discordgo.New("Bot " + mb.botToken)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}
	defer dg.Close()

	dg.AddHandler(func(sess *discordgo.Session, m *discordgo.MessageCreate) {
		mb.HandlerMessageCreate(sess, m)
	})

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	fmt.Println("Serving")
	<-ctx.Done()
}

// MutedUsers returns a string array of muted user IDs.
func (mb *MuteBot) MutedUsers() []string {
	defer mb.muMutedUsers.Unlock()
	mb.muMutedUsers.Lock()
	return mb.mutedUsers
}

// MutedChannelID gets the channel ID where everyone can only use the words in WordsFile.
func (mb *MuteBot) MutedChannelID() string {
	defer mb.muMutedChannelID.Unlock()
	mb.muMutedChannelID.Lock()
	return mb.mutedChannelID
}

// SetMutedUsers sets an array of user IDs to users that can only talk with words in the given WordsFile in MuteBotConfig
func (mb *MuteBot) SetMutedUsers(mutedUsers []string) {
	defer mb.muMutedUsers.Unlock()
	mb.muMutedUsers.Lock()
	mb.mutedUsers = mutedUsers
}

// SetMutedChannelID sets the channel ID where everyone can only use the words in WordsFile.
func (mb *MuteBot) SetMutedChannelID(channelID string) {
	defer mb.muMutedChannelID.Unlock()
	mb.muMutedChannelID.Lock()
	mb.mutedChannelID = channelID
}

// SetAfterUpdateFunc sets the update function ran after the slice of muted users updates.
func (mb *MuteBot) SetAfterUpdateFunc(update func()) {
	defer mb.muAfterUserUpdate.Unlock()
	mb.muAfterUserUpdate.Lock()
	mb.AfterUserUpdate = update
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
func (mb *MuteBot) HandlerMessageCreate(sess *discordgo.Session, msgEv *discordgo.MessageCreate) { //todo: consider moving all mutex to HandlerMessageCreate

	mb.session = sess                                                       // sess is used later in other pointer receiver functions
	if msgEv.Author.ID == mb.session.State.User.ID || msgEv.GuildID == "" { // is bot's own message || is a PM/DM
		return
	}

	defer mb.muCommandPrefix.Unlock()
	mb.muCommandPrefix.Lock()
	if msgHasCommandPrefix(msgEv.Content, mb.commandPrefix) && userCanAddBots(sess, msgEv.Author.ID, msgEv.ChannelID) {
		mb.processCommands(msgEv)
		return
	}
	mb.decideMessageRemoval(msgEv)
}

func userCanAddBots(sess *discordgo.Session, userID, channelID string) bool {
	aperms, err := sess.UserChannelPermissions(userID, channelID)
	if err != nil {
		log.Println("userCanAddBots", err)
	}
	return aperms&discordgo.PermissionManageServer != 0
}

func (mb *MuteBot) decideMessageRemoval(msgEv *discordgo.MessageCreate) {

	defer mb.muMutedUsers.Unlock()
	mb.muMutedUsers.Lock()
	if inSlice(mb.mutedUsers, msgEv.Author.ID) || msgEv.ChannelID == mb.mutedChannelID {
		badWords := badWords(msgEv, mb.wordStore)
		if !(len(badWords) > 0) {
			return
		}
		mb.session.ChannelMessageDelete(msgEv.ChannelID, msgEv.ID)

		notice := "You can only talk with the ten hundred most used words now. https://xkcd.com/simplewriter/\nThese words are not simple: "
		if msgEv.ChannelID == mb.mutedChannelID {
			notice = "You can only talk with the ten hundred most used words in this channel. https://xkcd.com/simplewriter/\nThese words are not simple: "
		}

		for i, badWord := range badWords {
			notice += badWord
			if i != len(badWords)-1 {
				notice += ", "
			}
		}
		notice += "\nPlease try again."
		if userCanAddBots(mb.session, msgEv.Author.ID, msgEv.ChannelID) && !(msgEv.ChannelID == mb.mutedChannelID) {
			notice += "\nPaste `" + mb.commandPrefix + " unmute " + msgEv.Author.ID + "` in the server to unmute yourself. [This line is displayed to users who can add bots only]"
		}

		sendPrivateMessage(mb.session, msgEv.Author.ID, notice)
	}
}

// todo: allow punctuation, strip punctuation when testing
// todo: check if player target is actually a playerID
func (mb *MuteBot) processCommands(msgEv *discordgo.MessageCreate) {
	msg := strings.Split(msgEv.Content, " ")
	if !(len(msg) >= 2) {
		return
	}
	cmd := msg[1]

	if cmd == "help" {
		sendPMHelp(mb.session, msgEv.Author.ID, mb.commandPrefix)
		return
	}

	defer mb.RunAfterUserUpdate() // the rest of the commands will update some value that needs to be written
	if cmd == "set" {
		fmt.Println("msgg", msg)
		mb.mutedChannelID = msgEv.ChannelID
		mb.session.ChannelMessageSend(msgEv.ChannelID, "All people are only allowed to talk with simple words in this area now.")
		return
	}

	if cmd == "rem" {
		mb.mutedChannelID = ""
		mb.session.ChannelMessageSend(msgEv.ChannelID, "Removed simple talk policing in this area.")
		return
	}

	if len(msg) >= 3 {
		thirdArgument := parseUserID(msg[2]) // userID for mute, unmute todo: check if user exists on server

		if cmd == "prefix" {
			mb.commandPrefix = thirdArgument
			sendPrivateMessage(mb.session, msgEv.Author.ID, "This mb will now respond to **"+mb.commandPrefix+"**")
			return
		}

		if cmd == "mute" {
			mb.muteProcedure(thirdArgument, msgEv)
			return
		}

		if cmd == "unmute" {
			mb.unmuteProcedure(thirdArgument, msgEv)
			return
		}
	}
	sendPrivateMessage(mb.session, msgEv.Author.ID, "Invalid command. Try "+mb.commandPrefix+" mute @USER")
	return
}

func (mb *MuteBot) unmuteProcedure(targetUser string, msgEv *discordgo.MessageCreate) {
	if !userExistsInGuild(mb.session, msgEv.GuildID, targetUser) {
		sendPrivateMessage(mb.session, msgEv.Author.ID, "User does not exist in this server.")
		return
	}

	alreadyUnmuted := mb.unmuteUser(mb.session, targetUser)
	if alreadyUnmuted {
		sendPrivateMessage(mb.session, msgEv.Author.ID, "That user is not muted.")
		return
	}
	mb.session.ChannelMessageSend(msgEv.ChannelID, "<@"+targetUser+"> can talk with any words now.")
	return
}

func (mb *MuteBot) RunAfterUserUpdate() {
	mb.muAfterUserUpdate.Lock()
	mb.AfterUserUpdate()
	mb.muAfterUserUpdate.Unlock()
}

func (mb *MuteBot) muteProcedure(targetUser string, msgEv *discordgo.MessageCreate) {
	if !userExistsInGuild(mb.session, msgEv.GuildID, targetUser) {
		sendPrivateMessage(mb.session, msgEv.Author.ID, "User does not exist in this server.")
		return
	}

	alreadyMuted := mb.muteUser(targetUser)
	if alreadyMuted {
		sendPrivateMessage(mb.session, msgEv.Author.ID, "That user is already muted.")
		return
	}
	mb.session.ChannelMessageSend(msgEv.ChannelID, "<@"+targetUser+"> can only talk with the ten hundred most used words now (simple words).")
	return
}

func (mb *MuteBot) muteUser(targetUser string) (alreadyMuted bool) {
	if inSlice(mb.mutedUsers, targetUser) {
		return true
	}
	mb.muMutedUsers.Lock()
	mb.mutedUsers = append(mb.mutedUsers, targetUser)
	mb.muMutedUsers.Unlock()
	return false
}

func (mb *MuteBot) unmuteUser(s *discordgo.Session, targetUser string) (alreadyUnmuted bool) {
	defer mb.muMutedUsers.Unlock()
	mb.muMutedUsers.Lock()
	userIndex := -1
	for i, name := range mb.mutedUsers {
		if targetUser == name {
			userIndex = i
		}
	}
	if userIndex == -1 {
		return true
	}
	mb.mutedUsers = append(mb.mutedUsers[:userIndex], mb.mutedUsers[userIndex+1:]...)
	return false
}

func msgHasCommandPrefix(msg, cmdPrefix string) bool {
	return len(msg) > len(cmdPrefix) && msg[0:len(cmdPrefix)] == cmdPrefix
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
	line0 := fmt.Sprintln("Commands:")
	cmd1 := fmt.Sprintf("**%v set (yourNewPrefix)** - Only allows the 1000 most common words in the channel this is ran in.\n", cmdPrefix)
	cmd2 := fmt.Sprintf("**%v rem (yourNewPrefix)** - Removes the restriction to the currently set channel.\n", cmdPrefix)
	cmd3 := fmt.Sprintf("**%v unmute (@User)** - Unmutes a user\n", cmdPrefix)
	cmd4 := fmt.Sprintf("**%v mute (@User)** - Restricts a user to using only the 1000 most common words.\n", cmdPrefix)
	cmd5 := fmt.Sprintf("**%v prefix (yourNewPrefix)** - Changes the prefix this bot responds to. Currently set to **%v **\n", cmdPrefix, cmdPrefix)

	sendPrivateMessage(session, userID, line0+cmd1+cmd2+cmd3+cmd4+cmd5)
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
