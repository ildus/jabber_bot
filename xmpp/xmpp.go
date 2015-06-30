package xmpp

/*
#cgo LDFLAGS: -lstrophe
#include "xmpp.h"
*/
import "C"
import "fmt"
import "time"

var clients map[string]*Client = nil

type Message struct {
	MessageType string
	From        string
	Text        string
}

type Client struct {
	Jid      string
	ConnInfo *C.xmpp_conn
	channel  chan *Message
	listen   bool
}

//export go_message_callback
func go_message_callback(jid *C.char, msg_type *C.char, from *C.char,
	message *C.char) {

	var jid_i = C.GoString(jid)
	var msg_type_i = C.GoString(msg_type)
	var from_i = C.GoString(from)
	var message_i = C.GoString(message)

	if client, ok := clients[jid_i]; ok {
		msg := &Message{
			MessageType: msg_type_i,
			From:        from_i,
			Text:        message_i,
		}
		client.channel <- msg
	}
	fmt.Println(jid_i, msg_type_i, from_i, message_i)
}

func (client *Client) Connect(pass string,
	host string, port C.short) {

	jid_i := C.CString(client.Jid)
	pass_i := C.CString(pass)
	var host_i *C.char = nil

	if len(host) > 0 {
		host_i = C.CString(host)
	}
	client.ConnInfo = C.open_xmpp_conn(jid_i, pass_i, host_i, port)
	clients[client.Jid] = client
}

func (client *Client) Disconnect() {
	client.listen = false
	delete(clients, client.Jid)
}

func (client *Client) Listen() {
	client.channel = make(chan *Message)
	go func() {
		client.listen = true
		for client.listen {
			C.check_xmpp_events(client.ConnInfo.ctx)
			time.Sleep(50 * time.Millisecond)
		}
		C.close_xmpp_conn(client.ConnInfo)
		close(client.channel)
	}()
}

func Init() {
	C.init_xmpp_library()
	clients = make(map[string]*Client)
}

func Shutdown() {
	C.shutdown_xmpp_library()
}
