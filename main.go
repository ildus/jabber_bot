package main

import (
	"./telegrambot"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	xmpp "github.com/mattn/go-xmpp"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"strings"
	"time"
)

var (
	config        = flag.String("config", "settings.json", "configuration file")
	conf          Configuration
	db            *sql.DB
	bot           *telegrambot.Bot
	accountsTable string = "accounts"
	rostersTable  string = "rosters"
)

const (
	MAX_ACCOUNTS    = 500
	SQL_GET_ACCOUNT = `select host, username, password, use_tls from %s
						where user_id = $1`
	SQL_CHECK_ACCOUNT = `select exists(select 1 
									   from %s 
									   where username = $1 limit 1);`
	SQL_ADD_ACCOUNT = `insert into %s (user_id, host, username, 
										password, use_tls) 
					   values ($1, $2, $3, $4, $5);`
)

type Account struct {
	Host     string
	Username string
	Password string
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
	log.Printf("Bot username is: %s", info["username"].(string))
	bot.SetWebhook(path.Join("https://", conf.BaseDomain, conf.HookPath))
}

func addAccount(user_id int, host string,
	username string, password string, use_tls bool) error {
	/* adding to accounts table */
	sql_add := fmt.Sprintf(SQL_ADD_ACCOUNT, accountsTable)
	_, err := db.Exec(sql_add, user_id, host, username, password, use_tls)
	if err != nil {
		log.Println("Account adding error: %s", err)
		return errors.New("Account adding error")
	}
	return nil
}

func ListenAs(user_id int, channel chan string) {
	var (
		host, username, password string
		use_tls                  bool
		talk                     *xmpp.Client
		err                      error
	)

	defer close(channel)
	sql_acc := fmt.Sprintf(SQL_GET_ACCOUNT, accountsTable)
	err = db.QueryRow(sql_acc, user_id).Scan(
		&host, &username, &password, &use_tls)
	if err != nil {
		log.Printf("Apparently user %s does not exists: %s", user_id, err)
		return
	}

	if use_tls {
		xmpp.DefaultConfig = tls.Config{
			ServerName:         strings.Split(host, ":")[0],
			InsecureSkipVerify: false,
		}
	}

	options := xmpp.Options{
		Host:          host,
		User:          username,
		Password:      password,
		NoTLS:         !use_tls,
		Debug:         true,
		Session:       false,
		Status:        "xa",
		StatusMessage: "i'm just a very very smart robot",
	}
	log.Println("Connecting")
	talk, err = options.NewClient()
	log.Println("ok")
	if err != nil {
		log.Println("Cannot connect to server: ", err)
		return
	}

	go func() {
		log.Println("Connected")
		for {
			chat, err := talk.Recv()
			if err != nil {
				log.Println("Talk recieving error: ", err)
				channel <- "EOF"
				break
			}
			switch v := chat.(type) {
			case xmpp.Chat:
				fmt.Println(v.Remote, v.Text)
			case xmpp.Presence:
				fmt.Println(v.From, v.Show)
			}
		}
	}()
	for {
		message := <-channel
		if message == "EOF" {
			break
		}

		message = strings.TrimRight(message, "\n")
		tokens := strings.SplitN(message, " ", 2)
		if len(tokens) == 2 {
			talk.Send(xmpp.Chat{Remote: tokens[0], Type: "chat", Text: tokens[1]})
		}
	}
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
