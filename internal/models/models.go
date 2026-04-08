package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"passwordHash" json:"-"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
}

type Vocabulary struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID       primitive.ObjectID `bson:"userId" json:"userId"`
	English      string             `bson:"english" json:"english"`
	Spanish      string             `bson:"spanish" json:"spanish"`
	Category     string             `bson:"category,omitempty" json:"category,omitempty"`
	Box          int                `bson:"box" json:"box"`
	NextReviewAt time.Time          `bson:"nextReviewAt" json:"nextReviewAt"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`
}
