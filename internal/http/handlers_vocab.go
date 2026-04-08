package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/haroldcamargo/english/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type quickAddRequest struct {
	English     string `json:"english"`
	Spanish     string `json:"spanish"`
	Description string `json:"description"`
	Category    string `json:"category"`
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
	req.Description = strings.TrimSpace(req.Description)
	req.Category = strings.TrimSpace(req.Category)

	if req.English == "" || req.Spanish == "" {
		writeError(w, http.StatusBadRequest, "english and spanish required")
		return
	}

	now := time.Now().UTC()
	active := true
	v := models.Vocabulary{
		ID:           primitive.NewObjectID(),
		UserID:       userID,
		English:      req.English,
		Spanish:      req.Spanish,
		Description:  req.Description,
		Category:     req.Category,
		Active:       &active,
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

type updateVocabRequest struct {
	English     string `json:"english"`
	Spanish     string `json:"spanish"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Active      *bool  `json:"active"`
}

func (a *API) UpdateVocab(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idHex := strings.TrimSpace(chi.URLParam(r, "id"))
	cardID, err := primitive.ObjectIDFromHex(idHex)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req updateVocabRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	req.English = strings.TrimSpace(req.English)
	req.Spanish = strings.TrimSpace(req.Spanish)
	req.Description = strings.TrimSpace(req.Description)
	req.Category = strings.TrimSpace(req.Category)

	if req.English == "" || req.Spanish == "" {
		writeError(w, http.StatusBadRequest, "english and spanish required")
		return
	}

	active := true
	if req.Active != nil {
		active = *req.Active
	}

	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"english":     req.English,
			"spanish":     req.Spanish,
			"description": req.Description,
			"category":    req.Category,
			"active":      active,
			"updatedAt":   now,
		},
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	res := a.db.Collection("vocabularies").FindOneAndUpdate(
		ctx,
		bson.M{"_id": cardID, "userId": userID},
		update,
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	)

	var out models.Vocabulary
	if err := res.Decode(&out); err != nil {
		writeError(w, http.StatusNotFound, "card not found")
		return
	}
	writeJSON(w, http.StatusOK, out)
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
