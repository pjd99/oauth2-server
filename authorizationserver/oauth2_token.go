package authorizationserver

import (
	"fmt"
	"strings"
	"github.com/ory-am/fosite"
	"github.com/dgrijalva/jwt-go"
	"log"
	"net/http"
	"github.com/ory-am/fosite/storage"
	 "database/sql"
        _ "github.com/go-sql-driver/mysql"
)

var err error

var pubKey string = "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4f5wg5l2hKsTeNem/V41\nfGnJm6gOdrj8ym3rFkEU/wT8RDtnSgFEZOQpHEgQ7JL38xUfU0Y3g6aYw9QT0hJ7\nmCpz9Er5qLaMXJwZxzHzAahlfA0icqabvJOMvQtzD6uQv6wPEyZtDTWiQi9AXwBp\nHssPnpYGIn20ZZuNlX2BrClciHhCPUIIZOQn/MmqTD31jSyjoQoV7MhhMTATKJx2\nXrHhR+1DcKJzQBSTAGnpYVaqpsARap+nwRipr3nUTuxyGohBTSmjJ2usSeQXHI3b\nODIRe1AuTyHceAbewn8b462yEWKARdpd9AjQW5SIVPfdsz5B6GlYQ5LdYKtznTuy\n7wIDAQAB\n-----END PUBLIC KEY-----\n"



func tokenEndpoint(rw http.ResponseWriter, req *http.Request) {
	// This context will be passed to all methods.

	ctx := fosite.NewContext()

	// Need to find if refresh or password grant. If refresh need to pinpoint original user. If password use username from req.

	var username string

	if err := req.ParseForm(); err != nil {
		log.Printf("Cannot parse post form for email")
	}

	log.Printf("Grant Type: %s\n", req.PostForm.Get("grant_type"))

	if req.PostForm.Get("grant_type") == "password" {
		username = req.PostForm.Get("username")
	} else if req.PostForm.Get("grant_type") == "refresh_token"{
		
		refToken := req.PostForm.Get("refresh_token")
		
        token, err := jwt.Parse(refToken, func(token *jwt.Token) (interface{}, error) {
            // Don't forget to validate the alg is what you expect:
        	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
	        	return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
        	}
            anoPubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(pubKey))
            if err != nil {
            	fmt.Errorf("failed to parse DER encoded public key: " + err.Error())
            }
            return anoPubKey, nil
        })


        if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
            username = claims["email"].(string)
			
    	} else {
            fmt.Println(err)
			username = "578e280b2629a7da0416303b"
    	}
		
	} else {
		username = "578e280b2629a7da0416303b"
	}

	// Create an empty session object which will be passed to the request handlers
	mySessionData := newSession(username)

	if ((req.PostForm.Get("grant_type") == "password") || (req.PostForm.Get("grant_type") == "refresh_token")){


		/*****
		// Create an sql.DB and check for errors
    		db, err = sql.Open("mysql", "pjd99:oxford13@/ess_auth_server")
    		if err != nil {
        		panic(err.Error())    
    		}
    		// sql.DB should be long lived "defer" closes it once this function ends
    		defer db.Close()

    		// Test the connection to the database
    		err = db.Ping()
    		if err != nil {
        		panic(err.Error())
    		}
		*****/

		var db *sql.DB = storage.GetDatabase()

   		// Grab from the database 
    	var databaseUsername  string
    	var databasePassword  string
		var databaseFirst_name  string
        var databaseLast_name  string
		var databaseEmail  string
        var databaseCreated  string
		var databaseScope string
    	err := db.QueryRow("SELECT user_name, password, first_name, last_name, email, scope, created FROM users WHERE email=?", username).Scan(&databaseUsername, &databasePassword, &databaseFirst_name, &databaseLast_name, &databaseEmail, &databaseScope, &databaseCreated)
   		//if err == nil {
        	//	fmt.Printf("username: %s and password: %s", databaseUsername, databasePassword)
        	//	return
    		//}
		mySessionData.JWTClaims.Add("email", databaseEmail)
		mySessionData.JWTClaims.Add("username", databaseUsername)
		mySessionData.JWTClaims.Add("created", databaseCreated)
		mySessionData.JWTClaims.Add("name", map[string]string{"last": databaseLast_name, "first":databaseFirst_name})
		
		//var scopeItems []string
	    result := strings.Split(databaseScope, ",")
	       // Display all elements.
	       //for i := range result {
			   //scopeItems = append(scopeItems, result[i])
	           //fmt.Println(result[i])
			   //}
		mySessionData.JWTClaims.Add("scope", result)
		//mySessionData.JWTClaims.Add("scope", []string{"profile", "apps"})

		var databaseAppEUI string

        	rows, err := db.Query("SELECT applications.app_eui FROM applications INNER JOIN user_application ON applications.app_eui = user_application.app_eui INNER JOIN users ON user_application.user_id = users.user_id WHERE users.email = ?", username)
        	if err != nil {
                	log.Fatal(err)
        	}
        	defer rows.Close()

        	var apps []string

        	for rows.Next() {
                	err := rows.Scan(&databaseAppEUI)
                	if err != nil {
                        	log.Fatal(err)
                	}
                	apps = append(apps, databaseAppEUI)
        	}
        	err = rows.Err()
        	if err != nil {
                	log.Fatal(err)
        	}
		mySessionData.JWTClaims.Add("apps", apps)


		clientID, clientSecret, ok := req.BasicAuth()
		if !ok {
			log.Printf("HTTP Authorization header missing or invalid")
		} else {
			mySessionData.JWTClaims.Add("client", clientID)
		}

		log.Printf("Client secret: %s\n", clientSecret)
		mySessionData.JWTClaims.Add("valid", true)
		mySessionData.JWTClaims.Add("_id", "578e280b2629a7da0416303b")
	}

	// This will create an access request object and iterate through the registered TokenEndpointHandlers to validate the request.
	accessRequest, err := oauth2.NewAccessRequest(ctx, req, mySessionData)

	log.Printf("Request info: %s", req)


	// Catch any errors, e.g.:
	// * unknown client
	// * invalid redirect
	// * ...
	if err != nil {
		log.Printf("Error occurred in NewAccessRequest: %s\nStack: \n%s", err, err.(stackTracer).StackTrace())
		oauth2.WriteAccessError(rw, accessRequest, err)
		return
	} 

	//else {
	//	currentSession := accessRequest.GetSession()
        //	if currentSession != nil {
        //        	userName :=  currentSession.GetUsername()
        //        	log.Printf("Uname or email: %s\n", userName)
	//		log.Printf("Session data:  %s\n", currentSession)
        //	}

	//	mySessionData.JWTClaims.Add("email", "pauldoherty@rfproximity.com")
        //	mySessionData.JWTClaims.Add("scope", []string{"profile", "apps"})
        //	mySessionData.JWTClaims.Add("client", "ttnctl")
        //	mySessionData.JWTClaims.Add("username", "pjd99")
        //	mySessionData.JWTClaims.Add("created", "2015-11-12T13:12:27.332Z")
	//      mySessionData.JWTClaims.Add("name", map[string]string{"last": "Doherty", "first": "Paul"})
       	//	mySessionData.JWTClaims.Add("valid", true)
        //	mySessionData.JWTClaims.Add("_id", "578e280b2629a7da0416303b")
	//        mySessionData.JWTClaims.Add("apps", []string{"70B3D57ED000124B"})
	//}

	// Grant requested scopes
	for _, scope := range accessRequest.GetRequestedScopes() {
		accessRequest.GrantScope(scope)
	}

	// Next we create a response for the access request. Again, we iterate through the TokenEndpointHandlers
	// and aggregate the result in response.
	response, err := oauth2.NewAccessResponse(ctx, req, accessRequest)
	if err != nil {
		log.Printf("Error occurred in NewAccessResponse: %s\nStack: \n%s", err, err.(stackTracer).StackTrace())
		oauth2.WriteAccessError(rw, accessRequest, err)
		return
	}

	// All done, send the response.
	oauth2.WriteAccessResponse(rw, accessRequest, response)

	// The client now has a valid access token
}
