#ifndef XMPP_H
#define XMPP_H

#include "strophe.h"

typedef struct xmpp_userdata
{
	int client_id;
} xmpp_userdata;

typedef struct xmpp_conn
{
	void *conn;
	void *ctx;
	void *userdata;
} xmpp_conn;

typedef struct roster_item {
	char *jid;
	char *name;
	char *subscription;
	void *next;
} roster_item;

void check_xmpp_events(void *ctxp, int timeout);
void close_xmpp_conn(xmpp_conn *conn);
void disconnect_xmpp_conn(xmpp_conn *conn);
void init_xmpp_library(void);
void shutdown_xmpp_library(void);

xmpp_conn *open_xmpp_conn(char *jid, char *pass, char *host, short port, int client_id);

void send_message(void *conn_i, char *type, char *to, char *message);
void get_roster(void *conn_i, void * const userdata);
void free_roster(roster_item *roster);
#endif
