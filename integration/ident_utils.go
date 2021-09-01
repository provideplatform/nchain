// +build integration nchain failing rinkeby ropsten kovan goerli nobookie basic bookie readonly bulk

package integration

import (
	"fmt"

	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideplatform/provide-go/api/ident"
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

func getUser(testID uuid.UUID) User {
	nchainUser := User{
		"nchain" + testID.String(),
		"user " + testID.String(),
		"nchain.user" + testID.String() + "@email.com",
		"secrit_password",
	}
	return nchainUser
}

func userFactoryByTestId(testID uuid.UUID) (*provide.User, error) {

	identUser := getUser(testID)

	return provide.CreateUser("", map[string]interface{}{
		"first_name": identUser.firstName,
		"last_name":  identUser.lastName,
		"email":      identUser.email,
		"password":   identUser.password,
	})
}

func userFactory(firstName, lastName, email, password string) (*provide.User, error) {
	return provide.CreateUser("", map[string]interface{}{
		"first_name": firstName,
		"last_name":  lastName,
		"email":      email,
		"password":   password,
	})
}

func getUserToken(email, password string) (*provide.Token, error) {
	authResponse, err := provide.Authenticate(email, password)
	if err != nil {
		return nil, fmt.Errorf("error authenticating user. Error: %s", err.Error())
	}

	return authResponse.Token, nil
}

func getUserTokenByTestId(testID uuid.UUID) (*provide.Token, error) {
	nchainUser := getUser(testID)
	authResponse, err := provide.Authenticate(nchainUser.email, nchainUser.password)
	if err != nil {
		return nil, fmt.Errorf("error authenticating user. Error: %s", err.Error())
	}

	return authResponse.Token, nil
}

func getOrgToken(testID uuid.UUID) (*string, error) {

	nchainOrg := Organization{
		"org" + testID.String(),
		"orgdesc " + testID.String(),
	}

	userToken, err := UserAndTokenFactory(testID)
	if err != nil {
		return nil, fmt.Errorf("error creating user and getting token. Error: %s ", err.Error())
	}

	userOrg, err := orgFactory(*userToken, nchainOrg.name, nchainOrg.description)
	if err != nil {
		return nil, fmt.Errorf("error creating org name(%s) in ident. Error %s", nchainOrg.name, err.Error())
	}

	orgToken, err := orgTokenFactory(*userToken, userOrg.ID)
	if err != nil {
		return nil, fmt.Errorf("error getting token for org (name: %s). Error: %s", nchainOrg.name, err.Error())
	}

	return orgToken.Token, nil
}

func AppAndTokenFactory(testID uuid.UUID, userID uuid.UUID) (*string, error) {
	_, _ = userFactoryByTestId(testID)

	token, err := getUserTokenByTestId(testID)
	if err != nil {
		return nil, fmt.Errorf("error generating user token. Error: %s", err.Error())
	}

	nchainApp := Application{
		"app" + testID.String(),
		"appdesc " + testID.String(),
	}

	app, err := appFactory(*token.AccessToken, nchainApp.name, nchainApp.description)
	if err != nil {
		return nil, fmt.Errorf("error generating application. Error: %s", err.Error())
	}

	appToken, err := appTokenFactory(*token.Token, app.ID)
	if err != nil {
		return nil, fmt.Errorf("error generating app token. Error: %s", err.Error())
	}

	return appToken.Token, nil
}

func UserAndTokenFactory(testID uuid.UUID) (*string, error) {
	// set up the user - who cares if we get a 409 - user already exists, just error if we can't get a token
	_, _ = userFactoryByTestId(testID)

	token, err := getUserTokenByTestId(testID)
	if err != nil {
		return nil, fmt.Errorf("error generating token. Error: %s", err.Error())
	}
	return token.AccessToken, nil
}

func appFactory(token, name, desc string) (*provide.Application, error) {
	return provide.CreateApplication(token, map[string]interface{}{
		"name":        name,
		"description": desc,
	})
}

func orgFactory(token, name, desc string) (*provide.Organization, error) {
	return provide.CreateOrganization(token, map[string]interface{}{
		"name":        name,
		"description": desc,
	})
}

func apporgFactory(token, applicationID, organizationID string) error {
	return provide.CreateApplicationOrganization(token, applicationID, map[string]interface{}{
		"organization_id": organizationID,
	})
}

func appTokenFactory(token string, applicationID uuid.UUID) (*provide.Token, error) {
	return provide.CreateToken(token, map[string]interface{}{
		"application_id": applicationID,
	})
}

func appUserTokenFactory(token string, applicationID, userID uuid.UUID) (*provide.Token, error) {
	return provide.CreateToken(token, map[string]interface{}{
		"application_id": applicationID,
		"user_id":        userID,
	})
}

func orgTokenFactory(token string, organizationID uuid.UUID) (*provide.Token, error) {
	return provide.CreateToken(token, map[string]interface{}{
		"organization_id": organizationID,
	})
}
