package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	authpb "k8s-microservices-app/proto/auth"
	invpb "k8s-microservices-app/proto/inventory"
	orderpb "k8s-microservices-app/proto/order"
)

var (
	authClient  authpb.AuthServiceClient
	orderClient orderpb.OrderServiceClient
	invClient   invpb.InventoryServiceClient
)

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func getUserIdFromToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if len(authHeader) < 8 {
		return "", fmt.Errorf("Unauthorized")
	}
	token := authHeader[7:]
	authResp, err := authClient.ValidateToken(context.Background(), &authpb.ValidateTokenRequest{Token: token})
	if err != nil || !authResp.Valid {
		return "", fmt.Errorf("Invalid token")
	}
	return authResp.UserId, nil
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "Method Not Allowed", 405); return }
	var creds struct { Username, Password string }
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil { http.Error(w, "Bad Request", 400); return }

	resp, err := authClient.Login(context.Background(), &authpb.LoginRequest{Username: creds.Username, Password: creds.Password})
	w.Header().Set("Content-Type", "application/json")
	if err != nil || resp.Error != "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Login failed"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"token": resp.Token})
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" { http.Error(w, "Method Not Allowed", 405); return }
	var creds struct { Username, Password string }
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil { http.Error(w, "Bad Request", 400); return }

	resp, err := authClient.Register(context.Background(), &authpb.RegisterRequest{Username: creds.Username, Password: creds.Password})
	w.Header().Set("Content-Type", "application/json")
	if err != nil || !resp.Success {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"error": resp.Error})
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func orderHandler(w http.ResponseWriter, r *http.Request) {
	userId, err := getUserIdFromToken(r)
	if err != nil { http.Error(w, err.Error(), 401); return }

	if r.Method == "GET" {
		oResp, err := orderClient.GetOrders(context.Background(), &orderpb.GetOrdersRequest{UserId: userId})
		if err != nil { http.Error(w, err.Error(), 500); return }
		w.Header().Set("Content-Type", "application/json")
		if oResp.Orders == nil {
			w.Write([]byte("[]"))
			return
		}
		json.NewEncoder(w).Encode(oResp.Orders)
		return
	} else if r.Method == "POST" {
		var req struct { ProductId string `json:"product_id"`; Quantity int32 `json:"quantity"` }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, "Bad Request", 400); return }

		oResp, err := orderClient.CreateOrder(context.Background(), &orderpb.CreateOrderRequest{
			UserId: userId, ProductId: req.ProductId, Quantity: req.Quantity})
		
		w.Header().Set("Content-Type", "application/json")
		if err != nil || !oResp.Success {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": oResp.Error})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"order_id": oResp.OrderId, "status": "CREATED"})
		return
	}
	http.Error(w, "Method Not Allowed", 405)
}

func inventoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		iResp, err := invClient.GetAllInventory(context.Background(), &invpb.GetAllInventoryRequest{})
		if err != nil { http.Error(w, err.Error(), 500); return }
		w.Header().Set("Content-Type", "application/json")
		if iResp.Items == nil {
			w.Write([]byte("[]"))
			return
		}
		json.NewEncoder(w).Encode(iResp.Items)
		return
	}
	http.Error(w, "Method Not Allowed", 405)
}

func inventoryItemHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" {
		// Extract product_id from /api/v1/inventory/{product_id}
		parts := strings.Split(r.URL.Path, "/")
		productId := parts[len(parts)-1]
		
		var req struct { Quantity int32 `json:"quantity"` }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil { http.Error(w, "Bad Request", 400); return }
		
		sResp, err := invClient.SetStock(context.Background(), &invpb.SetStockRequest{ProductId: productId, Quantity: req.Quantity})
		w.Header().Set("Content-Type", "application/json")
		if err != nil || !sResp.Success {
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]string{"error": sResp.Error})
			return
		}
		json.NewEncoder(w).Encode(map[string]bool{"success": true})
		return
	}
	http.Error(w, "Method Not Allowed", 405)
}

func main() {
	authHost := os.Getenv("AUTH_HOST")
	if authHost == "" { authHost = "auth-service:50051" }
	orderHost := os.Getenv("ORDER_HOST")
	if orderHost == "" { orderHost = "order-service:50053" }
	invHost := os.Getenv("INVENTORY_HOST")
	if invHost == "" { invHost = "inventory-service:50052" }
	notifHost := os.Getenv("NOTIFICATION_HOST")
	if notifHost == "" { notifHost = "notification-service:8080" }

	conn1, err := grpc.Dial(authHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { log.Fatal(err) }
	authClient = authpb.NewAuthServiceClient(conn1)

	conn2, err := grpc.Dial(orderHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { log.Fatal(err) }
	orderClient = orderpb.NewOrderServiceClient(conn2)

	conn3, err := grpc.Dial(invHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil { log.Fatal(err) }
	invClient = invpb.NewInventoryServiceClient(conn3)

	http.HandleFunc("/api/v1/auth/login", corsMiddleware(loginHandler))
	http.HandleFunc("/api/v1/auth/register", corsMiddleware(registerHandler))
	http.HandleFunc("/api/v1/orders", corsMiddleware(orderHandler))
	http.HandleFunc("/api/v1/inventory", corsMiddleware(inventoryHandler))
	http.HandleFunc("/api/v1/inventory/", corsMiddleware(inventoryItemHandler))

	// Proxy notifications to notification-service HTTP port
	notifURL, _ := url.Parse("http://" + notifHost)
	proxy := httputil.NewSingleHostReverseProxy(notifURL)
	http.HandleFunc("/api/v1/notifications", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = "/notifications" // Rewrite path for target
		proxy.ServeHTTP(w, r)
	}))

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	log.Println("API Gateway listening on :80")
	log.Fatal(http.ListenAndServe(":80", nil))
}
