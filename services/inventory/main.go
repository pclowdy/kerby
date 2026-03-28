package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	pb "k8s-microservices-app/proto/inventory"
)

type server struct {
	pb.UnimplementedInventoryServiceServer
	db *sql.DB
}

func (s *server) CheckStock(ctx context.Context, req *pb.CheckStockRequest) (*pb.CheckStockResponse, error) {
	var qty int32
	err := s.db.QueryRow("SELECT quantity FROM inventory WHERE product_id=$1", req.ProductId).Scan(&qty)
	if err == sql.ErrNoRows {
		// Auto-create missing products for demo
		err = s.db.QueryRow("INSERT INTO inventory (product_id, quantity) VALUES ($1, $2) RETURNING quantity", req.ProductId, 10).Scan(&qty)
	}
	if err != nil {
		return &pb.CheckStockResponse{Available: false}, nil
	}
	return &pb.CheckStockResponse{Available: qty >= req.Quantity}, nil
}

func (s *server) DeductStock(ctx context.Context, req *pb.DeductStockRequest) (*pb.DeductStockResponse, error) {
	var newQty int32
	err := s.db.QueryRow("UPDATE inventory SET quantity = quantity - $1 WHERE product_id = $2 AND quantity >= $1 RETURNING quantity", req.Quantity, req.ProductId).Scan(&newQty)
	if err != nil {
		if err == sql.ErrNoRows {
			return &pb.DeductStockResponse{Success: false, Error: "Product not found or insufficient stock"}, nil
		}
		return &pb.DeductStockResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.DeductStockResponse{Success: true, RemainingQuantity: newQty}, nil
}

func (s *server) GetAllInventory(ctx context.Context, req *pb.GetAllInventoryRequest) (*pb.GetAllInventoryResponse, error) {
	rows, err := s.db.Query("SELECT product_id, quantity FROM inventory")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []*pb.InventoryItem
	for rows.Next() {
		var item pb.InventoryItem
		if err := rows.Scan(&item.ProductId, &item.Quantity); err == nil {
			items = append(items, &item)
		}
	}
	return &pb.GetAllInventoryResponse{Items: items}, nil
}

func (s *server) SetStock(ctx context.Context, req *pb.SetStockRequest) (*pb.SetStockResponse, error) {
	// Upsert query for Postgres
	_, err := s.db.Exec("INSERT INTO inventory (product_id, quantity) VALUES ($1, $2) ON CONFLICT (product_id) DO UPDATE SET quantity = EXCLUDED.quantity", req.ProductId, req.Quantity)
	if err != nil {
		return &pb.SetStockResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.SetStockResponse{Success: true}, nil
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

	_, _ = db.Exec("CREATE TABLE IF NOT EXISTS inventory (id SERIAL PRIMARY KEY, product_id VARCHAR(50) UNIQUE, quantity INT)")

	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterInventoryServiceServer(s, &server{db: db})

	healthcheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthcheck)
	healthcheck.SetServingStatus("inventory", grpc_health_v1.HealthCheckResponse_SERVING)

	log.Printf("Inventory Service on :50052")
	log.Fatal(s.Serve(lis))
}
