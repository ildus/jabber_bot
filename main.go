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
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

var (
	config = flag.String("config", "settings.json", "configuration file")
	conf   Configuration
	bot    *telegrambot.Bot

	users           = make(map[int]*User)
	currentUpdateId = 0
)

const (
	MAX_ACCOUNTS = 2
	GREETING     = `
Hi, this is a jabber client for Telegram.
Start the work with ` + "`" + "/connect" + "`" + `command.
For now it can only get messages and supports replying to them.`
)

const (
	CMD_UNDEFINED   = iota
	CMD_CONNECT     = iota
	CMD_CHECK       = iota
	CMD_DISCONNECT  = iota
	CMD_BOT_MESSAGE = iota
	CMD_MESSAGE     = iota
	CMD_START       = iota
)

const (
	STATUS_CONNECTING   = iota
	STATUS_CONNECTED    = iota
	STATUS_DISCONNECTED = iota
)

/* Contains user details */
type User struct {
	accounts map[string]*Account
	command  *Command
}

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
	status      int
}

/* Configuration, filled from settings file */
type Configuration struct {
	Listen      int    `json:"listen"`
	Token       string `json:"token"`
	BaseDomain  string `json:"base_domain"`
	HookPath    string `json:"hook_path"`
	AdminUserId int    `json:"admin_user_id"`
	Debug       bool   `json:"debug"`
}

/* Messages to bot parsed to this struct if they has some meaning */
type Command struct {
	Cmd  int
	Jid  string
	Host string
	Port uint16

	From    int
	UserId  int
	Message string

	/* Used in multi-step commands */
	Step  int
	MsgId int
}

func (c *Command) Next(msg string) {
	c.MsgId = bot.SendReplyMessage(c.From, msg)
	c.Step += 1
}

func (c *Command) Again(msg string) {
	c.MsgId = bot.SendReplyMessage(c.From, msg)
}

/* Load configuration from specified file and connect to database */
func loadConfiguration() {
	flag.Parse()
	confData, err := ioutil.ReadFile(*config)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(confData, &conf)
	if err != nil {
		log.Fatal("Configuration decoding error: ", err)
	}

	validToken := regexp.MustCompile(`^\d+:[\w\-]+$`)
	if !validToken.MatchString(conf.Token) {
		log.Fatal("Invalid token format")
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

	var account *Account
	var found bool

	user_accounts := users[user_id].accounts
	if account, found = user_accounts[jid]; !found {
		if len(user_accounts) >= MAX_ACCOUNTS {
			return errors.New("Accounts limit exceeded")
		}

		account = &Account{
			Jid:    jid,
			client: &xmpp.Client{Jid: jid},
			status: STATUS_DISCONNECTED,
		}
		user_accounts[jid] = account
	}

	if account.status == STATUS_CONNECTED {
		return errors.New("Account already connected")
	} else if account.status == STATUS_CONNECTING {
		return errors.New("Account is already trying to connect")
	}

	account.Host = host
	account.Port = port
	account.status = STATUS_CONNECTING
	account.messageJids = make(map[int]string)

	client := account.client
	err := client.Connect(password, host, port)
	if err != nil {
		log.Println("Connection error: ", err)
		return errors.New("Connection error")
	}

	client.Listen()

	// start listening messages from jabber
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Println("Messages listening error: ", err)

				if conf.Debug {
					debug.PrintStack()
				}
			}

			account.status = STATUS_DISCONNECTED
			SendMessage(user_id, fmt.Sprintf("%s disconnected (or connection failed)", jid))
		}()

		for event := range client.Channel {
			switch event.EventType {
			case xmpp.XMPP_CONN_CONNECT:
				{
					SendMessage(user_id, fmt.Sprintf("%s connected", jid))
					account.status = STATUS_CONNECTED
				}
			case xmpp.XMPP_CONN_DISCONNECT:
				{
					SendMessage(user_id, fmt.Sprintf("%s disconnected", jid))
				}
			case xmpp.XMPP_CONN_FAIL:
				{
					SendMessage(user_id, fmt.Sprintf("%s connection fail", jid))
				}
			case xmpp.XMPP_MESSAGE:
				{
					msg := event.Msg
					msg_jid := strings.Split(msg.From, "/")[0]
					text := fmt.Sprintf("%s %s", msg_jid, msg.Text)
					message_id := SendMessage(user_id, text)
					if message_id > 0 {
						account.messageJids[message_id] = msg_jid
					}
				}
			}
		}
	}()

	return nil
}

func Disconnect(user_id int) {
	if user, ok := users[user_id]; ok {
		for _, account := range user.accounts {
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
	command := &Command{
		From: message.From.Id,
		Step: 0,
		Cmd:  CMD_UNDEFINED,
	}

	if command.From == 0 {
		log.Println("Something went wrong (command.From == 0)")
		return nil, errors.New("Internal error")
	}

	if len(text) > 5000 {
		return nil, errors.New("Too long command")
	}

	parts := strings.Split(strings.Trim(text, " "), " ")
	cmd := parts[0]
	if cmd == "/connect" {
		command.Cmd = CMD_CONNECT
	} else if cmd == "/check" && conf.AdminUserId == command.From {
		command.Cmd = CMD_CHECK
	} else if cmd == "/disconnect" {
		command.Cmd = CMD_DISCONNECT
	} else if cmd == "/start" {
		command.Cmd = CMD_START
	} else if cmd == "/bot_message" && conf.AdminUserId == command.From {
		user_id, err := strconv.Atoi(parts[1])
		if err == nil {
			command.Cmd = CMD_BOT_MESSAGE
			command.Message = strings.Join(parts[2:], " ")
			command.UserId = user_id
		}
	} else if cmd == "/message" {
		// todo: add support message to anybody
		command.Cmd = CMD_MESSAGE
	}

	/*
	 * If command can't be parsed, try to send it to admin.
	 * It can be just an error
	 */
	if command.Cmd == CMD_UNDEFINED {
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

	return command, nil
}

func onUpdate(update *telegrambot.Update) {
	var command *Command
	var user *User
	var err error

	defer func() {
		if err := recover(); err != nil {
			log.Println("Update handling error: ", err)

			if conf.Debug {
				debug.PrintStack()
			}
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

	// only private chats allowed
	if user_id != message.Chat.Id {
		return
	}

	// skip forwared messages
	if message.ForwardDate > 0 {
		return
	}

	/* adding to users map */
	if _, ok := users[user_id]; !ok {
		users[user_id] = &User{
			command:  nil,
			accounts: make(map[string]*Account),
		}
	}

	command = nil
	reply_to_id := message.ReplyTo.MessageId
	user = users[user_id]

	if reply_to_id > 0 {
		// is reply
		reply_failed := true

		if user.command != nil && user.command.MsgId == reply_to_id {
			command = user.command
			reply_failed = false
		} else {
			user_accounts := user.accounts
			for _, account := range user_accounts {
				if jid, ok := account.messageJids[reply_to_id]; ok {
					account.client.SendMessage(jid, message.Text)
					return
				}
			}
		}

		if reply_failed {
			SendMessage(user_id, "Error. Reply wasn't sent")
			return
		}
	}

	if command == nil {
		command, err = parseCommand(message)
		if err != nil {
			SendMessage(message.From.Id, err.Error())
			return
		}
	}

	switch command.Cmd {
	case CMD_CONNECT:
		{
			switch command.Step {
			case 0:
				{
					command.Next("Enter your jabber id:")
					user.command = command
				}
			case 1:
				{
					if !EmailIsValid(message.Text) {
						command.Again("Invalid jabber id. Try again")
						break
					}
					command.Next("Enter your password:")
					command.Jid = message.Text
				}
			case 2:
				{
					err = Connect(command.From, command.Jid, message.Text,
						command.Host, command.Port)
					if err != nil {
						SendMessage(command.From, err.Error())
					}
				}
			default:
				log.Println("Something went wrong in connecting")
			}
		}
	case CMD_DISCONNECT:
		Disconnect(command.From)
	case CMD_BOT_MESSAGE:
		SendMessage(command.UserId, command.Message)
	case CMD_START:
		SendMessage(command.From, GREETING)
	case CMD_CHECK:
		text := `
		I'm alive.
		Users count: %d.
		`
		SendMessage(conf.AdminUserId, fmt.Sprintf(text, len(users)))
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
