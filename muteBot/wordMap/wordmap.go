package wordMap

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"strings"
)

type WordMap struct {
	wordMap map[string]struct{}
}

func (ws WordMap) Exists(s string) bool {
	if _, exists := ws.wordMap[strings.ToLower(s)]; exists {
		return true
	}
	return false
}

func NewWordMap(words io.Reader) (*WordMap, error) {
	wordsBytes := make([]byte, 0, 5887) // size of original 1000.txt
	wordsBytes, err := ioutil.ReadAll(words)
	if err != nil {
		log.Println("NewWordMap:", err)
		return nil, fmt.Errorf("NewWordMap: %v", err)
	}

	var wordMap = make(map[string]struct{})
	var word strings.Builder
	for _, letter := range wordsBytes {
		if letter == '\n' {
			wordMap[strings.ToLower(word.String())] = struct{}{}
			word.Reset()
			continue
		}
		word.WriteByte(letter)
	}
	return &WordMap{wordMap}, nil

}
