# Idea

* User sends `connect` command to bot
* Bot opens connection with jabber account and waits for messages
* If it gets message, bot sends message to user with some id of sender
* Using that id user can reply to message 
* Bot keeps connected users somewhere, and if it fails it sends message 
	to users about need to reconnection
* User sends `disconnect` to bot and bot forgets about him
* User can check the connection

# Commands format

## Connection

/connect user@example.com <pass> [<host>] [<port>]
/check
/disconnect

## Messages format

Incoming: #1 sender@example.com hey, here is your message
Outcoming: #1 ok, got it

## Implementation details

Message from server: map[update_id:7.34575208e+08 message:
						map[date:1.435737795e+09 
							text:work work 
							message_id:11 
							from: map[first_name:Ivan 
									  username:user1 id:4.663978e+07
									  ] 
							chat: map[username:user1 
										id:4.663978e+07 
										first_name:Ivan]
							]
						]
Manual setting of hook:
curl --data "url=https:url" https://api.telegram.org/bot<token>/setWebhook
