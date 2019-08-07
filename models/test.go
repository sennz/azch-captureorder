package models

import (
	"crypto/tls"
	"net"
	"net/url"
	"context"
	"fmt"
	"io/ioutil"

	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	//"github.com/Microsoft/ApplicationInsights-Go/appinsights"
    "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	amqp10 "pack.ag/amqp"
	"gopkg.in/matryer/try.v1"
)

// Order represents the order json
type OrderNew struct {
	ID           			bson.ObjectId		`json:"id" bson:"_id,omitempty"`
	EmailAddress      string  				`json:"emailAddress"`
	Product           string  				`json:"product"`
	Total             float64 				`json:"total"`
	Status            string  				`json:"status"`
}

// Environment variables
var mongoHost = os.Getenv("MONGOHOST")
var mongoUsername = os.Getenv("MONGOUSER")
var mongoPassword = os.Getenv("MONGOPASSWORD")
var mongoSSL = false 
var mongoPort = ""
var amqpURL = os.Getenv("AMQPURL")
var teamName = os.Getenv("TEAMNAME")
var mongoPoolLimit = 25

// MongoDB variables
var mongoDBSession *mgo.Session
var mongoDBSessionError error

// MongoDB database and collection names
var mongoDatabaseName = "akschallenge"
var mongoCollectionName = "orders"
var mongoCollectionShardKey = "_id"

// AMQP 1.0 variables
var amqp10Client *amqp10.Client
var amqp10Session *amqp10.Session
var amqpSender *amqp10.Sender
var serivceBusName string

// Application Insights telemetry clients
//var ChallengeTelemetryClient appinsights.TelemetryClient
//var CustomTelemetryClient appinsights.TelemetryClient

// For tracking and code branching purposes
var isCosmosDb = strings.Contains(mongoHost, "documents.azure.com")
var db string // CosmosDB or MongoDB?

// ReadMongoPasswordFromSecret reads the mongo password from the flexvol mount if present
func ReadMongoPasswordFromSecret(file string) (string, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}

	secret := string(b)
	return secret, err

}

//// BEGIN: NON EXPORTED FUNCTIONS
func init() {
	
	// Log to stdout by default
	log.SetOutput(os.Stdout)

	rand.Seed(time.Now().UnixNano())

	// If there is a mongo-password secret in the flexvol mount reset mongoPassword var 
	if mongoPassword == "" {
		secret, err := ReadMongoPasswordFromSecret("/kvmnt/mongo-password")
		if err != nil {
			fmt.Print(err)
		}
		mongoPassword = secret
		fmt.Println(mongoPassword)
	}

	// Validate environment variables
	validateVariable(mongoHost, "MONGOHOST")
	validateVariable(mongoUsername, "MONGOUSERNAME")
	validateVariable(mongoPassword, "MONGOPASSWORD")
	//validateVariable(amqpURL, "AMQPURL")
	validateVariable(teamName, "TEAMNAME")

	var mongoPoolLimitEnv = os.Getenv("MONGOPOOL_LIMIT")
	if mongoPoolLimitEnv != "" {
		if limit, err := strconv.Atoi(mongoPoolLimitEnv); err == nil {
			mongoPoolLimit = limit
		}
	}
	log.Printf("MongoDB pool limit set to %v. You can override by setting the MONGOPOOL_LIMIT environment variable." , mongoPoolLimit)

	// Initialize the MongoDB client
	initMongo()

	// Initialize the AMQP client if AMQPURL is passed
	if amqpURL != "" {
		initAMQP()
	}
}

// Logs out value of a variable
func validateVariable(value string, envName string) {
	if len(value) == 0 {
		log.Printf("The environment variable %s has not been set", envName)
	} else {
		log.Printf("The environment variable %s is %s", envName, value)
	}
}



