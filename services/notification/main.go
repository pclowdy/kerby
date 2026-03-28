package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/redis/go-redis/v9"
)

func main() {
	log.Println("Notification Service starting...")
	
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" { redisHost = "redis:6379" }
	rdb := redis.NewClient(&redis.Options{Addr: redisHost})

	go subscribeToOrders(rdb)

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	http.HandleFunc("/notifications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		
		ctx := context.Background()
		events, err := rdb.LRange(ctx, "recent_notifications", 0, 49).Result()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "[%s]", strings.Join(events, ","))
	})

	log.Println("Notification Service health check listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func subscribeToOrders(rdb *redis.Client) {
	ctx := context.Background()
	pubsub := rdb.Subscribe(ctx, "orders")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for msg := range ch {
		log.Printf("[Notification] Received Order Event: %s\n", msg.Payload)
		rdb.LPush(ctx, "recent_notifications", msg.Payload)
		rdb.LTrim(ctx, "recent_notifications", 0, 49)
	}
}
