package utils

import (
	"io/ioutil"
	"testing"
	"time"
)

func TestJwtCreate(t *testing.T) {

	keyBytes, err := ioutil.ReadFile("test_private.pem")
	if err != nil {
		t.Fatal(err)
	}
	token, err := CreateJWT("abc", keyBytes, time.Hour*24)
	if err != nil {
		t.Fatal(err)
	}
	expiration, err := ParseJWTTokenExpirationTime(token)
	if err != nil {
		t.Fatal(err)
	}
	current := time.Now().UTC()
	if expiration.Before(current) {
		t.Fatal("JWT error, expiration should be after current time")
	}
}
