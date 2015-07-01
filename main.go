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
	"time"
)

var (
	config        = flag.String("config", "settings.json", "configuration file")
	conf          Configuration
	bot           *telegrambot.Bot
	accountsTable string = "accounts"
	rostersTable  string = "rosters"

	accounts map[int]*Account
)

const (
	MAX_ACCOUNTS = 500
)

type Account struct {
	Jid  string
	Host string
	Port uint16

	client *xmpp.Client
}

type Configuration struct {
	Listen      int      `json:"listen"`
	Database    string   `json:"database"`
	Token       string   `json:"token"`
	BaseDomain  string   `json:"base_domain"`
	HookPath    string   `json:"hook_path"`
	TestAccount []string `json:"test_account"`
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

	account := &Account{Jid: jid, Host: host, Port: port, client: client}
	accounts[user_id] = account
	client.Listen()
	go func() {
		defer delete(accounts, user_id)
		for msg := range client.Channel {
			account.SendMessage(msg)
		}
	}()

	return nil
}

func Disconnect(user_id int) {
	if account, ok := accounts[user_id]; ok {
		account.client.Disconnect()
	}
}

func (a *Account) SendMessage(msg *xmpp.Message) {
}

func listen() {
	if bot.Hook == nil {
		log.Fatal("Hook is not installed")
	}
	routes := mux.NewRouter()
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
