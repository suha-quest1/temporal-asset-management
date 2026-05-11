package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type emailEntry struct {
	Timestamp string                 `json:"timestamp"`
	Payload   map[string]interface{} `json:"payload"`
}

var (
	mu     sync.Mutex
	emails []emailEntry
)

func main() {
	addr := envOrDefault("LISTEN_ADDR", ":8081")

	mux := http.NewServeMux()

	// POST /send — accepts and logs an email payload
	mux.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		entry := emailEntry{
			Timestamp: time.Now().Format(time.RFC3339),
			Payload:   payload,
		}

		mu.Lock()
		emails = append(emails, entry)
		mu.Unlock()

		log.Printf("[MOCK-SES] Email sent: to=%v subject=%v", payload["to"], payload["subject"])
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	})

	// GET /emails — returns all logged emails (email log viewer)
	mux.HandleFunc("/emails", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(emails)
	})

	// GET / — simple health / UI
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count := len(emails)
		mu.Unlock()
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><head><title>Mock SES</title>
		<style>body{font-family:monospace;padding:20px}table{border-collapse:collapse;width:100%}
		td,th{border:1px solid #ddd;padding:8px;text-align:left}</style>
		<script>
		async function load(){
			const r=await fetch('/emails');const data=await r.json();
			const t=document.getElementById('log');t.innerHTML='';
			data.forEach(e=>{
				const row=t.insertRow();
				row.insertCell().textContent=e.timestamp;
				row.insertCell().textContent=e.payload.to||'';
				row.insertCell().textContent=e.payload.subject||'';
				row.insertCell().textContent=e.payload.type||'notification';
			});
			document.getElementById('count').textContent=data.length;
		}
		setInterval(load,2000);window.onload=load;
		</script></head><body>
		<h2>Mock SES - Email Log Viewer</h2>
		<p>Total emails: <span id="count">` + string(rune(count+'0')) + `</span></p>
		<table><thead><tr><th>Time</th><th>To</th><th>Subject</th><th>Type</th></tr></thead>
		<tbody id="log"></tbody></table></body></html>`))
	})

	log.Printf("Mock SES listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
