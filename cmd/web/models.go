package main

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

type Mongo struct {
	ID primitive.ObjectID `bson:"_id"`
}

func (m *Mongo) GetMongoCollectionName() string {
	panic("GetMongoCollectionName not implemented")
	return ""
}

type Post struct {
	Mongo   `inline`
	Title   string `bson:"title"`
	Author    string `bson:"author"`
	Content string `bson:"content"`
}

func (p *Post) GetMongoCollectionName() string {
	return "posts"
}

func (p *Post) Insert(ctx context.Context, db *mongo.Database) error {
	p.ID = primitive.NewObjectID()
	coll := db.Collection(p.GetMongoCollectionName())
	_, err := coll.InsertOne(ctx, p)
	if err != nil {
		return err
	}
	return nil
}

func GetPost(ctx context.Context, db *mongo.Database, id primitive.ObjectID) (*Post, error) {
	var p Post
	coll := db.Collection(p.GetMongoCollectionName())
	res := coll.FindOne(ctx, bson.M{"_id": id})
	if err := res.Decode(&p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (p *Post) Update(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(p.GetMongoCollectionName())

	opts := options.Update().SetUpsert(true)
	filter := bson.D{{"_id", p.ID}}
	update := bson.D{{"$set", bson.D{{"title", p.Title},{"content", p.Content}, {"author",p.Author}}}}
	result, err := coll.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		log.Fatal(err)
	}

	if result.MatchedCount != 0 {
		fmt.Println("matched and replaced an existing document")
		return nil
	}

	return err
}

func (p *Post) Delete(ctx context.Context, db *mongo.Database) error {
	coll := db.Collection(p.GetMongoCollectionName())
	_, err := coll.DeleteOne(ctx, bson.M{"_id": p.ID})
	return err
}

func (hd *HandlerData) GetPosts() ([]Post, error) {
	p := Post{}
	coll := (*hd).db.Collection(p.GetMongoCollectionName())

	cur, err := coll.Find((*hd).ctx, bson.M{})
	if err != nil {
		return nil, err
	}

	var posts []Post
	if err = cur.All((*hd).ctx, &posts); err != nil {
		return nil, err
	}

	return posts, nil
}

func (hd *HandlerData) Find(field string, value interface{}) ([]Post, error) {
	p := Post{}
	coll := (*hd).db.Collection(p.GetMongoCollectionName())

	cur, err := coll.Find((*hd).ctx, bson.M{field: value})
	if err != nil {
		return nil, err
	}

	var posts []Post
	if err = cur.All(hd.ctx, &posts); err != nil {
		return nil, err
	}

	return posts, nil
}
