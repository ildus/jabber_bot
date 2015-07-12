package main

import (
	"regexp"
)

var (
	email_re = regexp.MustCompile(".+@.+\\..+")
)

func EmailIsValid(text string) bool {
	matched := email_re.Match([]byte(text))
	return matched
}
