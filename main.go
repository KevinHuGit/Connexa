package main

import (
	"context"
	"log"
	"os"
	"fmt"

	"finkedin/controllers"
	"finkedin/models"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	err := godotenv.Load()
	if err != nil {
			log.Fatalf("Error loading .env file: %v", err)
	}
	
	username := os.Getenv("MONGO_USER")
	password := os.Getenv("MONGO_PASS")

	mongoURI := fmt.Sprintf(
		"mongodb+srv://%s:%s@cluster0.22s2i.mongodb.net/",
		username, password,
	)

	err = models.ConnectDB(
		mongoURI,
		"mydb",
		"users",
	)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := models.Client.Disconnect(context.Background()); err != nil {
			log.Printf("Error disconnecting from MongoDB: %v", err)
		}
	}()

	router := gin.Default()
	
	router.Static("/static", "./static")
	router.Static("/uploads", "./uploads") // New static route for uploaded images

	router.LoadHTMLGlob("templates/*")

	// Define routes
	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "home.html", nil)
	})
	router.GET("/user/:user_id", controllers.ProfileHandler)
	router.POST("/user/:user_id", controllers.ChatbotHandler)


	// Serve the create profile page
	router.GET("/create_profile", func(c *gin.Context) {
		c.HTML(200, "create_profile.html", nil)
	})
	router.POST("/create_profile", controllers.CreateUser)

	// Serve the login page
	router.GET("/login", func(c *gin.Context) {
		c.HTML(200, "login.html", nil)
	})
	router.POST("/login", controllers.LoginHandler)

	// Serve the update profile page
	router.GET("/user/:user_id/update_profile", func(c *gin.Context) {
		userID := c.Param("user_id")

		c.HTML(200, "update_profile.html", gin.H{
				"UserID": userID,
		})
	})
	router.POST("/user/:user_id/update_profile", controllers.UpdateProfile)

	router.Run(":8080")
}
