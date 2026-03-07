package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dev/agent-runtime/internal/runtime"
)

// typingInterval defines how often to resend the "typing" action.
// Telegram's indicator expires after ~5 seconds, so we refresh every 4.
const typingInterval = 4 * time.Second

// progressThrottle is the minimum interval between status message edits
// to avoid hitting Telegram's rate limits.
const progressThrottle = 2 * time.Second

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
	b.discardPendingUpdates()
	for {
		b.pollUpdates()
		time.Sleep(2 * time.Second)
	}
}

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

				// Start typing indicator loop
				start := time.Now()
				typingDone := b.startTypingLoop(chatID)

				// Send initial status message that we will edit with progress
				statusMsgID := b.sendMessageGetID(chatID, "\u2699\ufe0f Thinking...")

				// Build progress callback — edits the status message in real-time
				tracker := newProgressTracker(b, chatID, statusMsgID)
				reply, _ := b.rt.ProcessMessageWithProgress(sid, u.Message.Text, tracker.onProgress)
				close(typingDone)
				elapsed := time.Since(start)

				log.Printf("[telegram] Reply to chat=%s session=%s (took %s, %d chars)", chatID, sid, elapsed.Round(time.Millisecond), len(reply))

				// Delete the status message now that we have the final reply
				b.deleteMessage(chatID, statusMsgID)

				if len(reply) > 4000 {
					reply = reply[:4000] + "\n...[TRUNCATED]"
				}
				b.sendMessage(chatID, reply)
			}
		}
	}
}

// progressTracker manages real-time status updates via Telegram message editing.
type progressTracker struct {
	bot      *Bot
	chatID   string
	msgID    int
	mu       sync.Mutex
	lastEdit time.Time
	steps    []string
	lastText string
}

func newProgressTracker(bot *Bot, chatID string, msgID int) *progressTracker {
	return &progressTracker{
		bot:    bot,
		chatID: chatID,
		msgID:  msgID,
	}
}

func (pt *progressTracker) onProgress(event runtime.ProgressEvent) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	var line string
	switch event.Phase {
	case runtime.PhaseThinking:
		if event.Depth > 0 {
			line = fmt.Sprintf("\u2699\ufe0f Thinking... (turn %d)", event.Depth+1)
		} else {
			line = "\u2699\ufe0f Thinking..."
		}
	case runtime.PhaseToolStart:
		line = fmt.Sprintf("\u25b6\ufe0f %s", event.Message)
	case runtime.PhaseToolEnd:
		// Replace the last tool_start line with a completed version
		if len(pt.steps) > 0 {
			last := pt.steps[len(pt.steps)-1]
			if strings.HasPrefix(last, "\u25b6\ufe0f") {
				if strings.Contains(event.Message, "failed") {
					pt.steps[len(pt.steps)-1] = fmt.Sprintf("\u274c %s", event.Message)
				} else {
					pt.steps[len(pt.steps)-1] = fmt.Sprintf("\u2705 %s", event.Message)
				}
				pt.throttledEdit()
				return
			}
		}
		line = fmt.Sprintf("\u2705 %s", event.Message)
	case runtime.PhaseToken:
		// Streaming tokens — skip in Telegram (too chatty)
		return
	case runtime.PhaseStatus:
		line = fmt.Sprintf("ℹ️ %s", event.Message)
	case runtime.PhaseError:
		line = fmt.Sprintf("\u274c Error: %s", event.Message)
	default:
		return
	}

	pt.steps = append(pt.steps, line)
	pt.throttledEdit()
}

func (pt *progressTracker) throttledEdit() {
	now := time.Now()
	if now.Sub(pt.lastEdit) < progressThrottle {
		return
	}
	pt.lastEdit = now

	// Show last 8 steps to keep it compact
	display := pt.steps
	if len(display) > 8 {
		display = display[len(display)-8:]
	}
	text := strings.Join(display, "\n")
	if text == pt.lastText {
		return
	}
	pt.lastText = text

	go pt.bot.editMessage(pt.chatID, pt.msgID, text)
}

func (b *Bot) getCurrentSession(chatID string) string {
	if sid, ok := b.sessions[chatID]; ok && sid != "" {
		return sid
	}
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

// sendMessageGetID sends a message and returns its message_id for later editing.
func (b *Bot) sendMessageGetID(chatID, text string) int {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", b.token)
	payload := map[string]string{"chat_id": chatID, "text": text}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	var result struct {
		Ok     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	return result.Result.MessageID
}

// editMessage edits an existing Telegram message.
func (b *Bot) editMessage(chatID string, messageID int, text string) {
	if messageID == 0 {
		return
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/editMessageText", b.token)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
	}
	body, _ := json.Marshal(payload)
	http.Post(url, "application/json", bytes.NewBuffer(body))
}

// deleteMessage removes a Telegram message.
func (b *Bot) deleteMessage(chatID string, messageID int) {
	if messageID == 0 {
		return
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/deleteMessage", b.token)
	payload := map[string]interface{}{
		"chat_id":    chatID,
		"message_id": messageID,
	}
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
