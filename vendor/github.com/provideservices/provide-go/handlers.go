package provide

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

const defaultResultsPerPage = 25

// CORSMiddleware is a working middlware for using CORS with gin
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Encoding, Authorization, Cache-Control, Content-Length, Content-Type, Origin, User-Agent, X-CSRF-Token, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Expose-Headers", "X-Total-Results-Count")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// TrackAPICalls returns gin middleware for tracking API calls
func TrackAPICalls() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer trackAPICall(c)
		c.Next()
	}
}

// AuthorizedSubjectID returns the requested JWT subject if it matches
func AuthorizedSubjectID(c *gin.Context, subject string) *uuid.UUID {
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
	ParseBearerAuthorizationHeader(c, &keyfn)
	uuidV4, err := uuid.FromString(id)
	if err != nil {
		return nil
	}
	return &uuidV4
}

// Paginate the current request given the page number and results per page;
// returns the modified SQL query and adds x-total-results-count header to
// the response
func Paginate(c *gin.Context, db *gorm.DB, model interface{}) *gorm.DB {
	page := int64(1)
	rpp := int64(defaultResultsPerPage)
	if c.Query("page") != "" {
		if _page, err := strconv.ParseInt(c.Query("page"), 10, 8); err == nil {
			page = _page
		}
	}
	if c.Query("rpp") != "" {
		if _rpp, err := strconv.ParseInt(c.Query("rpp"), 10, 8); err == nil {
			rpp = _rpp
		}
	}
	query, totalResults := paginate(db, model, page, rpp)
	if totalResults != nil {
		c.Header("x-total-results-count", fmt.Sprintf("%d", *totalResults))
	}
	return query
}

// ParseBearerAuthorizationHeader parses a bearer authorization header
// expecting to find a valid JWT token; returns the token if present
func ParseBearerAuthorizationHeader(c *gin.Context, keyfunc *func(_jwtToken *jwt.Token) (interface{}, error)) (*jwt.Token, error) {
	authorization := c.GetHeader("authorization")
	if authorization == "" {
		return nil, errors.New("No authentication header provided")
	}
	hdrprts := strings.Split(authorization, "bearer ")
	if len(hdrprts) != 2 {
		return nil, fmt.Errorf("Failed to parse bearer authentication header: %s", authorization)
	}
	authorization = hdrprts[1]
	jwtToken, err := jwt.Parse(authorization, func(_jwtToken *jwt.Token) (interface{}, error) {
		if keyfunc != nil {
			fn := *keyfunc
			return fn(_jwtToken)
		}
		return nil, nil
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to parse bearer authentication header as valid JWT; %s", err.Error())
	}
	return jwtToken, err
}
