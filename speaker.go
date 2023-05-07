package ovsklingo

import (
	htgotts "github.com/hegedustibor/htgo-tts"
	"github.com/hegedustibor/htgo-tts/handlers"
	"github.com/hegedustibor/htgo-tts/voices"
	"os"
)

type speaker struct {
	speakers map[Language]htgotts.Speech
}

func NewSpeaker(languages []Language) *speaker {
	speaker := &speaker{
		speakers: make(map[Language]htgotts.Speech),
	}
	for _, language := range languages {
		speaker.addSpeaker(language)
	}
	return speaker
}

func (s *speaker) addSpeaker(language Language) {
	voice := map[Language]string{
		Italian: voices.Italian,
		Spanish: voices.Spanish,
		English: voices.English,
	}[language]
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	speech := htgotts.Speech{Folder: dir + "/audio", Language: voice, Handler: &handlers.Native{}}
	s.speakers[language] = speech
}

func (s *speaker) getSpeaker(language Language) htgotts.Speech {
	return s.speakers[language]
}
