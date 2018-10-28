package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/gillchristian/netpipe/types"
)

// TODO: create CLI app

var addr = flag.String("addr", "gillchristian.xyz", "http service address")

func main() {
	flag.Parse()
	log.SetFlags(0)

	err := checkCmd()
	check(err)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	data, err := getChannel()
	check(err)

	fmt.Printf("New channel create: %s/netpipe/%s\n", *addr, data.ID)

	c, err := connect(data)
	check(err)
	defer c.Close()

	done := make(chan struct{})
	out := make(chan []byte)

	go runCmd(done, out, os.Args[1], os.Args[2:]...)

	handleMsgs(done, out, interrupt, c)
}

func handleMsgs(done chan struct{}, out chan []byte, interrupt chan os.Signal, c *websocket.Conn) {
	for {
		select {
		case <-done:
			return
		case line := <-out:
			err := c.WriteMessage(websocket.TextMessage, line)
			if err != nil {
				log.Println("Failed to send message:", err)
				// TODO: don't end process if connection fails
				return
			}
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Failed to close connection:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}

	}
}

func getChannel() (types.CreatedChannel, error) {
	u := url.URL{Scheme: "https", Host: *addr, Path: "/netpipe/create"}
	res, err := http.Get(u.String())
	if err != nil {
		return types.CreatedChannel{}, err
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return types.CreatedChannel{}, err
	}
	data := types.CreatedChannel{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return types.CreatedChannel{}, err
	}

	return data, nil
}

func connect(channel types.CreatedChannel) (*websocket.Conn, error) {
	u := url.URL{
		Scheme:   "wss",
		Host:     *addr,
		Path:     "/netpipe/ws/" + channel.ID,
		RawQuery: "token=" + channel.Token,
	}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// TODO: allow both running cmd and reading from stdin
func checkCmd() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("Not enough arguments")
	}

	_, err := exec.LookPath(os.Args[1])
	if err != nil {
		return fmt.Errorf("Cannot find executable '%s'", os.Args[1])
	}

	return nil
}

func runCmd(done chan<- struct{}, out chan<- []byte, bin string, args ...string) {
	defer func() { done <- struct{}{} }()
	cmd := exec.Command(bin, args...)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	stdoutByLine := bufio.NewReader(stdout)
	stderrByLine := bufio.NewReader(stderr)

	out <- []byte("$ " + bin + " " + strings.Join(args, " ") + "\n")

	err := cmd.Start()
	if err != nil {
		return
	}

	wg := sync.WaitGroup{}

	wg.Add(2)
	go byLines(stdoutByLine, out, &wg)
	go byLines(stderrByLine, out, &wg)

	cmd.Wait()
	wg.Wait()
}

func byLines(rd *bufio.Reader, out chan<- []byte, wg *sync.WaitGroup) {
	for {
		l, err := rd.ReadBytes('\n')
		if err != nil {
			wg.Done()
			return
		}
		fmt.Print(string(l))
		out <- l
	}
}
