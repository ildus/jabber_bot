#include "stdlib.h"
#include "stdio.h"
#include "strophe.h"
#include "string.h"
#include "assert.h"
#include "xmpp.h"

extern void go_message_callback(int, char *, char *, char *);
extern void go_conn_callback(int, int);

int send_message(void *conn_i, void *ctx_i,
				 char *type, char *to, char *message)
{
	xmpp_stanza_t *reply, *body, *text;
	char *msg_type = type;
	xmpp_conn_t	*conn = (xmpp_conn_t *)conn_i;
	xmpp_ctx_t *ctx = (xmpp_ctx_t *)ctx_i;

	if (msg_type == NULL)
		msg_type = "chat";

	reply = xmpp_stanza_new(ctx);
	xmpp_stanza_set_name(reply, "message");
	xmpp_stanza_set_type(reply, msg_type);
	xmpp_stanza_set_attribute(reply, "to", to);

	body = xmpp_stanza_new(ctx);
	xmpp_stanza_set_name(body, "body");

	text = xmpp_stanza_new(ctx);
	xmpp_stanza_set_text(text, message);
	xmpp_stanza_add_child(body, text);
	xmpp_stanza_add_child(reply, body);

	xmpp_send(conn, reply);
	xmpp_stanza_release(reply);
	return 0;
}

/* Handle all message to send them to go callback */
int message_handler(xmpp_conn_t * const conn,
					xmpp_stanza_t * const stanza,
					void * const userdata)
{
	char *from, *message, *msg_type;
	xmpp_ctx_t *ctx = xmpp_conn_get_context(conn);

	if (!xmpp_stanza_get_child_by_name(stanza, "body"))
	{
		printf("no body");
		return 1;
	}
	if (xmpp_stanza_get_attribute(stanza, "type") != NULL
			&& !strcmp(xmpp_stanza_get_attribute(stanza, "type"), "error"))
	{
		printf("some error");
		return 1;
	}

	from = xmpp_stanza_get_attribute(stanza, "from");
	message = xmpp_stanza_get_text(xmpp_stanza_get_child_by_name(stanza, "body"));
	msg_type = xmpp_stanza_get_type(stanza);

	int client_id = ((xmpp_userdata *)userdata)->client_id;
	go_message_callback(client_id, msg_type, from, message);

	return 1;
}

void conn_handler(xmpp_conn_t * const conn, const xmpp_conn_event_t status,
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
		xmpp_handler_add(conn, message_handler, NULL, "message", NULL, userdata);

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
void check_xmpp_events(void *ctxp, int timeout)
{
	xmpp_ctx_t *ctx = (xmpp_ctx_t *)ctxp;
	xmpp_run_once(ctxp, timeout);
}

void disconnect_xmpp_conn(xmpp_conn *conn)
{
	xmpp_disconnect((xmpp_conn_t *)conn->conn);
}

/* release our connection and context */
void close_xmpp_conn(xmpp_conn *conn)
{
    xmpp_conn_release((xmpp_conn_t *)conn->conn);
	xmpp_ctx_free((xmpp_ctx_t *)conn->ctx);
	free(conn->userdata);
	free(conn);
}

void init_xmpp_library()
{
	xmpp_initialize();
}

void shutdown_xmpp_library()
{
	xmpp_shutdown();
}

xmpp_conn *open_xmpp_conn(char *jid, char *pass, char *host, short port,
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
    err = xmpp_connect_client(conn, host, port, conn_handler, userdata);
	free(jid);
	free(pass);
	if (host != NULL) free(host);

	if (err != 0)
		return NULL;

	result = (xmpp_conn *) malloc(sizeof(xmpp_conn));
	result->conn = (void *)conn;
	result->ctx = (void *)ctx;
	result->userdata = userdata;
	return result;
}
