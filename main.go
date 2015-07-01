package main

import (
	"./telegrambot"
	"./xmpp"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

var (
	config        = flag.String("config", "settings.json", "configuration file")
	conf          Configuration
	bot           *telegrambot.Bot
	accountsTable string = "accounts"
	rostersTable  string = "rosters"

	accounts        map[int]*Account
	currentUpdateId = 0
)

const (
	MAX_ACCOUNTS = 500
)

const (
	CMD_CONNECT    = iota
	CMD_CHECK      = iota
	CMD_DISCONNECT = iota
)

type Account struct {
	UserId int
	Jid    string
	Host   string
	Port   uint16

	client *xmpp.Client
}

type Configuration struct {
	Listen     int    `json:"listen"`
	Token      string `json:"token"`
	BaseDomain string `json:"base_domain"`
	HookPath   string `json:"hook_path"`
}

type Command struct {
	Cmd      int
	Jid      string
	Password string
	Host     string
	Port     uint16
}

/* Load configuration from specified file and connect to database */
func loadConfiguration() {
	confData, err := ioutil.ReadFile(*config)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(confData, &conf)
	if err != nil {
		log.Fatal("Configuration decoding error: ", err)
	}

	xmpp.Init()
}

func setupBot() {
	bot = &telegrambot.Bot{Token: conf.Token}
	ok, info := bot.GetMe()
	if !ok {
		log.Fatal("Bot setup error")
	}
	log.Printf("Bot username is: %s", info["username"].(string))
	hookPath := path.Join("https://", conf.BaseDomain, conf.HookPath)
	log.Println("Hook expected on: ", hookPath)
}

func Connect(user_id int, jid string, password string,
	host string, port uint16) error {
	if accounts == nil {
		accounts = make(map[int]*Account)
	}

	/* adding to accounts table */
	if _, ok := accounts[user_id]; ok {
		return errors.New("Account already connected")
	}

	if len(accounts) >= MAX_ACCOUNTS {
		return errors.New("Accounts limit exceeded")
	}

	client := &xmpp.Client{Jid: jid}
	err := client.Connect(password, host, port)
	if err != nil {
		log.Println("Connection error: ", err)
		return errors.New("Connection error")
	}

	account := &Account{
		Jid:    jid,
		Host:   host,
		Port:   port,
		client: client,
	}
	accounts[user_id] = account
	client.Listen()
	go func() {
		defer delete(accounts, user_id)
		for msg := range client.Channel {
			text := fmt.Sprintf("%s %s", msg.From, msg.Text)
			SendMessage(user_id, text)
		}
	}()

	return nil
}

func Disconnect(user_id int) {
	if account, ok := accounts[user_id]; ok {
		account.client.Disconnect()
	}
}

func SendMessage(user_id int, text string) {
	bot.SendMessage(user_id, text)
}

func parseCommand(text string) (*Command, error) {
	var err error
	parts := strings.Split(strings.Trim(text, " "), " ")
	cmd := parts[0]
	if cmd == "/connect" {
		if len(parts) < 3 {
			return nil, errors.New("Need more args")
		}
		jid, pass := parts[1], parts[2]
		host, port := "", 0
		if len(parts) == 4 {
			host = parts[3]
		}
		if len(parts) == 5 {
			port, err = strconv.Atoi(parts[4])
			if err != nil {
				return nil, errors.New("Port format error")
			}
		}
		command := &Command{
			Cmd:      CMD_CONNECT,
			Jid:      jid,
			Password: pass,
			Host:     host,
			Port:     uint16(port),
		}
		return command, nil
	} else if cmd == "/check" || cmd == "/ch" {
		return &Command{Cmd: CMD_CHECK}, nil
	} else if cmd == "/disconnect" || cmd == "/d" {
		return &Command{Cmd: CMD_DISCONNECT}, nil
	}
	return nil, errors.New("Unknown command")
}

func onUpdate(update *telegrambot.Update) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Update handling error: ", err)
		}
	}()

	if currentUpdateId > 0 && update.Id < currentUpdateId {
		return
	}

	currentUpdateId = update.Id
	message := &update.Msg

	// there is no message
	if message.From.Id == 0 {
		return
	}

	// only private chat
	if message.From.Id != message.Chat.Id {
		return
	}

	// skip forwared messages
	if message.ForwardDate > 0 {
		return
	}

	command, err := parseCommand(message.Text)
	if err != nil {
		SendMessage(message.From.Id, err.Error())
		return
	}

	if command.Cmd == CMD_CONNECT {
		err = Connect(message.From.Id, command.Jid, command.Password,
			command.Host, command.Port)
		if err != nil {
			SendMessage(message.From.Id, err.Error())
		}
	} else if command.Cmd == CMD_DISCONNECT {
		Disconnect(message.From.Id)
	}
}

func listen() {
	routes := mux.NewRouter()
	bot.OnUpdate = onUpdate
	routes.HandleFunc(conf.HookPath, bot.Hook)
	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", conf.Listen),
		Handler:        routes,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}

func main() {
	log.Println("Server started")
	loadConfiguration()
	setupBot()
	listen()
}
