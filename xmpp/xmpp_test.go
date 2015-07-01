package xmpp

import (
	"flag"
	"fmt"
	"testing"
	"time"
)

var (
	jid  = flag.String("jid", "test@example.com", "test jid")
	pass = flag.String("pass", "pass", "test password")
)

func TestConnection(t *testing.T) {
	if *jid == "test@example.com" {
		fmt.Println("Specify jid and password for testing")
		fmt.Println("	-jid=<account>")
		fmt.Println("	-pass=<pass>")
		return
	}
	Init()
	client := Client{Jid: *jid}
	client.Connect(*pass, "", 0)
	client.Listen()

	msg := <-client.Channel
	client.SendMessage(msg.From, "i'm out")
	time.Sleep(2 * time.Second)
	client.Disconnect()
	fmt.Println(msg)
	Shutdown()
}
