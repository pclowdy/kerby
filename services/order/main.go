package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	inv "k8s-microservices-app/proto/inventory"
	pb "k8s-microservices-app/proto/order"
)

type server struct {
	pb.UnimplementedOrderServiceServer
	db    *sql.DB
	rdb   *redis.Client
	invCl inv.InventoryServiceClient
}

func (s *server) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	log.Printf("Creating order for user %s, product %s", req.UserId, req.ProductId)

	// 1. Check Inventory
	invResp, err := s.invCl.CheckStock(ctx, &inv.CheckStockRequest{ProductId: req.ProductId, Quantity: req.Quantity})
	if err != nil || !invResp.Available {
		return &pb.CreateOrderResponse{Success: false, Error: "Insufficient stock"}, nil
	}

	// 2. Insert into DB
	orderId := uuid.New().String()
	_, err = s.db.Exec("INSERT INTO orders (id, user_id, product_id, quantity, status) VALUES ($1, $2, $3, $4, $5)", orderId, req.UserId, req.ProductId, req.Quantity, "CREATED")
	if err != nil {
		return &pb.CreateOrderResponse{Success: false, Error: "Failed to create order"}, nil
	}

	// 3. Update Inventory
	dResp, err := s.invCl.DeductStock(ctx, &inv.DeductStockRequest{ProductId: req.ProductId, Quantity: req.Quantity})
	if err != nil || !dResp.Success {
		return &pb.CreateOrderResponse{Success: false, Error: "Failed to deduct stock"}, nil
	}

	// 4. Redis Pub/Sub Event
	msg := fmt.Sprintf(`{"type":"ORDER_PLACED", "order_id":"%s", "user_id":"%s", "product_id":"%s", "status":"CREATED"}`, orderId, req.UserId, req.ProductId)
	s.rdb.Publish(ctx, "orders", msg)

	if dResp.RemainingQuantity < 5 {
		msgLow := fmt.Sprintf(`{"type":"LOW_STOCK", "product_id":"%s", "remaining":%d}`, req.ProductId, dResp.RemainingQuantity)
		s.rdb.Publish(ctx, "orders", msgLow)
	}

	return &pb.CreateOrderResponse{Success: true, OrderId: orderId}, nil
}

func (s *server) GetOrders(ctx context.Context, req *pb.GetOrdersRequest) (*pb.GetOrdersResponse, error) {
	rows, err := s.db.Query("SELECT id, product_id, quantity, status FROM orders WHERE user_id=$1", req.UserId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*pb.OrderItem
	for rows.Next() {
		var o pb.OrderItem
		if err := rows.Scan(&o.OrderId, &o.ProductId, &o.Quantity, &o.Status); err == nil {
			orders = append(orders, &o)
		}
	}
	return &pb.GetOrdersResponse{Orders: orders}, nil
}



func main() {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" { dbHost = "postgres" }
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" { dbUser = "user" }
	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" { dbPass = "pass" }

	connStr := fmt.Sprintf("host=%s port=5432 user=%s password=%s dbname=postgres sslmode=disable", dbHost, dbUser, dbPass)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < 5; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}

	_, _ = db.Exec("CREATE TABLE IF NOT EXISTS orders (id VARCHAR(50) PRIMARY KEY, user_id VARCHAR(50), product_id VARCHAR(50), quantity INT, status VARCHAR(20))")

	// Redis Config
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" { redisHost = "redis:6379" }
	rdb := redis.NewClient(&redis.Options{Addr: redisHost})

	// Inventory Client connection
	invHost := os.Getenv("INVENTORY_HOST")
	if invHost == "" { invHost = "inventory-service:50052" }
	conn, err := grpc.Dial(invHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	invCl := inv.NewInventoryServiceClient(conn)

	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterOrderServiceServer(s, &server{db: db, rdb: rdb, invCl: invCl})

	healthcheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthcheck)
	healthcheck.SetServingStatus("order", grpc_health_v1.HealthCheckResponse_SERVING)

	log.Printf("Order Service on :50053")
	log.Fatal(s.Serve(lis))
}
