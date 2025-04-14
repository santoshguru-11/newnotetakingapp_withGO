package main

import (
	"crypto/tls"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

// Note represents a single note in the application
type Note struct {
	ID      int    `json:"id"`
	Content string `json:"content"`
}

// Global variables for note management
var (
	notes   = make(map[int]Note)
	notesID = 0
	noteMux sync.RWMutex
)

// saveNotesToFile persists notes to a JSON file
func saveNotesToFile() error {
	noteMux.RLock()
	defer noteMux.RUnlock()

	data, err := json.MarshalIndent(notes, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile("notes.json", data, 0644)
}

// loadNotesFromFile loads notes from the JSON file
func loadNotesFromFile() error {
	data, err := ioutil.ReadFile("notes.json")
	if err != nil {
		if os.IsNotExist(err) {
			notesID = 0
			notes = make(map[int]Note)
			return nil
		}
		return err
	}

	noteMux.Lock()
	defer noteMux.Unlock()

	notes = make(map[int]Note)
	if err := json.Unmarshal(data, &notes); err != nil {
		return err
	}

	notesID = 0
	for id := range notes {
		if id > notesID {
			notesID = id
		}
	}

	return nil
}

// ...rest of existing code...
func noteAppHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Accept") == "application/json" || r.Header.Get("Content-Type") == "application/json" {
		handleNoteJSON(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		noteMux.RLock()
		notesList := make([]Note, 0, len(notes))
		for _, note := range notes {
			notesList = append(notesList, note)
		}
		noteMux.RUnlock()

		tmpl := `<!DOCTYPE html>
<html>
<head>
    <title>Notes App</title>
    <style>
        body { max-width: 800px; margin: 0 auto; padding: 20px; font-family: Arial, sans-serif; }
        .note { border: 1px solid #ddd; margin: 10px 0; padding: 15px; border-radius: 5px; position: relative; }
        textarea { width: 100%; height: 100px; margin: 10px 0; padding: 10px; }
        button { padding: 10px 20px; background: #4CAF50; color: white; border: none; border-radius: 4px; cursor: pointer; }
        button:hover { background: #45a049; }
        .delete-btn {
            position: absolute;
            top: 10px;
            right: 10px;
            padding: 5px 10px;
            background: #ff4444;
            color: white;
            border: none;
            border-radius: 4px;
            cursor: pointer;
        }
        .delete-btn:hover { background: #cc0000; }
    </style>
</head>
<body>
    <h1>📝 Notes App</h1>
    <div id="newNote">
        <textarea id="content" placeholder="Enter your note here..."></textarea>
        <button onclick="addNote()">Add Note</button>
    </div>
    <div id="notes"></div>

    <script>
    function loadNotes() {
        fetch(window.location.href, {
            headers: {'Accept': 'application/json'}
        })
        .then(response => response.json())
        .then(notes => {
            const notesDiv = document.getElementById('notes');
            notesDiv.innerHTML = '';
            if (notes.length === 0) {
                notesDiv.innerHTML = '<p style="text-align: center; color: #666;">No notes yet. Create your first note!</p>';
                return;
            }
            notes.sort((a, b) => b.id - a.id).forEach(note => {
                notesDiv.innerHTML += '<div class="note">' +
                    '<button class="delete-btn" onclick="deleteNote(' + note.id + ')">Delete</button>' +
                    '<h3>Note #' + note.id + '</h3>' +
                    '<p>' + note.content + '</p>' +
                    '</div>';
            });
        })
        .catch(error => console.error('Error loading notes:', error));
    }

    function deleteNote(id) {
        if (!confirm('Are you sure you want to delete this note?')) {
            return;
        }
        fetch(window.location.href + '?id=' + id, {
            method: 'DELETE',
            headers: {'Accept': 'application/json'}
        })
        .then(response => {
            if (!response.ok) throw new Error('Network response was not ok');
            loadNotes();
        })
        .catch(error => {
            console.error('Error deleting note:', error);
            alert('Failed to delete note. Please try again.');
        });
    }

    function addNote() {
        const content = document.getElementById('content').value.trim();
        if (!content) {
            alert('Please enter some content for your note');
            return;
        }
        fetch(window.location.href, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json'
            },
            body: JSON.stringify({content: content})
        })
        .then(response => {
            if (!response.ok) throw new Error('Network response was not ok');
            document.getElementById('content').value = '';
            loadNotes();
        })
        .catch(error => {
            console.error('Error adding note:', error);
            alert('Failed to add note. Please try again.');
        });
    }

    loadNotes();
    </script>
</body>
</html>`
		t := template.Must(template.New("notes").Parse(tmpl))
		t.Execute(w, notesList)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleNoteJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		noteMux.RLock()
		notesList := make([]Note, 0, len(notes))
		for _, note := range notes {
			notesList = append(notesList, note)
		}
		noteMux.RUnlock()
		json.NewEncoder(w).Encode(notesList)

	case http.MethodPost:
		var note Note
		if err := json.NewDecoder(r.Body).Decode(&note); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		noteMux.Lock()
		if len(notes) == 0 {
			notesID = 0
		}
		notesID++
		note.ID = notesID
		notes[note.ID] = note
		noteMux.Unlock()

		if err := saveNotesToFile(); err != nil {
			log.Printf("Error saving notes: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(note)

	case http.MethodDelete:
		noteID := r.URL.Query().Get("id")
		if noteID == "" {
			http.Error(w, "Note ID is required", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(noteID)
		if err != nil {
			http.Error(w, "Invalid note ID", http.StatusBadRequest)
			return
		}

		noteMux.Lock()
		delete(notes, id)
		noteMux.Unlock()

		if err := saveNotesToFile(); err != nil {
			log.Printf("Error saving notes: %v", err)
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func startNoteAppServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", noteAppHandler)

	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	server := &http.Server{
		Addr:      ":8444",
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	certFile := "server.crt"
	keyFile := "server.key"

	log.Printf("Starting HTTPS server on https://localhost:8444")
	if err := server.ListenAndServeTLS(certFile, keyFile); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func main() {
	if err := loadNotesFromFile(); err != nil {
		log.Printf("Error loading notes: %v", err)
	}
	startNoteAppServer()
}
