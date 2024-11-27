package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWT(t *testing.T) {
	uuid := uuid.New()
	t.Logf("UUID is: %v", uuid)
	expiresIn := time.Duration(60 * time.Second)
	signed, err := MakeJWT(uuid, "bobby", expiresIn)
	if err != nil {
		t.Fatalf("Failed to created JWT: %v", err)	
	}

	t.Logf("Created a JWT: %v", signed)

	valid, err := ValidateJWT(signed, "bobby")
	if err != nil {
		t.Fatalf("Could not validate JWT: %v", err)
	}

	t.Logf("Valid JWT, userid: %v", valid)
}

func TestInvalidSecret(t *testing.T) {
	uuid := uuid.New()
	t.Logf("UUID is: %v", uuid)
	expiresIn := time.Duration(60 * time.Second)
	signed, err := MakeJWT(uuid, "bobby", expiresIn)
	if err != nil {
		t.Fatalf("Failed to created JWT: %v", err)	
	}

	t.Logf("Created a JWT: %v", signed)

	_, err = ValidateJWT(signed, "booby")
	if err != nil {
		t.Logf("Invalid secret (good thing)")
	} else {
		t.Fatal("Valid secret(BAD THING)")
	}
}

func TestExpiredToken(t *testing.T) {
	uuid := uuid.New()
	t.Logf("UUID is: %v", uuid)
	expiresIn := time.Duration(1 * time.Millisecond)
	signed, err := MakeJWT(uuid, "bobby", expiresIn)
	if err != nil {
		t.Fatalf("Failed to created JWT: %v", err)	
	}

	t.Logf("Created a JWT: %v", signed)

	_, err = ValidateJWT(signed, "booby")
	if err != nil {
		t.Logf("Token expired (good thing)")
	} else {
		t.Fatal("Valid token(BAD THING)")
	}
}

func TestBearer(t *testing.T) {

	header := http.Header{}
	header.Add("Authorization", "Bearer abc123")

	token, err := GetBearerToken(header)
	if err != nil {
		t.Fatal(err)
	}

	if token != "abc123" {
		t.Fatalf("Expected token to be abc123, got %v", token)
	}

	t.Logf("Token valid")
}