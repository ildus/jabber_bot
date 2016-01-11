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

type UserAccounts map[string]*Account

var (
	config        = flag.String("config", "settings.json", "configuration file")
	conf          Configuration
	bot           *telegrambot.Bot
	accountsTable string = "accounts"
	rostersTable  string = "rosters"

	accounts        map[int]UserAccounts
	currentUpdateId = 0
)

const (
	MAX_ACCOUNTS = 500
)

const (
	CMD_CONNECT     = iota
	CMD_CHECK       = iota
	CMD_DISCONNECT  = iota
	CMD_BOT_MESSAGE = iota
	CMD_MESSAGE     = iota
)

/* Account represents connection on this level,
	UserId - id of telegram user
	Jid, Host, Port - connection params

  For security reasons we do not keep password here
    messageJids - used for messages reply, when we get some message we keep
	its id and sender, and when user wants to reply, we can determine receiver
	of reply 

  client - xmpp connection
*/
type Account struct {
	UserId int
	Jid    string
	Host   string
	Port   uint16

	messageJids map[int]string
	client      *xmpp.Client
}

/* Configuration, filled from settings file */
type Configuration struct {
	Listen      int    `json:"listen"`
	Token       string `json:"token"`
	BaseDomain  string `json:"base_domain"`
	HookPath    string `json:"hook_path"`
	AdminUserId int    `json:"admin_user_id"`
}

/* Messages to bot parsed to this struct if they has some meaning */
type Command struct {
	Cmd      int
	Jid      string
	Password string
	Host     string
	Port     uint16

	UserId  int
	Message string
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

/* Creates bot, checks it and shows in console hook path */
func setupBot() {
	bot = &telegrambot.Bot{Token: conf.Token}
	ok, info := bot.GetMe()
	if !ok {
		log.Fatal("Bot setup error")
	}
	log.Printf("Bot username is: %s", info["username"].(string))
	hookPath := "https://" + path.Join(conf.BaseDomain, conf.HookPath)
	log.Println("Hook expected on: ", hookPath)

	go func() {
		time.Sleep(2 * time.Second)
		ok := bot.SetWebhook(hookPath)
		if ok {
			log.Println("Bot setWebhook: ok")
		} else {
			log.Fatal("Bot setWebhook: error")
		}
	}()
}

/* Creates xmpp connection
   User can have more than one connection, it all is keeped in `accounts`
   First key is user_id, second - jid

   It creates client, start client listening, and start own goroutine 
   that listens messages from client channel
*/

func Connect(user_id int, jid string, password string,
	host string, port uint16) error {
	if accounts == nil {
		accounts = make(map[int]UserAccounts)
	}

	/* adding to accounts table */
	if _, ok := accounts[user_id]; !ok {
		accounts[user_id] = make(UserAccounts)
	}

	if _, ok := accounts[user_id][jid]; ok {
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

	user_accounts := accounts[user_id]
	account := &Account{
		Jid:    jid,
		Host:   host,
		Port:   port,
		client: client,
	}
	user_accounts[jid] = account
	client.Listen()

	// start listening messages from jabber
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Println("Messages listening error: ", err)
			}
		}()
		defer delete(user_accounts, jid)

		messageJids := make(map[int]string)
		user_accounts[jid].messageJids = messageJids
		for msg := range client.Channel {
			msg_jid := strings.Split(msg.From, "/")[0]
			text := fmt.Sprintf("%s %s", msg_jid, msg.Text)
			message_id := SendMessage(user_id, text)
			if message_id > 0 {
				messageJids[message_id] = msg_jid
			}
		}
	}()

	return nil
}

func Disconnect(user_id int) {
	if user_accounts, ok := accounts[user_id]; ok {
		for jid, account := range user_accounts {
			log.Println("%s disconnected", jid)
			account.client.Disconnect()
		}
	}
}

func SendMessage(user_id int, text string) int {
	return bot.SendMessage(user_id, text)
}

func parseCommand(message *telegrambot.Message) (*Command, error) {
	defer func() {
		if err := recover(); err != nil {
			log.Println("Command parse error", message.Text)
		}
	}()

	text := message.Text

	if len(text) > 5000 {
		return nil, errors.New("Too long command")
	}

	var err error
	parts := strings.Split(strings.Trim(text, " "), " ")
	cmd := parts[0]
	if cmd == "/connect" {
		if len(parts) < 3 {
			return nil, errors.New("Need more args")
		}
		jid, pass := parts[1], parts[2]
		if !EmailIsValid(jid) {
			return nil, errors.New("Enter valid jid")
		}
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
	} else if cmd == "/bot_message" && conf.AdminUserId > 0 {
		user_id, err := strconv.Atoi(parts[1])
		if err == nil {
			msg := strings.Join(parts[2:], " ")
			return &Command{
				Cmd:     CMD_BOT_MESSAGE,
				Message: msg,
				UserId:  user_id,
			}, nil
		}
	} else if cmd == "/message" {
		receiver := parts[1]
		if !EmailIsValid(receiver) {
			return nil, errors.New("Enter valid recipient")
		}
		msg := strings.Join(parts[2:], "")
		return &Command{
			Cmd:     CMD_MESSAGE,
			Message: msg,
			Jid:     receiver,
		}, nil
	}

	if conf.AdminUserId > 0 {
		text := fmt.Sprintf("%d %s %s %s",
			message.From.Id,
			message.From.FirstName,
			message.From.Username,
			message.Text)
		SendMessage(conf.AdminUserId, text)
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
	user_id := message.From.Id

	// there is no message
	if user_id == 0 {
		return
	}

	// only private chat
	if user_id != message.Chat.Id {
		return
	}

	// skip forwared messages
	if message.ForwardDate > 0 {
		return
	}

	reply_to_id := message.ReplyTo.MessageId
	if reply_to_id > 0 {
		// is reply
		user_accounts, ok := accounts[user_id]
		if !ok {
			SendMessage(user_id, "You are not connected")
		}
		for _, account := range user_accounts {
			if jid, ok := account.messageJids[reply_to_id]; ok {
				account.client.SendMessage(jid, message.Text)
				break
			}
		}

		return
	}

	command, err := parseCommand(message)
	if err != nil {
		SendMessage(message.From.Id, err.Error())
		return
	}

	switch command.Cmd {
	case CMD_CONNECT:
		{
			err = Connect(message.From.Id, command.Jid, command.Password,
				command.Host, command.Port)
			if err != nil {
				SendMessage(message.From.Id, err.Error())
			}
		}
	case CMD_DISCONNECT:
		Disconnect(message.From.Id)
	case CMD_BOT_MESSAGE:
		{
			if conf.AdminUserId == user_id {
				SendMessage(command.UserId, command.Message)
			}
		}
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
