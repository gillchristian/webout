// websockets.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/gillchristian/webout/types"
)

var port = flag.String("port", "8080", "port to run at")

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// TODO: rename
type MemChannel struct {
	Conns map[string]*websocket.Conn
	Close bool
	Lines [][]byte
	types.Channel
}
type Channels map[string]*MemChannel

// TODO: persist channels - files ? redis ?

// GET  /                 -> home
// POST /api/create       -> creates a channel -> returns `channel-id`
//                        -> the creator is the only one allowed to post messages
// GET  /c/:channel-id    -> render the `channel-id` page and connects to the channel's socket
// GET  /c/ws/:channel-id -> websocket for `channel-id`

var funcs = template.FuncMap{"s": func(b []byte) string { return string(b) }}

var ts = struct {
	Channel  *template.Template
	NotFound *template.Template
}{
	Channel: template.Must(template.New("channel.html").
		Funcs(funcs).
		ParseFiles("templates/channel.html")),
	NotFound: template.Must(template.ParseFiles("templates/404.html")),
}

func main() {
	flag.Parse()
	channels := Channels{}

	r := mux.NewRouter()

	db := connect()
	defer db.Close()

	r.HandleFunc("/api/create", func(w http.ResponseWriter, r *http.Request) {
		newChan := types.Channel{
			CreatedAt: time.Now(),
		}

		err := db.Insert(&newChan)
		if err != nil {
			// TODO return error
			fmt.Println(err)
		}

		channel := &MemChannel{
			Conns:   make(map[string]*websocket.Conn),
			Lines:   [][]byte{},
			Channel: newChan,
		}

		channels[newChan.ID] = channel

		// TODO: handle error
		data, _ := json.Marshal(newChan)

		fmt.Fprint(w, string(data))
		fmt.Println(r.Method, 200, r.URL.Path)
	})

	r.HandleFunc("/c/ws/{id}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		id, ok := vars["id"]
		if !ok {
			fmt.Fprint(w, "missing id")
			return
		}
		channel, ok := channels[id]
		if !ok {
			fmt.Fprint(w, "channel doesn't exist")
			return
		}

		if channel.Close {
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			conn.Close()
			return
		}

		channel.Conns[conn.RemoteAddr().String()] = conn

		// TODO: send error when token doesn't match
		token, ok := r.URL.Query()["token"]
		tokenMatches := len(token) > 0 && token[0] == channel.Token
		if !ok || !tokenMatches {
			return
		}

		for {
			// TODO: handle creator disconnect
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("Failed to read message from '%s'. Closing connection\n", conn.RemoteAddr())
				conn.Close()
				channel.Close = true
				fmt.Printf("Channel '%s' closed. Closing all listeners\n", id)
				for _, c := range channel.Conns {
					c.Close()
				}
				channel.Conns = nil
				return
			}
			channel.Lines = append(channel.Lines, msg)

			// TODO: don't block reading msgs. create goroutine and chan
			for _, c := range channel.Conns {
				if err = c.WriteMessage(msgType, msg); err != nil {
					fmt.Printf("Disconnecting '%s', failed to send message\n", c.RemoteAddr())
					c.Close()
					delete(channel.Conns, c.RemoteAddr().String())
				}
			}
		}
	})

	r.HandleFunc("/c/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		channel, ok := channels[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			if err := ts.NotFound.Execute(w, struct{ ID string }{id}); err != nil {
				w.WriteHeader(http.StatusNotFound) // Calling twice uses the last one?
				http.ServeFile(w, r, "public/500.html")
				fmt.Println(r.Method, 500, r.URL.Path, err)
			} else {
				fmt.Println(r.Method, 404, r.URL.Path)
			}
			return
		}

		// TODO: add func to execute template and handle errors
		if err := ts.Channel.Execute(w, channel); err != nil {
			w.WriteHeader(http.StatusNotFound)
			http.ServeFile(w, r, "public/500.html")
			fmt.Println(r.Method, 500, r.URL.Path, err)
		} else {
			fmt.Println(r.Method, 200, r.URL.Path)
		}
	})

	// TODO: ServeFile should work for serving the whole directory
	r.HandleFunc("/ws.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/ws.js")
		fmt.Println(r.Method, 200, r.URL.Path)
	})

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/index.html")
		fmt.Println(r.Method, 200, r.URL.Path)
	})

	http.Handle("/", r)

	fmt.Printf("Starting server at http://localhost:%s\n", *port)

	err := http.ListenAndServe(":"+*port, nil)
	fmt.Println(err) // http.ListenAndServe error is never nil
}
