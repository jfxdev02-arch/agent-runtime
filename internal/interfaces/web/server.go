package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dev/agent-runtime/internal/runtime"
)

type Server struct {
	rt   *runtime.Runtime
	port string
}

func NewServer(rt *runtime.Runtime, port string) *Server {
	return &Server{rt: rt, port: port}
}

func (s *Server) Start() error {
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/api/chat", s.handleChat)
	fmt.Printf("Web server listening on port %s\n", s.port)
	return http.ListenAndServe(":"+s.port, nil)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html>
<head><title>Agentic Runtime</title></head>
<body>
	<h1>Agentic Runtime (RPi 3)</h1>
	<div id="chat" style="height:400px;border:1px solid #ccc;overflow-y:scroll;margin-bottom:10px;padding:10px;"></div>
	<input type="text" id="msg" style="width:80%" />
	<button onclick="send()">Send</button>

	<script>
		const chat = document.getElementById('chat');
		function appendMsg(role, text) {
			chat.innerHTML += "<b>" + role + ":</b> <pre>" + text + "</pre><br/>";
			chat.scrollTop = chat.scrollHeight;
		}
		async function send() {
			const input = document.getElementById('msg');
			const text = input.value;
			if (!text) return;
			appendMsg('User', text);
			input.value = '';
			const res = await fetch('/api/chat', {
				method: 'POST',
				headers: {'Content-Type': 'application/json'},
				body: JSON.stringify({session_id: 'web-default', message: text})
			});
			const data = await res.json();
			appendMsg('Agent', data.reply);
		}
	</script>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SessionID string `json:"session_id"`
		Message   string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reply, _ := s.rt.ProcessMessage(req.SessionID, req.Message)
	json.NewEncoder(w).Encode(map[string]string{"reply": reply})
}
