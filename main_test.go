package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCommandParse(t *testing.T) {
	var err error
	text := "/connect user@example.com pass host.com 50"
	command, _ := parseCommand(text)
	assert.Equal(t, command.Cmd, CMD_CONNECT)
	assert.Equal(t, command.Jid, "user@example.com")
	assert.Equal(t, command.Password, "pass")
	assert.Equal(t, command.Port, int16(50))

	text = "/disconnect"
	command, _ = parseCommand(text)
	assert.Equal(t, command.Cmd, CMD_DISCONNECT)

	text = "/check"
	command, _ = parseCommand(text)
	assert.Equal(t, command.Cmd, CMD_CHECK)

	text = "/conn asdf asdf"
	command, err = parseCommand(text)
	assert.Nil(t, command)
	assert.NotNil(t, err)

	text = "/connect user@example.com"
	command, err = parseCommand(text)
	assert.Nil(t, command)
	assert.NotNil(t, err)

	text = "/connect user@example.com pass"
	command, err = parseCommand(text)
	assert.NotNil(t, command)
	assert.Nil(t, err)
}
