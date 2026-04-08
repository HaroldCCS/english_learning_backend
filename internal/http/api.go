package http

import (
	"github.com/haroldcamargo/english/backend/internal/config"
	"go.mongodb.org/mongo-driver/mongo"
)

type API struct {
	cfg config.Config
	db  *mongo.Database
}

func NewAPI(cfg config.Config, database *mongo.Database) *API {
	return &API{cfg: cfg, db: database}
}
