Jabber transport for Telegram
-----------------------------

This is telegram bot that implements jabber transport in telegram. Now it can
connect, disconnect, receive messages, and support replying to messages

Building
----------

Strophe library used for xmpp connection so it must be installed

	# on mac
	brew install libstrophe
	# on linux, get sources, ./configure && make install
	go get github.com/gorilla/mux
	go build

Setup
-----

Create settings.json in root. Format:

	{
		"listen": 9000, //port
		"token": "asdf", //telegram auth token
		"base_domain": "example.com", //your domain
		"hook_path": "/bot/hook", // url where will updates come
		"admin_user_id": 234234 // if you want get messages (not commands) from users
	}

Then you must setup base server (like nginx), proxy `hook_path` in its config
to bot port.

Commands
--------

* /connect jid pass <host> <port>
* /disconnect
