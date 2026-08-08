package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"

	"freelance-btc/internal/auth"
	"freelance-btc/internal/bitcoin"
	"freelance-btc/internal/config"
	"freelance-btc/internal/order"
	"freelance-btc/internal/user"
)

func main() {
	cfg := config.Load()

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true",
		cfg.DBUser, cfg.DBPass, cfg.DBHost, cfg.DBPort, cfg.DBName)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("db open:", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal("db ping:", err)
	}

	rpc := bitcoin.NewClient(cfg.BtcRPCHost, cfg.BtcRPCPort, cfg.BtcRPCUser, cfg.BtcRPCPassword, cfg.WalletPassphrase)

	// health-check RPC при старте — аналог "RPC is connected OK." из вашего скрипта
	if hash, err := rpc.GetBestBlockHash(); err != nil {
		log.Println("WARNING: bitcoind RPC not reachable at startup:", err)
	} else {
		log.Println("bitcoind RPC OK, best block hash:", hash)
	}

	scanner := bitcoin.NewScanner(rpc, db, 60*time.Second, cfg.MinConfirmations)
	go scanner.Run()

	orderSvc := order.NewService(db, rpc)
	userSvc := user.NewService(db)
	jwtManager := auth.NewManager(cfg.JWTSecret, time.Duration(cfg.JWTTTLMin)*time.Minute)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api")
	{
		// ---- Публичные эндпоинты аутентификации ----
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", func(c *gin.Context) {
				var req struct {
					Email       string `json:"email" binding:"required"`
					Password    string `json:"password" binding:"required"`
					DisplayName string `json:"display_name" binding:"required"`
					Role        string `json:"role" binding:"required"` // "client" | "freelancer"
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				u, err := userSvc.Register(req.Email, req.Password, req.DisplayName, req.Role)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				token, err := jwtManager.GenerateToken(u.ID, u.Role)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusCreated, gin.H{"user": u, "token": token})
			})

			authGroup.POST("/login", func(c *gin.Context) {
				var req struct {
					Email    string `json:"email" binding:"required"`
					Password string `json:"password" binding:"required"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				u, err := userSvc.Authenticate(req.Email, req.Password)
				if err != nil {
					c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
					return
				}
				token, err := jwtManager.GenerateToken(u.ID, u.Role)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"user": u, "token": token})
			})
		}

		// ---- Защищённые эндпоинты (нужен Authorization: Bearer <token>) ----
		protected := api.Group("")
		protected.Use(jwtManager.Middleware())
		{
			protected.GET("/me", func(c *gin.Context) {
				userID := c.GetInt64("user_id")
				u, err := userSvc.GetByID(userID)
				if err != nil {
					c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, u)
			})

			protected.PUT("/me/payout-address", func(c *gin.Context) {
				userID := c.GetInt64("user_id")
				var req struct {
					Address string `json:"address" binding:"required"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				if err := userSvc.SetPayoutAddress(userID, req.Address); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			protected.POST("/orders", func(c *gin.Context) {
				clientID := c.GetInt64("user_id") // берём из токена, а не из тела запроса
				var req struct {
					Title       string  `json:"title" binding:"required"`
					Description string  `json:"description"`
					AmountBTC   float64 `json:"amount_btc" binding:"required"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				o, err := orderSvc.CreateOrder(clientID, req.Title, req.Description, req.AmountBTC)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusCreated, o)
			})

			protected.POST("/orders/:id/release", func(c *gin.Context) {
				var orderID int64
				if _, err := fmt.Sscanf(c.Param("id"), "%d", &orderID); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
					return
				}
				txid, err := orderSvc.ReleaseEscrow(orderID)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"txid": txid})
			})
		}
	}

	log.Println("listening on :" + cfg.HTTPPort)
	if err := r.Run(":" + cfg.HTTPPort); err != nil {
		log.Fatal(err)
	}
}
