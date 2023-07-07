package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func placeOrder(w http.ResponseWriter, r *http.Request) {
	var orderReq OrderRequest
	err := json.NewDecoder(r.Body).Decode(&orderReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Validate the input data
	if orderReq.CustomerID == "" || len(orderReq.ProductIDs) == 0 {
		http.Error(w, "Invalid order request", http.StatusBadRequest)
		return
	}

	// Convert CustomerID to ObjectID
	customerID, err := primitive.ObjectIDFromHex(orderReq.CustomerID)
	if err != nil {
		http.Error(w, "Invalid customer ID", http.StatusBadRequest)
		return
	}

	// Check if the customer exists
	customer := Customer{}
	customersCollection := client.Database("e-commerce").Collection("customers")
	err = customersCollection.FindOne(context.Background(), bson.M{"_id": customerID}).Decode(&customer)
	if err != nil {
		http.Error(w, "Customer not found", http.StatusNotFound)
		return
	}

	// Check if the products exist
	products := []Product{}
	productsCollection := client.Database("e-commerce").Collection("product")

	// Convert ProductIDs to ObjectIDs
	objectIDs := make([]primitive.ObjectID, len(orderReq.ProductIDs))
	for i, productID := range orderReq.ProductIDs {
		objectID, err := primitive.ObjectIDFromHex(productID)
		if err != nil {
			http.Error(w, "Invalid product ID", http.StatusBadRequest)
			return
		}
		objectIDs[i] = objectID
	}

	cursor, err := productsCollection.Find(context.Background(), bson.M{"_id": bson.M{"$in": objectIDs}})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())
	err = cursor.All(context.Background(), &products)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if the discount code is valid
	discount := Discount{}
	discountsCollection := client.Database("e-commerce").Collection("discounts")
	err = discountsCollection.FindOne(context.Background(), bson.M{"code": orderReq.DiscountCode}).Decode(&discount)
	if err != nil && err != mongo.ErrNoDocuments {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create the order
	order := Order{
		CustomerID: orderReq.CustomerID,
		Date:       time.Now(),
		Status:     "pending",
		Products:   products,
	}

	// Apply the discount if it exists
	if err == nil {
		order.applyDiscount(discount)
	}

	// Save the order to the database
	ordersCollection := client.Database("e-commerce").Collection("orders")
	result, err := ordersCollection.InsertOne(context.Background(), order)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	orderRes := OrderResponse{
		OrderID: result.InsertedID.(primitive.ObjectID),
		Message: "Order placed successfully",
	}

	json.NewEncoder(w).Encode(orderRes)
}

func (order *Order) applyDiscount(discount Discount) {
	switch discount.Code {
	case "IC003":
		order.applyPercentageDiscount(0.1)
	case "IC042":
		order.applyCategoryDiscount("electronic", 0.05)
	case "IC015":
		order.applyWeekendDiscount(0.1)
	}
}

func (order *Order) applyPercentageDiscount(percentage float64) {
	for i := range order.Products {
		order.Products[i].Price -= order.Products[i].Price * percentage
	}
}

func (order *Order) applyCategoryDiscount(category string, percentage float64) {
	for i := range order.Products {
		if contains(order.Products[i].Categories, category) {
			order.Products[i].Price -= order.Products[i].Price * percentage
		}
	}
}

func (order *Order) applyWeekendDiscount(percentage float64) {
	if order.Date.Weekday() == time.Saturday || order.Date.Weekday() == time.Sunday {
		order.applyPercentageDiscount(percentage)
	}
}

func contains(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

func (order *Order) CalculateTotalPrice() float64 {
	total := 0.0
	for _, product := range order.Products {
		total += product.Price
	}
	return total
}

func generateCSVReport(w http.ResponseWriter, r *http.Request) {
	// Fetch order information from the database
	ordersCollection := client.Database("e-commerce").Collection("orders")
	cursor, err := ordersCollection.Find(context.Background(), bson.M{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	// Create a new CSV file
	file, err := os.Create("report.csv")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()

	// Create a CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write the header row
	header := []string{"Order ID", "Customer Name", "Order Date", "Total Price", "Status"}
	writer.Write(header)

	// Iterate over the orders and write the data rows
	for cursor.Next(context.Background()) {
		var order Order
		err := cursor.Decode(&order)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		row := []string{
			order.ID.Hex(),
			getCustomerName(order.CustomerID),
			order.Date.Format(time.RFC3339),
			strconv.FormatFloat(order.CalculateTotalPrice(), 'f', 2, 64),
			order.Status,
		}
		writer.Write(row)
	}

	// Set the response headers
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=report.csv")
}

func getCustomerName(customerID string) string {
	customerObjectID, err := primitive.ObjectIDFromHex(customerID)
	if err != nil {
		return ""
	}

	customer := Customer{}
	customersCollection := client.Database("e-commerce").Collection("customers")
	err = customersCollection.FindOne(context.Background(), bson.M{"_id": customerObjectID}).Decode(&customer)
	if err != nil {
		return ""
	}
	return customer.Name
}

func getCustomerData(customerID string) (string, string) {
	customerObjectID, err := primitive.ObjectIDFromHex(customerID)
	if err != nil {
		return "", ""
	}

	customer := Customer{}
	customersCollection := client.Database("e-commerce").Collection("customers")
	err = customersCollection.FindOne(context.Background(), bson.M{"_id": customerObjectID}).Decode(&customer)
	if err != nil {
		return "", ""
	}
	return customer.Name, customer.Email
}

func sendOrderReminderEmail(customerID string) {
	customerName, customerEmail := getCustomerData(customerID)

	// Compose the email body
	orderProducts := getCustomerOrderProducts(customerID)
	orderDetails := formatOrderDetails(orderProducts)
	emailBody := fmt.Sprintf("Dear %s,\n\nYou have a pending order with the following products:\n%s\n\nPlease complete the checkout process by visiting the link: %s\n\nBest regards,\nYour E-commerce Team", customerName, orderDetails, "checkout-link")

	log.Printf("Sending order reminder email to customer %s (Email: %s):\n%s", customerName, customerEmail, emailBody)
}

func runPendingOrderReminderTask() {
	// Get the current time in the server's timezone
	now := time.Now()

	// Query the database for customers with pending orders
	pendingOrders := getCustomersWithPendingOrders()

	// Iterate over the pending orders and send reminder emails
	for _, order := range pendingOrders {
		// Calculate the time until midnight
		midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
		durationUntilMidnight := midnight.Sub(now)

		// Schedule the email to be sent at midnight
		time.AfterFunc(durationUntilMidnight, func() {
			sendOrderReminderEmail(order.CustomerID)
		})
	}
}

func getCustomersWithPendingOrders() []Order {

	ordersCollection := client.Database("e-commerce").Collection("orders")
	cursor, err := ordersCollection.Find(context.Background(), bson.M{"status": "pending"})
	if err != nil {
		log.Println("Error querying database:", err)
		return nil
	}
	defer cursor.Close(context.Background())

	var orders []Order
	err = cursor.All(context.Background(), &orders)
	if err != nil {
		log.Println("Error decoding orders:", err)
		return nil
	}

	return orders
}

func getCustomerOrderProducts(customerID string) []Product {

	customerObjectID, err := primitive.ObjectIDFromHex(customerID)
	if err != nil {
		log.Println("Error converting customerID:", err)
		return nil
	}

	productsCollection := client.Database("e-commerce").Collection("products")
	cursor, err := productsCollection.Find(context.Background(), bson.M{"customer_id": customerObjectID})
	if err != nil {
		log.Println("Error querying database:", err)
		return nil
	}
	defer cursor.Close(context.Background())

	var products []Product
	err = cursor.All(context.Background(), &products)
	if err != nil {
		log.Println("Error decoding products:", err)
		return nil
	}

	return products
}

func formatOrderDetails(orderProducts []Product) string {

	var orderDetails string
	for _, product := range orderProducts {
		orderDetails += fmt.Sprintf("- %s: $%.2f\n", product.Name, product.Price)
	}

	return orderDetails
}
