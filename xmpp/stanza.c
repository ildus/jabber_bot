#include "strophe.h"
#include "stanza.h"

xmpp_stanza_t *
stanza_create_roster_iq(xmpp_ctx_t *ctx)
{
    xmpp_stanza_t *iq = xmpp_stanza_new(ctx);
    xmpp_stanza_set_name(iq, "iq");
    xmpp_stanza_set_type(iq, "get");
    xmpp_stanza_set_id(iq, "roster");

    xmpp_stanza_t *query = xmpp_stanza_new(ctx);
    xmpp_stanza_set_name(query, "query");
    xmpp_stanza_set_ns(query, XMPP_NS_ROSTER);

    xmpp_stanza_add_child(iq, query);
    xmpp_stanza_release(query);

    return iq;
}

xmpp_stanza_t *
stanza_create_message(xmpp_ctx_t *ctx, char *to, char *message)
{
	xmpp_stanza_t	*msg,
					*body,
					*text;

	msg = xmpp_stanza_new(ctx);
	xmpp_stanza_set_name(msg, "message");
	xmpp_stanza_set_type(msg, "chat");
	xmpp_stanza_set_attribute(msg, "to", to);

	body = xmpp_stanza_new(ctx);
	text = xmpp_stanza_new(ctx);

	xmpp_stanza_set_name(body, "body");
	xmpp_stanza_set_text(text, message);
	xmpp_stanza_add_child(body, text);
	xmpp_stanza_add_child(msg, body);

	xmpp_stanza_release(text);
	xmpp_stanza_release(body);

	return msg;
}
