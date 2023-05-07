package ovsklingo

import (
	"bufio"
	"context"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	openai "github.com/sashabaranov/go-openai"
	"golang.org/x/exp/slices"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

// const initPrompt string = "I want you to ask as an LANG friend. Your name is NAME, you should never tell anyone that you are an ai nor a model language. You must have your own opinion about everything. I want to have a casual conversation with you. As an LANG native speaker, you will only understand LANG, you will only speak LANG, and correct my LANG if it's not correct. If I say something incorrect or something in other language, you should correct me and explain me how to say it. Also you shoud try to use simple sentences in LANG as I do not have a very good level. I want you to only send maximum 3 sentences. You should only act as NAME, you shouldn't make the whole conversation. My name is "
const initPrompt = "Voglio che tu chieda come amico LANG. Il tuo nome è NAME, non dovresti mai dire a nessuno che sei un AI né un linguaggio modello. Devi avere la tua opinione su tutto. Voglio avere una conversazione casuale con te. Come madrelingua LANG, capirai solo LANG, parlerai solo LANG e correggerai il mio LANG se non è corretto. Se dico qualcosa di sbagliato o qualcosa in un'altra lingua, dovresti correggermi e spiegarmi come dirlo. Inoltre dovresti provare a usare frasi semplici in LANG perché non ho un livello molto buono. Voglio che tu invii solo massimo 3 frasi. Dovresti agire solo come NOME, non dovresti fare l'intera conversazione. Mi chiamo "

//const initPrompt string = "I want you to act as a spoken LANG teacher and improver. I will speak to you in LANG and you will reply to me in LANG to practice my spoken LANG. I want you to keep your reply neat, limiting the reply to 100 words. I want you to strictly correct my grammar mistakes, typos, and factual errors. I want you to ask me a question in your reply. Now let's start practicing, you could ask me a question first. Remember, I want you to strictly correct my grammar mistakes, typos, and factual errors. Please don't correct everything before this message: Your name is NAME, and mine is "

var names = map[string]string{
	"Italian": "Mario",
	"Spanish": "Julio",
	"English": "John",
}

func (o *Ovsklingo) getReply(user int64, msg string) (string, error) {
	o.messages[user] = append(o.messages[user], openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: msg,
	})
	o.activeChats[user] = true
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
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println(err)
	}
	fName := fmt.Sprintf("%d-%s", chatId, "speech")
	fullPath := fmt.Sprintf("%s/audio/%s.mp3", dir, fName)
	speech := o.speaker.getSpeaker(o.lang[chatId])
	spF, err := speech.CreateSpeechFile(message, fName)
	fmt.Println(spF)
	if err != nil {
		log.Fatalf("Error while creating speech file: %s", err)
	}
	audioBytes, err := os.ReadFile(fullPath)
	if err != nil {
		log.Fatalf("Error while reading audio file: %s", err)
	}
	audioFile := tgbotapi.FileBytes{
		Name:  "audio.mp3",
		Bytes: audioBytes,
	}
	upload := tgbotapi.NewVoiceUpload(chatId, audioFile)
	os.Remove(fullPath)
	o.bot.Send(upload)
}

func (o *Ovsklingo) getTextFromVoice(ctx context.Context, msg *tgbotapi.Message) (string, error) {
	voice := msg.Voice
	getFileName := func(messageId int, format string) string {
		return fmt.Sprintf("/tmp/%d.%s", messageId, format)
	}
	url, _ := o.bot.GetFileDirectURL(voice.FileID)
	fmt.Println(url)
	response, err := http.Get(url)
	if err != nil {
		log.Panic(err)
	}
	defer response.Body.Close()
	mp3Name := getFileName(msg.MessageID, "mp3")
	oggName := getFileName(msg.MessageID, "ogg")
	localFile, err := os.Create(oggName)
	if err != nil {
		log.Panic(err)
	}
	defer localFile.Close()

	_, err = io.Copy(localFile, response.Body)
	if err != nil {
		log.Panic(err)
	}
	localFile.Close()

	cmd := exec.Command("ffmpeg", "-i", oggName, "-acodec", "libmp3lame", mp3Name)

	// Run the command and wait for it to finish
	err = cmd.Run()
	if err != nil {
		panic(err)
	}
	// convert oga file localFile to mp3 file
	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		Language: "it",
		FilePath: mp3Name,
	}
	resp, err := o.openai.CreateTranscription(ctx, req)
	if err != nil {
		fmt.Printf("Transcription error: %v\n", err)
		return "", err
	}
	os.Remove(mp3Name)
	os.Remove(oggName)
	fmt.Println(resp.Text)
	return resp.Text, nil
}

func (o *Ovsklingo) Start() error {
	updates, err := o.bot.GetUpdatesChan(tgbotapi.UpdateConfig{})
	for update := range updates {
		chatId := update.Message.Chat.ID
		msgTxt := update.Message.Text
		if msgTxt == "/start" {
			o.activeChats[chatId] = false
			reply := tgbotapi.NewMessage(chatId, "What language do you want to learn?")
			reply.ReplyMarkup = tgbotapi.NewReplyKeyboard(
				tgbotapi.NewKeyboardButtonRow(
					tgbotapi.NewKeyboardButton(string(Italian)),
					tgbotapi.NewKeyboardButton(string(Spanish)),
					tgbotapi.NewKeyboardButton(string(English)),
				),
			)
			o.bot.Send(reply)
			continue
		}
		if slices.Contains([]Language{Italian, Spanish, English}, Language(msgTxt)) && o.activeChats[chatId] == false {
			o.messages[chatId] = make([]openai.ChatCompletionMessage, 0)
			o.activeChats[chatId] = true
			o.lang[chatId] = Language(msgTxt)
			prompt := strings.ReplaceAll(initPrompt, "LANG", msgTxt)
			prompt = strings.ReplaceAll(prompt, "NAME", names[msgTxt])
			reply, _ := o.getReply(chatId, prompt+update.Message.Chat.FirstName)
			msg := tgbotapi.NewMessage(chatId, reply)
			o.bot.Send(msg)
			o.sendAudio(update.Message.Chat.ID, reply)
			continue
		}
		if update.Message.Voice != nil {
			audioMsg, err := o.getTextFromVoice(context.Background(), update.Message)
			if err != nil {
				log.Printf("Error while converting voice to text: %s", err)
				continue
			}
			msgTxt = audioMsg
		}
		log.Printf("[%s] %s", update.Message.From.UserName, msgTxt)
		reply, _ := o.getReply(update.Message.Chat.ID, msgTxt)
		msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
		o.bot.Send(msg)
		o.sendAudio(update.Message.Chat.ID, reply)
	}
	return err
}

type Ovsklingo struct {
	bot         *tgbotapi.BotAPI
	openai      *openai.Client
	reader      *bufio.Reader
	messages    map[int64][]openai.ChatCompletionMessage
	activeChats map[int64]bool
	speaker     *speaker
	lang        map[int64]Language
}

func NewOvsklingo(bot *tgbotapi.BotAPI, client *openai.Client, languages []Language) (Ovsklingo, error) {
	speaker := NewSpeaker(languages)
	return Ovsklingo{
		bot:         bot,
		openai:      client,
		reader:      bufio.NewReader(os.Stdin),
		speaker:     speaker,
		messages:    make(map[int64][]openai.ChatCompletionMessage, 0),
		activeChats: make(map[int64]bool, 0),
		lang:        make(map[int64]Language, 0),
	}, nil
}
