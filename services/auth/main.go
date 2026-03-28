package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"

	pb "k8s-microservices-app/proto/auth"
)

var jwtSecret = []byte("super-secret")

type server struct {
	pb.UnimplementedAuthServiceServer
	db *sql.DB
}

func (s *server) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return &pb.RegisterResponse{Success: false, Error: "Hash failed"}, nil
	}

	role := "consumer"
	if req.Username == "admin" { role = "admin" }

	_, err = s.db.Exec("INSERT INTO users (username, password, role) VALUES ($1, $2, $3)", req.Username, string(hashed), role)
	if err != nil {
		return &pb.RegisterResponse{Success: false, Error: "Username may be taken"}, nil
	}
	return &pb.RegisterResponse{Success: true}, nil
}

func (s *server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	var id int
	var role string
	var hash string
	err := s.db.QueryRow("SELECT id, role, password FROM users WHERE username=$1", req.Username).Scan(&id, &role, &hash)
	if err == sql.ErrNoRows {
		return &pb.LoginResponse{Error: "Invalid credentials"}, nil
	}
	if err != nil {
		log.Printf("DB Error during login: %v", err)
		return &pb.LoginResponse{Error: "DB Error"}, nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(req.Password))
	if err != nil {
		return &pb.LoginResponse{Error: "Invalid credentials"}, nil
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": req.Username,
		"user_id":  id,
		"role":     role,
		"exp":      time.Now().Add(time.Hour * 72).Unix(),
	})
	ts, _ := token.SignedString(jwtSecret)
	return &pb.LoginResponse{Token: ts}, nil
}

func (s *server) ValidateToken(ctx context.Context, req *pb.ValidateTokenRequest) (*pb.ValidateTokenResponse, error) {
	token, err := jwt.Parse(req.Token, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return &pb.ValidateTokenResponse{Valid: false}, nil
	}
	claims, _ := token.Claims.(jwt.MapClaims)
	userID := fmt.Sprintf("%v", claims["user_id"])
	return &pb.ValidateTokenResponse{Valid: true, UserId: userID}, nil
}

func main() {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "postgres"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "user"
	}
	dbPass := os.Getenv("DB_PASSWORD")
	if dbPass == "" {
		dbPass = "pass"
	}
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
	_, _ = db.Exec("CREATE TABLE IF NOT EXISTS users (id SERIAL PRIMARY KEY, username TEXT, password TEXT)")
	_, _ = db.Exec("ALTER TABLE users ADD COLUMN IF NOT EXISTS role TEXT DEFAULT 'consumer'")

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	pb.RegisterAuthServiceServer(s, &server{db: db})
	healthcheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthcheck)
	healthcheck.SetServingStatus("auth", grpc_health_v1.HealthCheckResponse_SERVING)

	log.Printf("Auth Service on :50051")
	log.Fatal(s.Serve(lis))
}
