package xmpp

/*
#cgo LDFLAGS: -lstrophe
#include "xmpp.h"
*/
import "C"
import "errors"
import "log"
import "sync"

/* includes created connections
key - jid
client - Client instance
*/
var (
	clients map[int]*Client = nil
	mutex   sync.Mutex
	counter int = 0
)

const (
	XMPP_CONN_CONNECT    = C.XMPP_CONN_CONNECT
	XMPP_CONN_DISCONNECT = C.XMPP_CONN_DISCONNECT
	XMPP_CONN_FAIL       = C.XMPP_CONN_FAIL
	XMPP_MESSAGE         = XMPP_CONN_FAIL + 1
)

type Event struct {
	EventType int
	Msg       *Message
}

type Message struct {
	MessageType string
	From        string
	Text        string
}

type Buddy struct {
	Name         string
	Jid          string
	Subscription string
}

type Client struct {
	Id       int
	Jid      string
	ConnInfo *C.xmpp_conn
	Channel  chan *Event
	Roster   []*Buddy
	listen   bool
}

/* when we get some message from connection,
	this callback is called

   clientId - who got message
   from - sender jid
   message - text of message

   function just get associated channel
   for this client_id (user connection)
   and sends filled Message to this channel
*/

//export go_message_callback
func go_message_callback(client_id C.int, msg_type *C.char, from *C.char,
	message *C.char) {

	clientId := int(client_id)
	log.Println("xmpp.Callback ", clientId)

	if client, ok := clients[clientId]; ok {
		var msg_type_i = C.GoString(msg_type)
		var from_i = C.GoString(from)
		var message_i = C.GoString(message)

		msg := &Message{
			MessageType: msg_type_i,
			From:        from_i,
			Text:        message_i,
		}
		client.Channel <- &Event{Msg: msg, EventType: XMPP_MESSAGE}
	}
}

//export go_roster_callback
func go_roster_callback(client_id C.int, roster *C.roster_item) {
	clientId := int(client_id)

	if client, ok := clients[clientId]; ok {
		item := roster
		result := make([]*Buddy, 0)
		for item != nil {
			buddy := &Buddy{
				Jid:          C.GoString(item.jid),
				Name:         C.GoString(item.name),
				Subscription: C.GoString(item.subscription),
			}
			result = append(result, buddy)
			item = (*C.roster_item)(item.next)
		}
		client.Roster = result
	}

	C.free_roster(roster)
}

//export go_conn_callback
func go_conn_callback(client_id C.int, event_type C.int) {
	clientId := int(client_id)
	log.Println("xmpp.ConnCallback ", clientId)

	if client, ok := clients[clientId]; ok {
		client.Channel <- &Event{EventType: int(event_type)}
		if event_type == XMPP_CONN_CONNECT {
			C.get_roster(client.ConnInfo.conn, client.ConnInfo.userdata)
		} else {
			client.listen = false
		}
	}
}

/* Opens jabber connection for client */
func (client *Client) Connect(pass string,
	host string, port uint16) error {

	jid_i := C.CString(client.Jid)
	pass_i := C.CString(pass)
	var host_i *C.char = nil

	log.Println("xmpp.Connect: ", client.Jid)

	if len(host) > 0 {
		host_i = C.CString(host)
	}

	mutex.Lock()
	counter += 1
	mutex.Unlock()

	client.ConnInfo = C.open_xmpp_conn(jid_i, pass_i, host_i,
		C.short(port), C.int(counter))

	if client.ConnInfo != nil {
		client.Id = counter
		clients[counter] = client
		return nil
	}
	return errors.New("Connection error")
}

/* Sends message to somebody */
func (client *Client) SendMessage(jid string, message string) {
	jid_i := C.CString(jid)
	msg_type := C.CString("chat")
	message_i := C.CString(message)
	C.send_message(client.ConnInfo.conn, msg_type, jid_i, message_i)
}

/*
 * Breaks getting events from jabber server
 * in client goroutine and deletes
 * this client from clients map
 */
func (client *Client) Disconnect() {
	C.disconnect_xmpp_conn(client.ConnInfo)
}

/*
 * Gets events from jabber connection,
 * if some event is happening and it is message, then it
 * goes to callback defined above
 */
func (client *Client) Listen() {
	client.Channel = make(chan *Event)
	go func() {
		client.listen = true
		for client.listen {
			C.check_xmpp_events(client.ConnInfo.ctx, 100)
		}
		C.close_xmpp_conn(client.ConnInfo)
		close(client.Channel)
		delete(clients, client.Id)
	}()
}

func Init() {
	C.init_xmpp_library()
	clients = make(map[int]*Client)
}

func Shutdown() {
	C.shutdown_xmpp_library()
}
