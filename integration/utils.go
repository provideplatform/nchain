// +build integration

package integration

import (
	"fmt"

	provide "github.com/provideservices/provide-go/api/ident"
)

type User struct {
	firstName string
	lastName  string
	email     string
	password  string
}

type Application struct {
	name        string
	description string
}

type Organization struct {
	name        string
	description string
}


func userFactory(firstName, lastName, email, password string) (*provide.User, error) {
	return provide.CreateUser("", map[string]interface{}{
		"first_name": firstName,
		"last_name":  lastName,
		"email":      email,
		"password":   password,
	})
}

func getUserToken(firstName, lastName, email, password string) (*provide.Token, error) {
	// set up the user - who cares if we get a 409 - user already exists, just error if we can't get a token
	_, _ = userFactory(firstName, lastName, email, password)

	token, err := provide.Authenticate(email, password)
	if err != nil {
		return nil, fmt.Errorf("error generating token. Error: %s", err.Error())
	}

	return token.Token, nil

}
