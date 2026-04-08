package http

import (
	"context"
	"encoding/json"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/haroldcamargo/english/backend/internal/models"
	"github.com/haroldcamargo/english/backend/internal/srs"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type practiceMode string

type practiceDirection string

const (
	ModeAuto     practiceMode = "auto"
	ModeMultiple practiceMode = "multiple"
	ModeWrite    practiceMode = "write"
	ModeFill     practiceMode = "fill"

	DirAuto practiceDirection = "auto"
	DirEnEs practiceDirection = "en-es"
	DirEsEn practiceDirection = "es-en"
)

type practiceNextRequest struct {
	Mode      practiceMode      `json:"mode"`
	Direction practiceDirection `json:"direction"`
}

type practiceItem struct {
	CardID      string            `json:"cardId"`
	Mode        practiceMode      `json:"mode"`
	Direction   practiceDirection `json:"direction"`
	Prompt      string            `json:"prompt"`
	Description string            `json:"description,omitempty"`
	Masked      string            `json:"masked,omitempty"`
	Choices     []string          `json:"choices,omitempty"`
}

type practiceAnswerRequest struct {
	CardID    string            `json:"cardId"`
	Mode      practiceMode      `json:"mode"`
	Direction practiceDirection `json:"direction"`
	Response  string            `json:"response"`
}

type practiceAnswerResponse struct {
	Correct      bool      `json:"correct"`
	Box          int       `json:"box"`
	NextReviewAt time.Time `json:"nextReviewAt"`
	Expected     string    `json:"expected,omitempty"`
}

func autoDirectionForBox(box int) practiceDirection {
	if box <= 2 {
		if rand.Float64() < 0.8 {
			return DirEnEs
		}
		return DirEsEn
	}
	if rand.Float64() < 0.5 {
		return DirEnEs
	}
	return DirEsEn
}

func autoModeForBox(box int) practiceMode {
	if box <= 1 {
		return ModeMultiple
	}
	if box <= 3 {
		if rand.Float64() < 0.6 {
			return ModeMultiple
		}
		return ModeFill
	}
	if box <= 5 {
		if rand.Float64() < 0.5 {
			return ModeFill
		}
		return ModeWrite
	}
	return ModeWrite
}

func (a *API) PracticeNext(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req practiceNextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	now := time.Now().UTC()
	card, err := a.pickCard(ctx, userID, now)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			writeError(w, http.StatusNotFound, "no vocab")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to pick")
		return
	}

	mode := req.Mode
	if mode == "" || mode == ModeAuto {
		mode = autoModeForBox(card.Box)
	}

	direction := req.Direction
	if direction == "" || direction == DirAuto {
		direction = autoDirectionForBox(card.Box)
	}

	item := practiceItem{CardID: card.ID.Hex(), Mode: mode, Direction: direction}
	item.Description = strings.TrimSpace(card.Description)

	prompt, expected := buildPrompt(card, direction)
	item.Prompt = prompt

	if mode == ModeMultiple {
		choices, err := a.buildChoices(ctx, userID, card.ID, direction, expected)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to build choices")
			return
		}
		item.Choices = choices
	}
	if mode == ModeFill {
		item.Masked = maskWord(expected)
	}

	writeJSON(w, http.StatusOK, item)
}

func (a *API) PracticeAnswer(w http.ResponseWriter, r *http.Request) {
	userID, ok := UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req practiceAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	cardID, err := primitive.ObjectIDFromHex(strings.TrimSpace(req.CardID))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid cardId")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var card models.Vocabulary
	err = a.db.Collection("vocabularies").FindOne(ctx, bson.M{"_id": cardID, "userId": userID}).Decode(&card)
	if err != nil {
		writeError(w, http.StatusNotFound, "card not found")
		return
	}

	_, expected := buildPrompt(card, req.Direction)
	correct := isCorrect(req.Response, expected)

	now := time.Now().UTC()
	newBox := srs.NextBox(card.Box, correct)
	interval := srs.NextInterval(newBox, correct)
	nextAt := now.Add(interval)

	update := bson.M{
		"$set": bson.M{
			"box":          newBox,
			"nextReviewAt": nextAt,
			"updatedAt":    now,
		},
	}
	_, err = a.db.Collection("vocabularies").UpdateOne(ctx, bson.M{"_id": cardID, "userId": userID}, update)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update")
		return
	}

	resp := practiceAnswerResponse{Correct: correct, Box: newBox, NextReviewAt: nextAt}
	if !correct {
		resp.Expected = expected
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *API) pickCard(ctx context.Context, userID primitive.ObjectID, now time.Time) (models.Vocabulary, error) {
	activeFilter := bson.M{"$or": []bson.M{{"active": bson.M{"$exists": false}}, {"active": true}}}
	pipelineDue := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"userId": userID, "nextReviewAt": bson.M{"$lte": now}, "$or": activeFilter["$or"]}}},
		bson.D{{Key: "$sample", Value: bson.M{"size": 1}}},
	}
	cur, err := a.db.Collection("vocabularies").Aggregate(ctx, pipelineDue)
	if err != nil {
		return models.Vocabulary{}, err
	}
	defer cur.Close(ctx)
	if cur.Next(ctx) {
		var v models.Vocabulary
		if err := cur.Decode(&v); err != nil {
			return models.Vocabulary{}, err
		}
		return v, nil
	}

	var v models.Vocabulary
	err = a.db.Collection("vocabularies").FindOne(ctx,
		bson.M{"userId": userID, "$or": activeFilter["$or"]},
		options.FindOne().SetSort(bson.D{{Key: "nextReviewAt", Value: 1}}),
	).Decode(&v)
	return v, err
}

func (a *API) buildChoices(ctx context.Context, userID, excludeID primitive.ObjectID, dir practiceDirection, expected string) ([]string, error) {
	field := "spanish"
	if dir == DirEsEn {
		field = "english"
	}
	activeOr := []bson.M{{"active": bson.M{"$exists": false}}, {"active": true}}

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"userId": userID, "_id": bson.M{"$ne": excludeID}, "$or": activeOr}}},
		bson.D{{Key: "$sample", Value: bson.M{"size": 3}}},
		bson.D{{Key: "$project", Value: bson.M{field: 1}}},
	}
	cur, err := a.db.Collection("vocabularies").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	choices := []string{expected}
	for cur.Next(ctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		if v, ok := doc[field].(string); ok {
			v = strings.TrimSpace(v)
			if v != "" {
				choices = append(choices, v)
			}
		}
	}

	shuffleStrings(choices)
	return choices, nil
}

func buildPrompt(card models.Vocabulary, dir practiceDirection) (prompt string, expected string) {
	if dir == DirEsEn {
		return card.Spanish, card.English
	}
	return card.English, card.Spanish
}

func isCorrect(given, expected string) bool {
	g := strings.TrimSpace(strings.ToLower(given))
	e := strings.TrimSpace(strings.ToLower(expected))
	return g != "" && g == e
}

func shuffleStrings(s []string) {
	for i := len(s) - 1; i > 0; i-- {
		j := rand.IntN(i + 1)
		s[i], s[j] = s[j], s[i]
	}
}

func maskWord(word string) string {
	w := strings.TrimSpace(word)
	if w == "" {
		return ""
	}
	r := []rune(w)
	if len(r) <= 2 {
		return "_"
	}
	masked := make([]rune, len(r))
	copy(masked, r)
	toMask := len(r) / 2
	for k := 0; k < toMask; k++ {
		i := rand.IntN(len(r))
		if masked[i] != ' ' {
			masked[i] = '_'
		}
	}
	return string(masked)
}
