package main

import (
	"./telegrambot"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"time"
	//"github.com/mattn/go-xmpp"
)

var (
	config = flag.String("config", "settings.json", "configuration file")
	conf   Configuration
	db     *sql.DB
	bot    *telegrambot.Bot
)

const (
	MAX_ACCOUNTS      = 500
	SQL_GET_ACCOUNTS  = `select host, username, password from accounts;`
	SQL_CHECK_ACCOUNT = `select exists(select 1 
									   from accounts 
									   where username = $1 limit 1);`
	SQL_ADD_ACCOUNT = `insert into accounts (host, username, password) 
					   values ($1, $2, $3);`
)

type Account struct {
	Host     string
	Username string
	Password string
}

type Configuration struct {
	Listen     int    `json:"listen"`
	Database   string `json:"database"`
	Token      string `json:"token"`
	BaseDomain string `json:"base_domain"`
	HookPath   string `json:"hook_path"`
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
	db, err = sql.Open("postgres", conf.Database)
	if err != nil {
		log.Fatal(err)
	}
}

func setupBot() {
	bot = &telegrambot.Bot{Token: conf.Token}
	ok, info := bot.GetMe()
	if !ok {
		log.Fatal("Bot setup error")
	}
	log.Println("Bot username is: %s", info["username"].(string))
	bot.SetWebhook(path.Join("https://", conf.BaseDomain, conf.HookPath))
}

func listen() {
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
