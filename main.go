package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/robfig/cron"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client             *mongo.Client
	requestsPerIP      = make(map[string]int)
	requestsPerIPMutex = &sync.Mutex{}
)

func initMongoDB() {
	// Set up MongoDB client
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	var err error
	client, err = mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Ping the MongoDB server to check the connection
	err = client.Ping(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Connected to MongoDB!")
}

func rateLimiterMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the IP address of the request
		ip := r.RemoteAddr

		// Lock the mutex to synchronize access to the map
		requestsPerIPMutex.Lock()
		defer requestsPerIPMutex.Unlock()

		// Check if the IP address has already made 100 requests in the last minute
		if requestsPerIP[ip] >= 100 {
			// If so, return an error response
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		// Increment the number of requests for the IP address
		requestsPerIP[ip]++

		// Schedule a function to decrement the number of requests for the IP address after one minute
		time.AfterFunc(time.Minute, func() {
			requestsPerIPMutex.Lock()
			defer requestsPerIPMutex.Unlock()
			requestsPerIP[ip]--
		})

		// Call the next handler in the chain
		next.ServeHTTP(w, r)
	})
}

func main() {
	// Initialize the MongoDB connection
	initMongoDB()

	// Create a new router
	router := mux.NewRouter()

	// Define the route for placing an order
	router.HandleFunc("/place-order", placeOrder).Methods("POST")

	// Define the route for generating a CSV report
	router.HandleFunc("/generate-report", generateCSVReport).Methods("GET")

	// Add rate limiter middleware to the router
	router.Use(rateLimiterMiddleware)

	// Create a new cron job scheduler
	c := cron.New()

	// Add a cron job to run at midnight every day
	c.AddFunc("@midnight", func() {
		runPendingOrderReminderTask()
	})

	// Start the cron job scheduler
	c.Start()

	// Start the server
	fmt.Println("Server started on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
