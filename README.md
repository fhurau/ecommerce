# Simple Web API

This project is a Go web application with rate limiting and MongoDB integration.

## Description

This project implements a web server with rate limiting functionality using Gorilla Mux for routing and Go's `net/http` package. It includes a middleware that limits the number of requests per IP address, allowing only 100 requests per minute.

The server is integrated with MongoDB, utilizing the official MongoDB Go driver (`go.mongodb.org/mongo-driver`) to connect, ping, and interact with the MongoDB server.

Additionally, the project includes a cron job scheduler using the `github.com/robfig/cron` package. It sets up a cron job to run a specific task every day at midnight.

## Installation

1. Clone this repository: `git clone https://github.com/fhurau/ecommerce`
2. Change into the project directory: `cd ecommerce`
3. Make sure MongoDB is installed and running on mongodb://localhost:27017. Adjust the connection URI in the initMongoDB() function if needed.
4. Start the server: go run .
5. You can test the API endpoint by sending a POST/GET request to http://localhost:8080/API with the required JSON payload.

## API Routes
The following routes are available in the application:

1. '/place-order' - [POST] Place an order.
2. '/generate-report' - [GET] Generate a CSV report.
