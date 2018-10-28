// websockets.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"github.com/gillchristian/netpipe/types"
)

var port = flag.String("port", "8080", "port to run at")

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Channel struct {
	Conns map[string]*websocket.Conn
	Token string
	Lines [][]byte
}

type Channels map[string]*Channel

// TODO: store channel messages
// TODO: close channel when creator disconnects

// GET  /               -> displays help
// POST /               -> creates a channel -> returns `channel-id`
//                      -> the creator is the only one allowed to post messages
// GET  /:channel-id    -> render the `channel-id` page and connects to the channel's socket
// GET  /ws/:channel-id -> websocket for `channel-id`

var funcs = template.FuncMap{"s": func(b []byte) string { return string(b) }}

var ts = struct {
	Channel  *template.Template
	NotFound *template.Template
}{
	Channel: template.Must(template.New("channel.html").
		Funcs(funcs).
		ParseFiles("channel.html")),
	NotFound: template.Must(template.ParseFiles("404.html")),
}

func main() {
	flag.Parse()
	channels := Channels{}

	r := mux.NewRouter()

	r.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()
		channel := newChannel()

		channels[id] = channel

		data, _ := json.Marshal(types.CreatedChannel{ID: id, Token: channel.Token})
		fmt.Fprint(w, string(data))
	})

	r.HandleFunc("/ws/{id}", func(w http.ResponseWriter, r *http.Request) {
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

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			conn.Close()
			return
		}

		channel.Conns[conn.RemoteAddr().String()] = conn

		fmt.Printf("Added `%s` to the pool\n", conn.RemoteAddr())

		token, ok := r.URL.Query()["token"]
		tokenMatches := len(token) > 0 && token[0] == channel.Token
		if !ok || !tokenMatches {
			fmt.Printf("Client '%s' connected. Will only receive messages\n", conn.RemoteAddr())
			return
		}

		for {
			// TODO: handle creator disconnect
			msgType, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Printf("Failed to read message from '%s'. Closing connection\n", conn.RemoteAddr())
				conn.Close()
				delete(channel.Conns, conn.RemoteAddr().String())
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

	r.HandleFunc("/ws.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "ws.js")
	})

	r.HandleFunc("/{id}", func(w http.ResponseWriter, r *http.Request) {
		// TODO: channel.html should inform when creator was disconnected
		id := mux.Vars(r)["id"]
		channel, ok := channels[id]
		if !ok {
			ts.NotFound.Execute(w, struct{ ID string }{id})
			return
		}

		ts.Channel.Execute(w, channel)
	})

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "index.html")
	})

	http.Handle("/", r)

	fmt.Printf("Starting server at http://localhost:%s\n", *port)

	http.ListenAndServe(":"+*port, nil)
}

func newChannel() *Channel {
	channel := Channel{
		Conns: make(map[string]*websocket.Conn),
		Token: uuid.New().String(),
		Lines: [][]byte{},
	}

	return &channel
}