package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dev/agent-runtime/internal/runtime"
)

// typingInterval defines how often to resend the "typing" action.
// Telegram's indicator expires after ~5 seconds, so we refresh every 4.
const typingInterval = 4 * time.Second
const progressFirstUpdate = 20 * time.Second
const progressUpdateInterval = 30 * time.Second
const maxProgressMessages = 5

type Bot struct {
	token    string
	allowID  string
	rt       *runtime.Runtime
	offset   int
	sessions map[string]string
}

func NewBot(token, allowID string, rt *runtime.Runtime) *Bot {
	return &Bot{token: token, allowID: allowID, rt: rt, sessions: make(map[string]string)}
}

func (b *Bot) Start() {
	if b.token == "" {
		log.Println("Telegram token missing, skipping Bot.")
		return
	}
	log.Println("Starting Telegram bot polling...")
	// On startup, discard any pending updates to avoid re-processing old messages
	// after a service restart. We fetch with offset=-1 to get the last update,
	// then set our offset past it.
	b.discardPendingUpdates()
	for {
		b.pollUpdates()
		time.Sleep(2 * time.Second)
	}
}

// discardPendingUpdates fetches the latest pending update and advances the
// offset past it, so old messages are not re-processed after a restart.
func (b *Bot) discardPendingUpdates() {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates?offset=-1&timeout=0", b.token)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("[telegram] Failed to discard pending updates: %v", err)
		return
	}
	defer resp.Body.Close()

	var result struct {
		Ok     bool `json:"ok"`
		Result []struct {
			UpdateID int `json:"update_id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && result.Ok && len(result.Result) > 0 {
		b.offset = result.Result[len(result.Result)-1].UpdateID + 1
		log.Printf("[telegram] Discarded pending updates, offset set to %d", b.offset)
	} else {
		log.Println("[telegram] No pending updates to discard.")
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

			if u.Message.Text != "" {
				if strings.HasPrefix(u.Message.Text, "/") {
					log.Printf("[telegram] Command from chat=%s: %s", chatID, u.Message.Text)
					b.handleCommand(chatID, u.Message.Text)
					continue
				}

				sid := b.getCurrentSession(chatID)
				log.Printf("[telegram] Message from chat=%s session=%s: %s", chatID, sid, u.Message.Text)

				// Show "typing..." while the agent processes the message
				start := time.Now()
				typingDone := b.startTypingLoop(chatID)
				progressDone := b.startProgressLoop(chatID)
				reply, _ := b.rt.ProcessMessage(sid, u.Message.Text)
				close(typingDone)   // stop typing indicator
				close(progressDone) // stop progress messages
				elapsed := time.Since(start)

				log.Printf("[telegram] Reply to chat=%s session=%s (took %s, %d chars)", chatID, sid, elapsed.Round(time.Millisecond), len(reply))

				if len(reply) > 4000 {
					reply = reply[:4000] + "\n...[TRUNCATED]"
				}
				b.sendMessage(chatID, reply)
			}
		}
	}
}

func (b *Bot) getCurrentSession(chatID string) string {
	if sid, ok := b.sessions[chatID]; ok && sid != "" {
		return sid
	}
	// Legacy-compatible default session id.
	b.sessions[chatID] = chatID
	return chatID
}

func (b *Bot) newSession(chatID string) string {
	sid := b.rt.NewSessionID("tg-" + chatID)
	b.sessions[chatID] = sid
	b.rt.GetSession(sid)
	return sid
}

func (b *Bot) handleCommand(chatID, text string) {
	parts := strings.Fields(strings.TrimSpace(text))
	if len(parts) == 0 {
		return
	}
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/start":
		b.sendMessage(chatID, "Agentic Runtime ready. Use /help to see available commands.")
	case "/help":
		b.sendMessage(chatID, strings.Join([]string{
			"Available commands:",
			"/new - start a new chat session",
			"/history - list your recent sessions",
			"/use <session_id> - switch to a previous session",
			"/session - show current session id",
			"/help - show this message",
		}, "\n"))
	case "/new":
		sid := b.newSession(chatID)
		b.sendMessage(chatID, fmt.Sprintf("Started a new chat session: %s", sid))
	case "/session":
		b.sendMessage(chatID, "Current session: "+b.getCurrentSession(chatID))
	case "/history":
		prefix := "tg-" + chatID + "-"
		sessions, err := b.rt.ListChatSessions(prefix, 10)
		if err != nil {
			b.sendMessage(chatID, "Failed to load history.")
			return
		}
		if len(sessions) == 0 {
			b.sendMessage(chatID, "No previous sessions found. Use /new to start one.")
			return
		}
		current := b.getCurrentSession(chatID)
		var out strings.Builder
		out.WriteString("Recent sessions:\n")
		for _, s := range sessions {
			marker := " "
			if s.SessionID == current {
				marker = "*"
			}
			preview := strings.TrimSpace(s.LastMessage)
			if len(preview) > 48 {
				preview = preview[:48] + "..."
			}
			out.WriteString(fmt.Sprintf("%s %s\n", marker, s.SessionID))
			if preview != "" {
				out.WriteString("  " + preview + "\n")
			}
		}
		out.WriteString("Use /use <session_id> to continue an older chat.")
		b.sendMessage(chatID, out.String())
	case "/use":
		if len(parts) < 2 {
			b.sendMessage(chatID, "Usage: /use <session_id>")
			return
		}
		target := strings.TrimSpace(parts[1])
		if !strings.HasPrefix(target, "tg-"+chatID+"-") && target != chatID {
			b.sendMessage(chatID, "Invalid session id for this chat.")
			return
		}
		b.sessions[chatID] = target
		b.rt.GetSession(target)
		b.sendMessage(chatID, "Switched to session: "+target)
	default:
		b.sendMessage(chatID, "Unknown command. Use /help.")
	}
}

func (b *Bot) sendMessage(chatID, text string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.token)
	payload := map[string]string{"chat_id": chatID, "text": text}
	body, _ := json.Marshal(payload)
	http.Post(url, "application/json", bytes.NewBuffer(body))
}

// sendChatAction sends a chat action (e.g. "typing") to indicate activity.
func (b *Bot) sendChatAction(chatID, action string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendChatAction", b.token)
	payload := map[string]string{"chat_id": chatID, "action": action}
	body, _ := json.Marshal(payload)
	http.Post(url, "application/json", bytes.NewBuffer(body))
}

// startTypingLoop sends the "typing" indicator immediately and then keeps
// refreshing it every typingInterval until the returned channel is closed.
func (b *Bot) startTypingLoop(chatID string) chan struct{} {
	done := make(chan struct{})
	b.sendChatAction(chatID, "typing")
	go func() {
		ticker := time.NewTicker(typingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				b.sendChatAction(chatID, "typing")
			}
		}
	}()
	return done
}

// startProgressLoop sends periodic progress updates for long-running requests.
func (b *Bot) startProgressLoop(chatID string) chan struct{} {
	done := make(chan struct{})
	go func() {
		first := time.NewTimer(progressFirstUpdate)
		defer first.Stop()

		select {
		case <-done:
			return
		case <-first.C:
			b.sendMessage(chatID, "Still working on your request. I will send the result as soon as it is ready.")
		}

		ticker := time.NewTicker(progressUpdateInterval)
		defer ticker.Stop()
		count := 0
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				count++
				if count > maxProgressMessages {
					return
				}
				b.sendMessage(chatID, "Processing is still in progress. Thanks for waiting.")
			}
		}
	}()
	return done
}
