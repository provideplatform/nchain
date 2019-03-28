package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	uuid "github.com/kthomas/go.uuid"
	provide "github.com/provideservices/provide-go"
)

func AuthorizedSubjectId(c *gin.Context, subject string) *uuid.UUID {
	var id string
	keyfn := func(jwtToken *jwt.Token) (interface{}, error) {
		if claims, ok := jwtToken.Claims.(jwt.MapClaims); ok {
			if sub, subok := claims["sub"].(string); subok {
				subprts := strings.Split(sub, ":")
				if len(subprts) != 2 {
					return nil, fmt.Errorf("JWT subject malformed; %s", sub)
				}
				if subprts[0] != subject {
					return nil, fmt.Errorf("JWT claims specified non-%s subject: %s", subject, subprts[0])
				}
				id = subprts[1]
			}
		}
		return nil, nil
	}
	provide.ParseBearerAuthorizationHeader(c, &keyfn)
	uuidV4, err := uuid.FromString(id)
	if err != nil {
		return nil
	}
	return &uuidV4
}

func Render(obj interface{}, status int, c *gin.Context) {
	c.Header("content-type", "application/json; charset=UTF-8")
	c.Writer.WriteHeader(status)
	if &obj != nil && status != http.StatusNoContent {
		encoder := json.NewEncoder(c.Writer)
		encoder.SetIndent("", "    ")
		if err := encoder.Encode(obj); err != nil {
			panic(err)
		}
	} else {
		c.Header("content-length", "0")
	}
}

func RenderError(message string, status int, c *gin.Context) {
	err := map[string]*string{}
	err["message"] = &message
	Render(err, status, c)
}

func RequireParams(requiredParams []string, c *gin.Context) error {
	var errs []string
	for _, param := range requiredParams {
		if c.Query(param) == "" {
			errs = append(errs, param)
		}
	}
	if len(errs) > 0 {
		msg := strings.Trim(fmt.Sprintf("missing required parameters: %s", strings.Join(errs, ", ")), " ")
		RenderError(msg, 400, c)
		return errors.New(msg)
	}
	return nil
}
