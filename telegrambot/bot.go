package telegrambot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

const (
	BASE_URL = "https://api.telegram.org/bot"
)

type Bot struct {
	Token string
	Hook  func(w http.ResponseWriter, r *http.Request)
}

type BotResult map[string]interface{}
type ServerResponse struct {
	ok          bool
	description string
	result      BotResult
}

func BotHandler(w http.ResponseWriter, r *http.Request) {
	var update map[string]interface{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("Webhook body read error")
		goto end
	}

	err = json.Unmarshal(body, &update)
	if err != nil {
		log.Println("Webhook json parse error")
		goto end
	}

	log.Println("We got update %s", update)

end:
	//telegram server must not know about our problems
	fmt.Fprintf(w, "OK\n")
}

func (bot *Bot) Command(cmd string,
	params *url.Values) *ServerResponse {

	var result map[string]interface{}
	var err error
	var resp *http.Response

	//construct url
	cmd_url := BASE_URL + bot.Token + "/" + cmd
	if params == nil {
		resp, err = http.Get(cmd_url)
	} else {
		resp, err = http.PostForm(cmd_url, *params)
	}
	if err != nil {
		log.Printf("Request error with cmd %s: %s", cmd, err)
		return nil
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &result)

	var ok, exists bool
	if ok, exists = result["ok"].(bool); !exists {
		log.Printf("Incorrect answer from telegram: %s", string(body))
		return nil
	}

	serverResponse := &ServerResponse{ok: ok}
	if !ok {
		log.Printf("Something is wrong with request: %s",
			result["description"].(string))
		serverResponse.description = result["description"].(string)
	} else {
		st := result["result"].(map[string]interface{})
		serverResponse.result = BotResult(st)
	}
	return serverResponse
}

func (bot *Bot) GetMe() (bool, BotResult) {
	resp := bot.Command("getMe", nil)
	if resp != nil && resp.ok {
		return true, resp.result
	}
	return false, nil
}

func (bot *Bot) SetWebhook(hookUrl string) bool {
	values := &url.Values{}
	values.Add("url", hookUrl)
	resp := &struct {
		ok bool
	}{ok: true} //bot.Command("setWebhook", values)
	if resp != nil && resp.ok {
		bot.Hook = BotHandler
		return true
	}
	return false
}
