package db

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mg struct {
	*mongo.Client
}

func (m *mg) DB(name string) *db {
	return &db{m.Database(name)}
}

func (m *mg) CC(db, cc string) *cc {
	return m.DB(db).CC(cc)
}

type db struct {
	*mongo.Database
}

func (d *db) CC(name string) *cc {
	return &cc{d.Collection(name)}
}

type cc struct {
	*mongo.Collection
}

var MG mg

func Init(uri string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal(err)
	}
	MG = mg{client}
	return nil
}
