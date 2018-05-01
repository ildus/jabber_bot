FROM golang:alpine
RUN apk add --update git jansson-dev && rm -rf /var/cache/apk/*
RUN apk add openssh && rm -rf /var/cache/apk/*
EXPOSE 5001

RUN mkdir -p /deploy
RUN go get github.com/gorilla/mux
RUN git clone git@github.com:ildus/jabber_bot.git /deploy/jabber_bot
COPY start_bot.sh /deploy/start_bot.sh
RUN chmod +x /start_bot.sh

WORKDIR /deploy/jabber_bot
ENTRYPOINT /deploy/start_bot.sh
