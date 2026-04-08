package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/haroldcamargo/english/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type quickAddRequest struct {
	English  string `json:"english"`
	Spanish  string `json:"spanish"`
	Category string `json:"category"`
}

func (a *API) QuickAddVocab(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req quickAddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.English = strings.TrimSpace(req.English)
	req.Spanish = strings.TrimSpace(req.Spanish)
	req.Category = strings.TrimSpace(req.Category)

	if req.English == "" || req.Spanish == "" {
		writeError(w, http.StatusBadRequest, "english and spanish required")
		return
	}

	now := time.Now().UTC()
	v := models.Vocabulary{
		ID:           primitive.NewObjectID(),
		UserID:       userID,
		English:      req.English,
		Spanish:      req.Spanish,
		Category:     req.Category,
		Box:          1,
		NextReviewAt: now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	_, err := a.db.Collection("vocabularies").InsertOne(ctx, v)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save")
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (a *API) RecentVocab(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	limit := int64(20)
	cur, err := a.db.Collection("vocabularies").Find(ctx,
		bson.M{"userId": userID},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(limit),
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load")
		return
	}
	defer cur.Close(ctx)

	out := make([]models.Vocabulary, 0, limit)
	for cur.Next(ctx) {
		var v models.Vocabulary
		if err := cur.Decode(&v); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to decode")
			return
		}
		out = append(out, v)
	}
	writeJSON(w, http.StatusOK, out)
}
