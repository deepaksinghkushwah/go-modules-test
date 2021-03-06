package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"database/sql"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"

	"github.com/gin-gonic/gin"
)

var jwtKey = []byte("my_secret_key")

//User struct to define user
type User struct {
	ID        int
	Username  string
	Password  string
	Email     string
	FirstName string
	LastName  string
	Status    int
}

var (
	router = gin.Default()
)

//Credentials Create a struct to read the username and password from the request body
type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

//Claims Create a struct that will be encoded to a JWT.
// We add jwt.StandardClaims as an embedded type, to provide fields like expiry time
type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

//JWTToken to bind token in request body
type JWTToken struct {
	Token string `json:"token"`
}

func main() {
	// "Signin" and "Welcome" are the handlers that we will implement
	router.POST("/signin", Signin)
	router.POST("/welcome", Welcome)
	router.POST("/refresh", Refresh)

	// start the server on port 8000
	log.Fatal(router.Run(":8080"))
}

//Signin Create the Signin handler
func Signin(c *gin.Context) {
	var creds Credentials
	// Get the JSON body and decode into credentials
	if err := c.ShouldBindJSON(&creds); err != nil {
		c.JSON(http.StatusNotAcceptable, gin.H{"error": err})
		return
	}

	user, err := getUserFromDB(creds.Username, creds.Password)
	fmt.Println(user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "You are not authorzied",
		})
		return
	}

	// Declare the expiration time of the token
	// here, we have kept it as 5 minutes
	expirationTime := time.Now().Add(5 * time.Minute)
	// Create the JWT claims, which includes the username and expiry time
	claims := &Claims{
		Username: user.Username,
		StandardClaims: jwt.StandardClaims{
			// In JWT, the expiry time is expressed as unix milliseconds
			ExpiresAt: expirationTime.Unix(),
		},
	}

	// Declare the token with the algorithm used for signing, and the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Create the JWT string
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		// If there is an error in creating the JWT return an internal server error
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Finally, we set the client cookie for "token" as the JWT we just generated
	// we also set an expiry time which is the same as the token itself
	c.JSON(http.StatusOK, gin.H{
		"Name":    "token",
		"Value":   tokenString,
		"Expires": expirationTime,
	})
}

//Welcome create welcome handler function
func Welcome(c *gin.Context) {
	// We can obtain the session token from the requests cookies, which come with every request
	claims, err := CheckValidToken(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	// Finally, return the welcome message to the user, along with their
	// username given in the token
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("Welcome %s!", claims.Username),
	})
}

//Refresh to refresh your jwt token
func Refresh(c *gin.Context) {

	claims, err := CheckValidToken(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}
	// (END) The code uptil this point is the same as the first part of the `Welcome` route

	// We ensure that a new token is not issued until enough time has elapsed
	// In this case, a new token will only be issued if the old token is within
	// 30 seconds of expiry. Otherwise, return a bad request status
	if time.Unix(claims.ExpiresAt, 0).Sub(time.Now()) > 2*time.Hour {
		t := time.Unix(claims.ExpiresAt, 0)
		ut := t.Format(time.RFC3339)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Your token is new for session, wait for " + (ut) + " to renew token",
		})
		return
	}

	// Now, create a new token for the current use, with a renewed expiration time
	expirationTime := time.Now().Add(5 * time.Minute)
	claims.ExpiresAt = expirationTime.Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal Server Error",
		})
		return
	}
	// Set the new token as the users `session_token` cookie
	c.JSON(http.StatusOK, gin.H{
		"Name":    "session_token",
		"Value":   tokenString,
		"Expires": expirationTime,
	})

}

//CheckValidToken to check the token validation
func CheckValidToken(c *gin.Context) (*Claims, error) {
	var jwrtToken JWTToken
	if err := c.ShouldBindJSON(&jwrtToken); err != nil {
		return nil, customError("Token must be supplied")
	}
	tknStr := jwrtToken.Token

	// Get the JWT string from the cookie

	// Initialize a new instance of `Claims`
	claims := &Claims{}

	// Parse the JWT string and store the result in `claims`.
	// Note that we are passing the key in this method as well. This method will return an error
	// if the token is invalid (if it has expired according to the expiry time we set on sign in),
	// or if the signature does not match
	tkn, err := jwt.ParseWithClaims(tknStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			return nil, customError("Invalid token signature")
		}
		return nil, customError("Invalid token signature")
	}
	if !tkn.Valid {
		return nil, customError("Unauthorized Access, token not valid")
	}
	return claims, nil
}

func customError(message string) error {
	return errors.New(message)
}

func getUserFromDB(expUsername, expPassword string) (*User, error) {
	var user User
	db, err := sql.Open("mysql", "root:deepak@tcp(127.0.0.1:3306)/go-jwt")
	if err != nil {
		log.Fatalln(err)
	}

	defer db.Close()

	err = db.QueryRow("SELECT id, username, password, email, first_name, last_name FROM `user` WHERE username = ? AND password = ?", expUsername, expPassword).Scan(&user.ID, &user.Username, &user.Password, &user.Email, &user.FirstName, &user.LastName)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
