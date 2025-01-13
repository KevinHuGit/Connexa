package controllers

import (
	// "bytes"
	"context"
	"encoding/base64"
	// "image"
	// "image/jpeg"
	// "encoding/json"
	"fmt"
	"log"
	"io"
	"net/http"
	"time"
	"strings"
	"html/template"



	"finkedin/models"
	"github.com/gin-gonic/gin"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
	
)

func CreateUser(c *gin.Context) {
	var newUser models.User
	if err := c.ShouldBindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newUser.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	newUser.Password = string(hashedPassword)
	newUser.ID = primitive.NewObjectID()
	newUser.CreatedAt = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := models.Collection.InsertOne(ctx, newUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	insertedID := result.InsertedID.(primitive.ObjectID)
	c.JSON(http.StatusOK, gin.H{
		"message": "Profile created successfully",
		"userId":  insertedID.Hex(),
	})
}

func ProfileHandler(c *gin.Context) {
	userID := c.Param("user_id")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user models.User
	err = models.Collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user"})
		}
		return
	}

// Handle the profile picture
var profilePicture template.URL
if user.ProfilePicture != "" {
	// Check if the string already contains the data URI prefix
	if !strings.HasPrefix(user.ProfilePicture, "data:image") {
		// Detect the image type (you might want to store this in the database)
		// For now, we'll assume it's a PNG
		profilePicture = template.URL("data:image/jpeg;base64," + user.ProfilePicture)
	} else {
		// If it already has the prefix, use it as is
		profilePicture = template.URL(user.ProfilePicture)
	}
} else {
	// Default profile picture
	profilePicture = template.URL("/static/default-profile.png")
}
	

	profile := models.Profile{
		Username:      user.Username,
		ProfilePicture: profilePicture,
		Location:      user.Location,
		About:      user.About,
		Education:      user.Education,
		UserID:        user.ID.Hex(),
	}

	c.HTML(http.StatusOK, "profile.html", profile)
}

func LoginHandler(c *gin.Context) {
	var loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var user models.User
	err := models.Collection.FindOne(ctx, bson.M{"email": loginData.Username}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginData.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user_id": user.ID.Hex(),
	})
}

func UpdateProfile(c *gin.Context) {
	// Extract the user ID from the URL parameter
	userID := c.Param("user_id")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Parse form data to handle both JSON and file uploads
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // Limit to 10 MB
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse form data"})
		return
	}

	// Extract text fields from the form
	username := c.Request.FormValue("username")
	location := c.Request.FormValue("location")
	about := c.Request.FormValue("about")


	// Prepare update fields
	updateFields := bson.M{}
	if username != "" {
		updateFields["username"] = username
	}
	if location != "" {
		updateFields["location"] = location
	}
	if about != "" {
		updateFields["about"] = about
	}


	// Handle profile picture upload
	file, _, err := c.Request.FormFile("profile_picture")
	if err == nil && file != nil {
		defer file.Close()

		// Read the file content
		fileBytes, err := io.ReadAll(file)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read profile picture"})
			return
		}

		// Convert file content to base64 string
		profilePictureBase64 := base64.StdEncoding.EncodeToString(fileBytes)

		// Add the base64-encoded profile picture to the update fields
		updateFields["profile_picture"] = profilePictureBase64
	}

	// Update the database with the collected fields
	if len(updateFields) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		result, err := models.Collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": updateFields})
		if err != nil || result.MatchedCount == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile created successfully",
		"userId": userID,
	})}

// ChatbotHandler handles chatbot interactions using LangChainGo with Ollama
func ChatbotHandler(c *gin.Context) {
	// Extract user_id from the route
	userID := c.Param("user_id")
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var user models.User
	err = models.Collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching user"})
		}
		return
	}

	// Parse the user's message from the request body
	var requestBody struct {
		Message string `json:"message"`
	}
	if err := c.BindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	// Build context from user profile
	userContext := fmt.Sprintf("You are acting as %s, a student at %s living in %s. You have skills in %s and experience as %s. Be polite and helpful.",
		user.Username, user.Education, user.Location, user.Skills, user.Experience)

	// Initialize the Ollama LLM
	llm, err := ollama.New(ollama.WithModel("phi3:mini"))
	if err != nil {
		log.Printf("Failed to initialize Ollama LLM: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize chatbot"})
		return
	}

	// Prepare the input for the chatbot
	input := fmt.Sprintf("%s\n\nHuman: %s\nAssistant:", userContext, requestBody.Message)

	// Call the chatbot and get the response
	response, err := llm.Call(ctx, input, llms.WithTemperature(0.8))
	if err != nil {
		log.Printf("Error during chatbot interaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate chatbot response"})
		return
	}

	// Respond to the client with the chatbot's reply
	c.JSON(http.StatusOK, gin.H{
		"user_id": userID,
		"reply":   response,
	})
}