package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
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

func initFirebase() {
	opt := option.WithCredentialsFile("serviceAccountKey.json")

	app,err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		log.Fatal("Firebase init error:", err)
	}

	client, err = app.Messaging(context.Background())
	if err != nil{
		log.Fatal("Messaging init error:", err)	
	}
}

func initDB(){
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err !=nil {
		log.Fatal("MongoDB connection error:", err)
	}
	collection = mongoClient.Database("fcm_db").Collection("tokens")
	fmt.Println("Connected to MongoDB")
}

func saveToken(w http.ResponseWriter, r *http.Request) {
 w.Header().Set("Content-Type", "application/json")

 var req TokenRequest
 err := json.NewDecoder(r.Body).Decode(&req)
 if err != nil {
	http.Error(w, "Invalid Request", http.StatusBadRequest)
	return
 }

 filter := bson.M{"token": req.Token}
 update := bson.M{"$set": bson.M{
	"token": req.Token,
	"updatedAt": time.Now(),
},
"$setOnInsert":bson.M{
	"createdAt": time.Now(),
},
}
opts := options.Update().SetUpsert(true)
_, err = collection.UpdateOne(context.Background(), filter, update, opts)
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}

	fmt.Println("Token saved/updated:", req.Token)

	json.NewEncoder(w).Encode(map[string]string{
		"message": "token saved",
	})
}

func sendNotification(w http.ResponseWriter, r *http.Request) {
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		http.Error(w, "DB error", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())

	for cursor.Next(context.Background()) {
		var doc bson.M
		cursor.Decode(&doc)

		token := doc["token"].(string)

		message := &messaging.Message{
			Notification: &messaging.Notification{
				Title: "Backend Notification",
				Body:  "Sent from Go + MongoDB!",
			},
			Token: token,
		}

		_, err := client.Send(context.Background(), message)
		if err != nil {
			fmt.Println("Error sending to token:", token, err)
		} else {
			fmt.Println("Sent to:", token)
		}
	}

	fmt.Fprintln(w, "Notifications sent")
}

func main() {
	initFirebase()
	initDB()

	http.HandleFunc("/save-token", saveToken)
	http.HandleFunc("/send", sendNotification)

	http.Handle("/", http.FileServer(http.Dir("../frontend")))
	fmt.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}


