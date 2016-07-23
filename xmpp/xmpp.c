#include "stdlib.h"
#include "stdio.h"
#include "strophe.h"
#include "string.h"
#include "assert.h"
#include "xmpp.h"
#include "stanza.h"

extern void go_message_callback(int, char *, char *, char *);
extern void go_conn_callback(int, int);
extern void go_roster_callback(int, roster_item *);

void
send_message(void *conn_i, char *type, char *to, char *message)
{
	xmpp_conn_t	*conn = (xmpp_conn_t *)conn_i;
	xmpp_ctx_t *ctx = xmpp_conn_get_context(conn);

	xmpp_stanza_t *msg = stanza_create_message(ctx, to, message);
	xmpp_send(conn, msg);
	xmpp_stanza_release(msg);
}

static int
handle_roster(xmpp_conn_t * const conn,
		 xmpp_stanza_t * const stanza,
		 void * const userdata)
{
    xmpp_stanza_t	*query,
					*item;
	roster_item		*roster = NULL, *last = NULL;
    const char		*type,
					*name;

    type = xmpp_stanza_get_type(stanza);
    if (strcmp(type, "error") == 0) {
		fprintf(stderr, "ERROR: roster query failed\n");
		return 1;
	}
    else
	{
		query = xmpp_stanza_get_child_by_name(stanza, "query");
		for (item = xmpp_stanza_get_children(query); item; item = xmpp_stanza_get_next(item))
		{
			// do not forget to free returned texts after
			roster_item *buddy = (roster_item *)malloc(sizeof(roster_item));
			buddy->next = NULL;
			buddy->name = xmpp_stanza_get_attribute(item, "name");
			buddy->jid = xmpp_stanza_get_attribute(item, "jid");
			buddy->subscription = xmpp_stanza_get_attribute(item, "subscription");

			// save first buddy
			if (roster == NULL)
				roster = buddy;

			// make a list
			if (last != NULL)
				last->next = buddy;

			last = buddy;
		}

		assert(userdata != NULL);
		int client_id = ((xmpp_userdata *)userdata)->client_id;
		go_roster_callback(client_id, roster);
    }

    return 0;
}

void
free_roster(roster_item *roster)
{
	while (roster != NULL) {
		roster_item *item = roster;
		roster = item->next;
		free(item);
	}
}

void
get_roster(void *conn_i, void * const userdata)
{
	xmpp_conn_t *conn = (xmpp_conn_t *)conn_i;
	xmpp_ctx_t *ctx = xmpp_conn_get_context(conn);
	xmpp_stanza_t *iq = stanza_create_roster_iq(ctx);
	xmpp_id_handler_add(conn, handle_roster, "roster", userdata);
	xmpp_send(conn, iq);
	xmpp_stanza_release(iq);
}

/* Handle all message to send them to go callback */
static int
handle_message(xmpp_conn_t * const conn,
					xmpp_stanza_t * const stanza,
					void * const userdata)
{
	char *from, *message, *msg_type;
	xmpp_ctx_t *ctx = xmpp_conn_get_context(conn);

	if (!xmpp_stanza_get_child_by_name(stanza, "body"))
	{
		return 1;
	}
	if (xmpp_stanza_get_attribute(stanza, "type") != NULL
			&& !strcmp(xmpp_stanza_get_attribute(stanza, "type"), "error"))
	{
		return 1;
	}

	from = xmpp_stanza_get_attribute(stanza, "from");
	message = xmpp_stanza_get_text(xmpp_stanza_get_child_by_name(stanza, "body"));
	msg_type = xmpp_stanza_get_type(stanza);

	int client_id = ((xmpp_userdata *)userdata)->client_id;
	go_message_callback(client_id, msg_type, from, message);

	return 0;
}

static void
handle_conn(xmpp_conn_t * const conn, const xmpp_conn_event_t status,
		  const int error, xmpp_stream_error_t * const stream_error,
		  void * const userdata)
{
    xmpp_ctx_t *ctx = xmpp_conn_get_context(conn);

	/* send the status to high level */
	int client_id = ((xmpp_userdata *)userdata)->client_id;
	go_conn_callback(client_id, status);

	if (status == XMPP_CONN_CONNECT)
	{
		xmpp_stanza_t *pres;
		printf("DEBUG: User connected\n");
		xmpp_handler_add(conn, handle_message, NULL, "message", NULL, userdata);

		/* Send initial <presence/> so that we appear online to contacts */
		pres = xmpp_stanza_new(ctx);
		xmpp_stanza_set_name(pres, "presence");
		xmpp_send(conn, pres);
		xmpp_stanza_release(pres);
	}
	else
	{
		xmpp_stop(ctx);
		fprintf(stderr, "DEBUG: disconnected\n");
    }
}

/* trigger event loop check */
void
check_xmpp_events(void *ctxp, int timeout)
{
	xmpp_ctx_t *ctx = (xmpp_ctx_t *)ctxp;
	xmpp_run_once(ctxp, timeout);
}

void
disconnect_xmpp_conn(xmpp_conn *conn)
{
	xmpp_disconnect((xmpp_conn_t *)conn->conn);
}

/* release our connection and context */
void
close_xmpp_conn(xmpp_conn *conn)
{
    xmpp_conn_release((xmpp_conn_t *)conn->conn);
	xmpp_ctx_free((xmpp_ctx_t *)conn->ctx);
	free(conn->userdata);
	free(conn);
}

void
init_xmpp_library()
{
	xmpp_initialize();
}

void
shutdown_xmpp_library()
{
	xmpp_shutdown();
}

xmpp_conn *
open_xmpp_conn(char *jid, char *pass, char *host, short port,
	int client_id)
{
	int err;
	xmpp_ctx_t *ctx;
	xmpp_log_t *log;
	xmpp_conn_t *conn;
	xmpp_conn	*result;
	xmpp_userdata *userdata;

	log = xmpp_get_default_logger(XMPP_LEVEL_ERROR);
	assert(log);

	ctx = xmpp_ctx_new(NULL, log);
	assert(ctx);

	/* create a connection */
	conn = xmpp_conn_new(ctx);

	/* setup authentication information */
	xmpp_conn_set_jid(conn, jid);
	xmpp_conn_set_pass(conn, pass);

	/* create struct for passing through callbacks */
	userdata = (xmpp_userdata *) malloc(sizeof(xmpp_userdata));
	userdata->client_id = client_id;

	/* initiate connection */
    err = xmpp_connect_client(conn, host, port, handle_conn, userdata);
	free(jid);
	free(pass);
	if (host != NULL) free(host);

	if (err != 0)
		return NULL;

	result = (xmpp_conn *) malloc(sizeof(xmpp_conn));
	result->conn = (void *)conn;
	result->ctx = (void *)ctx;
	result->userdata = (void *)userdata;
	return result;
}
