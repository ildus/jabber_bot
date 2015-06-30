package xmpp

import (
	"flag"
	"fmt"
	"testing"
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

	msg := <-client.channel
	client.Disconnect()
	fmt.Println(msg)
	Shutdown()
}
