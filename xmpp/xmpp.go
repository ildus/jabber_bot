package xmpp

/*
#cgo LDFLAGS: -lstrophe
#include "xmpp.h"
*/
import "C"
import "time"
import "errors"

/* includes created connections
	key - jid
	client - Client instance
*/
var clients map[string]*Client = nil

type Message struct {
	MessageType string
	From        string
	Text        string
}

type Client struct {
	Jid      string
	ConnInfo *C.xmpp_conn
	Channel  chan *Message
	listen   bool
}

/* when we get some message from connection,
	this callback is called

   jid - who got message
   from - sender jid
   message - text of message

   function just get associated channel
   for this jid (user connection)
   and sends filled Message to this channel
*/

//export go_message_callback
func go_message_callback(jid *C.char, msg_type *C.char, from *C.char,
	message *C.char) {

	var jid_i = C.GoString(jid)

	if client, ok := clients[jid_i]; ok {
		var msg_type_i = C.GoString(msg_type)
		var from_i = C.GoString(from)
		var message_i = C.GoString(message)

		msg := &Message{
			MessageType: msg_type_i,
			From:        from_i,
			Text:        message_i,
		}
		client.Channel <- msg
	}
}

/* Opens jabber connection for client */
func (client *Client) Connect(pass string,
	host string, port uint16) error {

	jid_i := C.CString(client.Jid)
	pass_i := C.CString(pass)
	var host_i *C.char = nil

	if len(host) > 0 {
		host_i = C.CString(host)
	}
	client.ConnInfo = C.open_xmpp_conn(jid_i, pass_i, host_i,
		C.short(port))
	if client.ConnInfo != nil {
		clients[client.Jid] = client
		return nil
	}
	return errors.New("Connection error")
}

/* Sends message to somebody */
func (client *Client) SendMessage(jid string, message string) {
	jid_i := C.CString(jid)
	msg_type := C.CString("chat")
	message_i := C.CString(message)
	C.send_message(client.ConnInfo.conn, client.ConnInfo.ctx,
		msg_type, jid_i, message_i)
}

/* Breaks getting events from jabber server
	in client goroutine and deletes
	this client from clients map
*/
func (client *Client) Disconnect() {
	client.listen = false
	delete(clients, client.Jid)
}

/* Gets events from jabber connection,
	if some event is happening and it is message, then it
	goes to callback defined above
*/
func (client *Client) Listen() {
	client.Channel = make(chan *Message)
	go func() {
		client.listen = true
		for client.listen {
			C.check_xmpp_events(client.ConnInfo.ctx)
			time.Sleep(50 * time.Millisecond)
		}
		C.close_xmpp_conn(client.ConnInfo)
		close(client.Channel)
	}()
}

func Init() {
	C.init_xmpp_library()
	clients = make(map[string]*Client)
}

func Shutdown() {
	C.shutdown_xmpp_library()
}
