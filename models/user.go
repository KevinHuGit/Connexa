package models

import (
    "context"
    "html/template"
    "log"
    "time"

    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

// Job represents a single job entry in a user's experience
type Job struct {
    Title        string    `bson:"title" json:"title"`
    Company      string    `bson:"company" json:"company"`
    Location     string    `bson:"location" json:"location"`
    StartDate    time.Time `bson:"start_date" json:"start_date"`
    EndDate      *time.Time `bson:"end_date,omitempty" json:"end_date,omitempty"` // Pointer to handle current jobs
    Description  string    `bson:"description" json:"description"`
    IsCurrentJob bool      `bson:"is_current_job" json:"is_current_job"`
}

// User structure with updated Experience field
type User struct {
    ID             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
    Username       string            `bson:"username" json:"username"`
    Email          string            `bson:"email" json:"email"`
    Password       string            `bson:"password" json:"password"`
    ProfilePicture string            `bson:"profile_picture" json:"profile_picture"`
    Location       string            `bson:"location" json:"location"`
    Experience     []Job             `bson:"experience" json:"experience"` // Changed to array of Job
    About          string            `bson:"about" json:"about"`
    Education      string            `bson:"education" json:"education"`
    Skills         string            `bson:"skills" json:"skills"`
    CreatedAt      time.Time         `bson:"created_at" json:"created_at"`
}

type Profile struct {
    Username       string       `json:"username"`
    ProfilePicture template.URL `json:"profile_picture"`
    Location       string       `json:"location"`
    About          string       `json:"about"`
    Education      string       `json:"education"`
    UserID         string       `json:"user_id"`
}

var (
    Client     *mongo.Client
    Collection *mongo.Collection
)

func ConnectDB(uri, dbName, collectionName string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
    defer cancel()

    clientOptions := options.Client().ApplyURI(uri)
    var err error
    Client, err = mongo.Connect(ctx, clientOptions)
    if err != nil {
        return err	
    }

    err = Client.Ping(ctx, nil)
    if err != nil {
        return err
    }

    Collection = Client.Database(dbName).Collection(collectionName)
    log.Println("Connected to MongoDB!")
    return nil
}