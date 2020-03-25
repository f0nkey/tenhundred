package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"tenhundredmuter/mutebot"
)

type JSONConfig struct {
	WordsFile      string   `json:"wordsFile"`
	CommandPrefix  string   `json:"commandPrefix"`
	BotToken       string   `json:"botToken"`
	ServerID       string   `json:"serverID"`
	MutedChannelID string   `json:"mutedChannelID"`
	MutedUsers     []string `json:"mutedUsers"`
}

func main() {
	configBytes, err := ioutil.ReadFile("config.json")
	if err != nil {
		if strings.Contains(err.Error(), "The system cannot find the file specified.") {
			createDefaultConfigFile()
			fmt.Println("Created a default config file in this directory. Rerun after populating it.")
		}
	}

	mbconf := mutebot.MuteBotConfig{}
	err = json.Unmarshal(configBytes, &mbconf)
	if err != nil {
		log.Fatal(err)
	}

	mb := mutebot.NewMuteBot(mbconf)
	updateJSONfile := func() {
		updateJSONConfigFile(mbconf, mb.MutedUsers(), mb.MutedChannelID(), mb.ServerID())
	}
	mb.SetAfterUpdateFunc(updateJSONfile)
	mb.Serve(context.Background())
}

func createDefaultConfigFile() {
	def := JSONConfig{
		WordsFile:      "wordList.txt", // wordList.txt taken from https://xkcd.com/simplewriter/words.js, removed "fuck" and "shit"
		CommandPrefix:  "!th",
		BotToken:       "",
		ServerID:       "", //todo: fix putting in serverID in the config file
		MutedChannelID: "",
		MutedUsers:     []string{},
	}

	jsBytes, err := json.MarshalIndent(def, "", "    ")
	if err != nil {
		log.Fatal("createDefaultConfigFile:", err)
	}

	ioutil.WriteFile("config.json", jsBytes, 0644)
}

func updateJSONConfigFile(mbconf mutebot.MuteBotConfig, mutedUsers []string, mutedChannelID, serverID string) {
	jsConf := JSONConfig{
		WordsFile:      mbconf.WordsFile,
		CommandPrefix:  mbconf.CommandPrefix,
		BotToken:       mbconf.BotToken,
		ServerID:       serverID,
		MutedChannelID: mutedChannelID,
		MutedUsers:     mutedUsers,
	}

	jsonBytes, err := json.MarshalIndent(jsConf, "", "    ")
	if err != nil {
		log.Fatal("updateJSONfile", err)
	}
	err = ioutil.WriteFile("config.json", jsonBytes, 0644)
	if err != nil {
		log.Fatal("updateJSONfile", err)
	}
}
