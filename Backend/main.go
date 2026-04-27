package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/api/option"
)

var client *messaging.Client
var collection *mongo.Collection

type TokenRequest struct {
	Token string `json:"token"`
}

func enableCORS(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
}

func initFirebase() {
	cred := os.Getenv("FIREBASE_CREDENTIALS")
	opt := option.WithCredentialsJSON([]byte(cred))

	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatal(err)
	}

	client, err = app.Messaging(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}

func initDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoURI := os.Getenv("MONGODB_URI")

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}

	collection = mongoClient.Database("fcm_db").Collection("tokens")
}

func saveToken(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)

	var req TokenRequest
	json.NewDecoder(r.Body).Decode(&req)

	collection.InsertOne(context.Background(), bson.M{
		"token": req.Token,
	})

	fmt.Fprintln(w, "Token saved")
}

func sendNotification(w http.ResponseWriter, r *http.Request) {
	enableCORS(&w)

	cursor, _ := collection.Find(context.Background(), bson.M{})
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var doc bson.M
		cursor.Decode(&doc)

		token := doc["token"].(string)

		message := &messaging.Message{
			Notification: &messaging.Notification{
				Title: "Hello",
				Body:  "Sent from Go backend",
			},
			Token: token,
		}

		client.Send(context.Background(), message)
	}

	fmt.Fprintln(w, "Sent")
}

func main() {
	initFirebase()
	initDB()

	http.HandleFunc("/save-token", saveToken)
	http.HandleFunc("/send", sendNotification)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Running on", port)
	http.ListenAndServe("0.0.0.0:"+port, nil)
}