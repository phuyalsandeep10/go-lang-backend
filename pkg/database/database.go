package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

var DB *sql.DB

type Config struct {
	User     string
	Password string
	Host     string
	Port     int
	DBName   string
}

func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on system environment variables")
	}

	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	host := os.Getenv("DB_HOST")
	portStr := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")

	if user == "" || password == "" || host == "" || portStr == "" || dbName == "" {
		return nil, fmt.Errorf("missing required environment variables")
	}

	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		return nil, fmt.Errorf("invalid DB_PORT format: %v", err)
	}

	return &Config{
		User:     user,
		Password: password,
		Host:     host,
		Port:     port,
		DBName:   dbName,
	}, nil
}

func InitDB() error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=30s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
	)
	log.Printf("Connecting to database with DSN: %s:%d/%s", cfg.Host, cfg.Port, cfg.DBName)

	DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	// Configure connection pooling
	DB.SetMaxOpenConns(100)           // Max simultaneous connections
	DB.SetMaxIdleConns(10)            // Max idle connections
	DB.SetConnMaxLifetime(5 * time.Minute) // Max lifetime of a connection

	if err = DB.Ping(); err != nil {
		DB.Close()
		return fmt.Errorf("failed to connect to database: %v", err)
	}

	log.Println("Database connected successfully.")
	return nil
}

func CloseDB() {
	if DB != nil {
		if err := DB.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		} else {
			log.Println("Database connection closed")
		}
	}
}
