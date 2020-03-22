package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"tenhundredmuter/muteBot"
)

type JSONConfig struct {
	WordsFile     string   `json:"wordsFile"`
	CommandPrefix string   `json:"commandPrefix"`
	BotToken      string   `json:"botToken"`
	ServerID      string   `json:"serverID"`
	Admins        []string `json:"admins"`
	MutedUsers    []string `json:"mutedUsers"`
}

func main() {
	configBytes, err := ioutil.ReadFile("config.json")
	if err != nil {
		if strings.Contains(err.Error(), "The system cannot find the file specified.") {
			createDefaultConfigFile()
			fmt.Println("Created a default config file in this directory. Rerun after populating it.")
		}
	}

	mbconf := muteBot.MuteBotConfig{}
	err = json.Unmarshal(configBytes, &mbconf)
	if err != nil {
		log.Fatal(err)
	}

	mb := muteBot.NewMuteBot(mbconf, func() {}) // wordList.txt taken from https://xkcd.com/simplewriter/ aka https://xkcd.com/simplewriter/words.js
	updateJSONfile := func() {
		updateJSONConfigFile(mbconf, mb.Admins(), mb.MutedUsers())
	}
	mb.SetAfterUpdateFunc(updateJSONfile)
	mb.Serve(context.Background())
}

func createDefaultConfigFile() {
	def := JSONConfig{
		WordsFile:     "wordList.txt",
		CommandPrefix: "!th",
		BotToken:      "",
		ServerID:      "",
		Admins:        []string{},
		MutedUsers:    []string{},
	}

	jsBytes, err := json.MarshalIndent(def, "", "    ")
	if err != nil {
		log.Fatal("createDefaultConfigFile:", err)
	}

	ioutil.WriteFile("config.json", jsBytes, 0644)
}

func updateJSONConfigFile(mbconf muteBot.MuteBotConfig, admins, mutedUsers []string) {
	jsConf := JSONConfig{
		WordsFile:     mbconf.WordsFile,
		CommandPrefix: mbconf.CommandPrefix,
		BotToken:      mbconf.BotToken,
		ServerID:      mbconf.ServerID,
		Admins:        admins,
		MutedUsers:    mutedUsers,
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
