package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dev/agent-runtime/internal/runtime"
)

type Bot struct {
	token    string
	allowID  string
	rt       *runtime.Runtime
	offset   int
}

func NewBot(token, allowID string, rt *runtime.Runtime) *Bot {
	return &Bot{token: token, allowID: allowID, rt: rt}
}

func (b *Bot) Start() {
	if b.token == "" {
		log.Println("Telegram token missing, skipping Bot.")
		return
	}
	log.Println("Starting Telegram bot polling...")
	for {
		b.pollUpdates()
		time.Sleep(2 * time.Second)
	}
}

func (b *Bot) pollUpdates() {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=%d&timeout=10", b.token, b.offset)
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var result struct {
		Ok     bool `json:"ok"`
		Result []struct {
			UpdateID int `json:"update_id"`
			Message  struct {
				Chat struct {
					ID int64 `json:"id"`
				} `json:"chat"`
				Text string `json:"text"`
			} `json:"message"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && result.Ok {
		for _, u := range result.Result {
			b.offset = u.UpdateID + 1

			chatID := strconv.FormatInt(u.Message.Chat.ID, 10)
			if b.allowID != "" && chatID != b.allowID {
				b.sendMessage(chatID, "Unauthorized. You are not in the allowlist.")
				continue
			}

			if u.Message.Text == "/start" {
				b.sendMessage(chatID, "Agentic Runtime Ready! Waiting for commands.")
			} else if u.Message.Text != "" {
				reply, _ := b.rt.ProcessMessage(chatID, u.Message.Text)
				if len(reply) > 4000 {
					reply = reply[:4000] + "\n...[TRUNCATED]"
				}
				b.sendMessage(chatID, reply)
			}
		}
	}
}

func (b *Bot) sendMessage(chatID, text string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.token)
	payload := map[string]string{"chat_id": chatID, "text": text}
	body, _ := json.Marshal(payload)
	http.Post(url, "application/json", bytes.NewBuffer(body))
}
