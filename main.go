package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"tenhundredmuter/tenhundredbot"
)

type JSONConfig struct {
	WordsFile      string   `json:"wordsFile"`
	CommandPrefix  string   `json:"commandPrefix"`
	BotToken       string   `json:"botToken"`
	ServerID       string   `json:"serverID"`
	MutedChannelID string   `json:"mutedChannelID"`
	MutedUsers     []string `json:"mutedUsers"`
	MaxMutedUsers  int      `json:"maxMutedUsers"`
}

func main() {
	configBytes, err := ioutil.ReadFile("config.json")
	if err != nil {
		if strings.Contains(err.Error(), "The system cannot find the file specified.") {
			createDefaultConfigFile()
			fmt.Println("Created a default config file in this directory. Rerun after populating it.")
		}
	}

	mbconf := tenhundredbot.TenHundredBotConfig{}
	err = json.Unmarshal(configBytes, &mbconf)
	if err != nil {
		log.Fatal(err)
	}
	if mbconf.BotToken == "" {
		log.Fatal("No Bot Token in config.json. See the GitHub README.md for information on how to get a Bot Token.")
	}

	mb := tenhundredbot.NewTenHundredBot(mbconf)
	updateJSONfile := func() {
		updateJSONConfigFile(mbconf, mb.MutedUsers(), mb.MutedChannelID(), mb.ServerID(), mb.CommandPrefix(), mb.MaxMutedUsers())
	}
	mb.SetAfterUpdateFunc(updateJSONfile)
	mb.Serve(context.Background())
}

func createDefaultConfigFile() {
	def := JSONConfig{
		WordsFile:      "wordList.txt", // wordList.txt taken from https://xkcd.com/simplewriter/words.js, removed "fuck" and "shit"
		CommandPrefix:  "!th",
		BotToken:       "",
		ServerID:       "",
		MutedChannelID: "",
		MaxMutedUsers:  30,
		MutedUsers:     []string{},
	}

	jsBytes, err := json.MarshalIndent(def, "", "    ")
	if err != nil {
		log.Fatal("createDefaultConfigFile:", err)
	}

	ioutil.WriteFile("config.json", jsBytes, 0644)
}

func updateJSONConfigFile(mbconf tenhundredbot.TenHundredBotConfig, mutedUsers []string, mutedChannelID, serverID, commandPrefix string, maxMutedUsers int) {
	jsConf := JSONConfig{
		WordsFile:      mbconf.WordsFile,
		CommandPrefix:  commandPrefix,
		BotToken:       mbconf.BotToken,
		ServerID:       serverID,
		MutedChannelID: mutedChannelID,
		MutedUsers:     mutedUsers,
		MaxMutedUsers:  maxMutedUsers,
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
