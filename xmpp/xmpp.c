#include "stdlib.h"
#include "stdio.h"
#include "strophe.h"
#include "string.h"
#include "assert.h"
#include "xmpp.h"

extern void go_message_callback(char *, char *, char *, char *);

int send_message(xmpp_conn_t * const conn, 
				 xmpp_ctx_t *ctx, 
				 char *type, char *to, char *message)
{
	xmpp_stanza_t *reply, *body, *text;
	char *msg_type = type;

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
	char *jid, *from, *message, *msg_type;
	xmpp_ctx_t *ctx = (xmpp_ctx_t*)userdata;

	if (!xmpp_stanza_get_child_by_name(stanza, "body")) 
		return 1;
	if (xmpp_stanza_get_attribute(stanza, "type") != NULL 
			&& !strcmp(xmpp_stanza_get_attribute(stanza, "type"), "error")) 
		return 1;
	
	from = xmpp_stanza_get_attribute(stanza, "from");
	message = xmpp_stanza_get_text(xmpp_stanza_get_child_by_name(stanza, "body"));
	msg_type = xmpp_stanza_get_type(stanza);
	
	jid = strdup(xmpp_conn_get_jid(conn)); 
	go_message_callback(jid, msg_type, from, message);
	free(jid);

	return 1;
}

void conn_handler(xmpp_conn_t * const conn, const xmpp_conn_event_t status, 
		  const int error, xmpp_stream_error_t * const stream_error,
		  void * const userdata)
{
    xmpp_ctx_t *ctx = (xmpp_ctx_t *)userdata;

	if (status == XMPP_CONN_CONNECT)
	{
		xmpp_stanza_t *pres;
		printf("DEBUG: connected\n");
		xmpp_handler_add(conn, message_handler, NULL, "message", NULL, ctx);
		
		/* Send initial <presence/> so that we appear online to contacts */
		pres = xmpp_stanza_new(ctx);
		xmpp_stanza_set_name(pres, "presence");
		xmpp_send(conn, pres);
		xmpp_stanza_release(pres);
	} 
	else 
	{
		fprintf(stderr, "DEBUG: disconnected\n");
		xmpp_stop(ctx);
    }
}

/* trigger event loop check */
void check_xmpp_events(void *ctxp)
{
	xmpp_ctx_t *ctx = (xmpp_ctx_t *)ctxp;
	xmpp_run_once(ctxp, 10000);
}

/* release our connection and context */
void close_xmpp_conn(xmpp_conn *conn)
{
    xmpp_conn_release((xmpp_conn_t *)conn->conn);
	xmpp_ctx_free((xmpp_ctx_t *)conn->ctx);
}

void init_xmpp_library()
{
	xmpp_initialize();
}

void shutdown_xmpp_library()
{
	xmpp_shutdown();
}

xmpp_conn *open_xmpp_conn(char *jid, char *pass, char *host, short port) 
{
	int err;
	xmpp_ctx_t *ctx;
	xmpp_log_t *log;
	xmpp_conn_t *conn;
	xmpp_conn	*result;

	log = xmpp_get_default_logger(XMPP_LEVEL_DEBUG);
	assert(log);

	ctx = xmpp_ctx_new(NULL, log);
	assert(ctx);

	/* create a connection */
	conn = xmpp_conn_new(ctx);

	/* setup authentication information */
	xmpp_conn_set_jid(conn, jid);
	xmpp_conn_set_pass(conn, pass);

	/* initiate connection */
    err = xmpp_connect_client(conn, host, port, conn_handler, ctx);
	free(jid);
	free(pass);
	if (host != NULL) free(host);

	if (err != 0) 
		return NULL;

	result = (xmpp_conn *) malloc(sizeof(xmpp_conn));
	result->conn = (void *)conn;
	result->ctx = (void *)ctx;
	printf("%p, %p", conn, ctx);
	return result;
}
