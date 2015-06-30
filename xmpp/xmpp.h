#ifndef XMPP_H
#define XMPP_H

#include "strophe.h"

typedef struct xmpp_conn
{
	void *conn;
	void *ctx;
} xmpp_conn;

void check_xmpp_events(void *ctxp);
void close_xmpp_conn(xmpp_conn *conn);
void init_xmpp_library(void);
void shutdown_xmpp_library(void);
xmpp_conn *open_xmpp_conn(char *jid, char *pass, char *host, short port); 

int send_message(xmpp_conn_t * const conn, 
				 xmpp_ctx_t *ctx, 
				 char *type, char *to, char *message);
#endif
