package main

import (
	"bufio"
	"encoding/json"
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

	"github.com/fatih/color"
	"github.com/gorilla/websocket"
	"github.com/urfave/cli"

	"github.com/gillchristian/webout/types"
)

var host string
var bold = color.New(color.Bold)

func main() {
	app := cli.NewApp()

	app.Name = "webout"
	app.Version = "0.0.1"
	app.Author = "Christian Gill (gillchristiang@gmail.com)"
	app.Usage = "Pipe terminal output to the browser"
	app.UsageText = "$ webout ping google.com"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "host",
			Value:       "webout.xyz", // use "localhost:<port>" locally
			Destination: &host,
			Hidden:      true,
		},
	}
	app.Action = webout

	app.Run(os.Args)
}

func webout(ctx *cli.Context) error {
	err := checkCmd(ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	data, err := getChannel(ctx)
	if err != nil {
		return cli.NewExitError(err, 1)
	}

	fmt.Printf("New channel created: %s\n\n", bold.Sprint(channelURL(data.ID)))

	c, err := connect(ctx, data)
	if err != nil {
		return cli.NewExitError(err, 1)
	}
	defer c.Close()

	done := make(chan struct{})
	out := make(chan []byte)
	argv := ctx.Args()

	go runCmd(done, out, argv[0], argv[1:]...)

	handleMsgs(done, out, interrupt, c)

	return nil
}

func handleMsgs(done chan struct{}, out chan []byte, interrupt chan os.Signal, c *websocket.Conn) {
	for {
		select {
		case <-done:
			return
		case line := <-out:
			err := c.WriteMessage(websocket.TextMessage, line)
			// TODO: don't end process if connection fails
			if err != nil {
				log.Println("Failed to send message:", err)
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

func getChannel(ctx *cli.Context) (types.Channel, error) {
	u := createURL()
	res, err := http.Get(u)
	if err != nil {
		return types.Channel{}, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return types.Channel{}, err
	}

	data := types.Channel{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return types.Channel{}, err
	}
	return data, nil
}

func connect(ctx *cli.Context, channel types.Channel) (*websocket.Conn, error) {
	u := wsURL(channel.ID, channel.Token)

	c, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func checkCmd(ctx *cli.Context) error {
	if ctx.NArg() == 0 {
		return fmt.Errorf("Not enough arguments")
	}

	argv := ctx.Args()

	_, err := exec.LookPath(argv[0])
	if err != nil {
		return fmt.Errorf("Cannot find executable '%s'", argv[0])
	}

	return nil
}

// TODO: allow both running cmd and stdin pipe
func runCmd(done chan<- struct{}, out chan<- []byte, bin string, args ...string) {
	defer func() { done <- struct{}{} }()
	cmd := exec.Command(bin, args...)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	out <- []byte("$ " + bin + " " + strings.Join(args, " ") + "\n")

	err := cmd.Start()
	if err != nil {
		sendErr(out, err)
		return
	}

	wg := sync.WaitGroup{}
	wg.Add(2)
	// TODO: consider using scanner instead of reader
	go pipeByLine(bufio.NewReader(stdout), out, &wg)
	go pipeByLine(bufio.NewReader(stderr), out, &wg)

	err = cmd.Wait()
	wg.Wait() // wait for reader goroutines
	if err != nil {
		sendErr(out, err)
	}
}

func pipeByLine(rd *bufio.Reader, out chan<- []byte, wg *sync.WaitGroup) {
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

func sendErr(out chan<- []byte, err error) {
	fmt.Println(err)
	out <- []byte(err.Error() + "\n")
}

func channelURL(id string) string {
	u := url.URL{Scheme: scheme(), Host: host, Path: "c/" + id}

	return u.String()
}

func wsURL(id, token string) string {
	u := url.URL{Scheme: wsScheme(), Host: host, Path: "c/ws/" + id, RawQuery: "token=" + token}

	return u.String()
}

func createURL() string {
	u := url.URL{Scheme: scheme(), Host: host, Path: "api/create"}
	return u.String()
}

func scheme() string {
	if strings.Contains(host, "local") {
		return "http"
	}
	return "https"
}

func wsScheme() string {
	if strings.Contains(host, "local") {
		return "ws"
	}
	return "wss"
}
