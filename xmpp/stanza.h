#ifndef STANZA_H
#define STANZA_H

xmpp_stanza_t *stanza_create_roster_iq(xmpp_ctx_t *ctx);
xmpp_stanza_t *stanza_create_message(xmpp_ctx_t *ctx, char *to, char *message);

#endif
