package main

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Product struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Name        string             `bson:"name"`
	Price       float64            `bson:"price"`
	Description string             `bson:"description"`
	Image       string             `bson:"image"`
	Categories  []string           `bson:"categories"`
}

type Category struct {
	ID   primitive.ObjectID `bson:"_id,omitempty"`
	Name string             `bson:"name"`
}

type Order struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	CustomerID string             `bson:"customer_id"`
	Date       time.Time          `bson:"date"`
	Status     string             `bson:"status"`
	Products   []Product          `bson:"products"`
}

type Customer struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Name     string             `bson:"name"`
	Email    string             `bson:"email"`
	Password string             `bson:"password"`
}

type Discount struct {
	Code     string `bson:"code"`
	Discount int    `bson:"discount"`
	Rules    string `bson:"rules"`
}

type OrderRequest struct {
	CustomerID   string   `json:"customer_id"`
	ProductIDs   []string `json:"product_ids"`
	DiscountCode string   `json:"discount_code"`
}

type OrderResponse struct {
	OrderID primitive.ObjectID `json:"order_id"`
	Message string             `json:"message"`
}
