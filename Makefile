build:
	go build

run: build
	./jabber_bot

# i have set up nginx on my server, so i'm just redirect proxy to my local
# development server for testing current code
tunnel:
	ssh -f -N -R 127.0.0.1:9000:127.0.0.1:9000 ${to}
