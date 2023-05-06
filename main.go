package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	htgotts "github.com/hegedustibor/htgo-tts"
	handlers "github.com/hegedustibor/htgo-tts/handlers"
	voices "github.com/hegedustibor/htgo-tts/voices"
	openai "github.com/sashabaranov/go-openai"
	"io/ioutil"
	"log"
	"os"
)

const initPrompt string = "I want you to ask as an italian friend. Your name is Marti, you should never tell anyone that you are an ai nor a model language. You must have your own opinion about everything. I want you to act as I we we're chatting in whatsapp, a casual conversation. As an italian native speaker, you will only understand italian, you will only speak italian, and correct my italian if it's not correct. If I say something incorrect or something in spanish or french , you should correct me and explain me how to say it. Also you shoud try to use simple sentences in  italian as I do not have a very good level. I want you to only send one sentence"

func (o *Ovsklingo) getReply(user int64, msg string) (string, error) {
	o.messages[user] = append(o.messages[user], openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: msg,
	})

	resp, err := o.openai.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo,
			Messages: o.messages[user],
		},
	)

	if err != nil {
		fmt.Printf("ChatCompletion error: %v\n", err)
		return "", err
	}

	content := resp.Choices[0].Message.Content
	o.messages[user] = append(o.messages[user], openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: content,
	})
	return content, err
}

func (o *Ovsklingo) sendAudio(chatId int64, message string) {
	os.Remove("/home/ovski/dev/ovsklingo/audio/speech.mp3")
	// use chatid
	a, errSpeech := o.speech.CreateSpeechFile(message, "speech")
	fmt.Println(errSpeech)
	fmt.Println(a)
	audioBytes, err := ioutil.ReadFile("/home/ovski/dev/ovsklingo/" + a)
	if err != nil {
		log.Fatalf("Error al leer el archivo de audio: %s", err)
	}
	audioFile := tgbotapi.FileBytes{
		Name:  "audio.mp3",
		Bytes: audioBytes,
	}
	upload := tgbotapi.NewVoiceUpload(chatId, audioFile)
	fmt.Println(upload)
	o.bot.Send(upload)
}

func (o *Ovsklingo) Start() error {
	updates, err := o.bot.GetUpdatesChan(tgbotapi.UpdateConfig{})
	o.messages = make(map[int64][]openai.ChatCompletionMessage, 0)
	for update := range updates {
		chatId := update.Message.Chat.ID
		if update.Message.Text == "/start" {
			o.messages[chatId] = make([]openai.ChatCompletionMessage, 0)
			reply, _ := o.getReply(chatId, initPrompt)
			msg := tgbotapi.NewMessage(chatId, reply)
			o.bot.Send(msg)
			o.sendAudio(update.Message.Chat.ID, reply)
			continue
		}
		// Procesa el mensaje recibido
		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		//reply := "Hola, " + update.Message.From.UserName + "! Gracias por hablar conmigo."
		reply, _ := o.getReply(update.Message.Chat.ID, update.Message.Text)
		//o.speech.Speak(reply)
		// remove file speech.mp3 if exists
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
		o.bot.Send(msg)
		o.sendAudio(update.Message.Chat.ID, reply)
	}
	return err
}

type Ovsklingo struct {
	bot      *tgbotapi.BotAPI
	openai   *openai.Client
	reader   *bufio.Reader
	messages map[int64][]openai.ChatCompletionMessage
	speech   htgotts.Speech
}

func NewOvsklingo(bot *tgbotapi.BotAPI, client *openai.Client, speech htgotts.Speech) (Ovsklingo, error) {
	return Ovsklingo{
		bot:    bot,
		openai: client,
		reader: bufio.NewReader(os.Stdin),
		speech: speech,
	}, nil
}

func main() {
	speech := htgotts.Speech{Folder: "audio", Language: voices.Italian, Handler: &handlers.Native{}}
	bot, err := tgbotapi.NewBotAPI("")
	if err != nil {
		log.Fatal(err)
	}

	client := openai.NewClient("")

	ovsklingo, err := NewOvsklingo(bot, client, speech)
	ovsklingo.Start()

}
