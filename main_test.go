package main

import (
	"log"
	"testing"
)

const (
	sqlInit = `
	create table accounts_test (
		id serial primary key,
		user_id int,
		host varchar,
		username varchar unique,
		password varchar,
		use_tls bool
	);
	create table rosters_test (
		id serial primary key,
		account_id int,
		name varchar
	)
	`
	sqlTearDown = `
	drop table if exists accounts_test;
	drop table if exists rosters_test;
	`
)

func setUp() {
	accountsTable = "accounts_test"
	rostersTable = "rosters_test"

	loadConfiguration()
	_, err := db.Exec(sqlTearDown)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(sqlInit)
	if err != nil {
		log.Fatal(err)
	}
}

func tearDown() {
	_, err := db.Exec(sqlTearDown)
	if err != nil {
		log.Fatal(err)
	}
}

func TestAddAccount(t *testing.T) {
	var err error
	var host, username, password string
	var use_tls bool

	setUp()
	err = addAccount(1, "umarta.com", "ildus@umarta.com", "pass", true)
	if err != nil {
		t.Fatal("add account error")
	}
	err = addAccount(1, "umarta.com", "ildus@umarta.com", "pass", true)
	if err == nil {
		t.Fatal("Second add must fail")
	}
	err = db.QueryRow(`select host, username, password, 
		use_tls from accounts_test where user_id=$1`, 1).Scan(&host,
		&username, &password, &use_tls)
	if err != nil {
		t.Fatal("Fetch error:", err)
	}
	tearDown()
}

func TestListen(t *testing.T) {
	setUp()
	err := addAccount(1, "umarta.com:5222", "test@umarta.com", "testtest", true)
	if err != nil {
		t.Fatal("Account adding error", err)
	}
	ch := make(chan string)
	log.Println("Start listening")
	ListenAs(1, ch)
	//ch <- "ildus@umarta.com test test"
	tearDown()
}
