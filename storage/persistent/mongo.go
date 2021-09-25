package persistent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"time"
	//"github.com/google/uuid"
	"miniblog/storage"
	"miniblog/storage/models"
	//"time"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
)

type Post struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	AuthorId  string             `bson:"authorId,omitempty" json:"authorId,omitempty"`
	Text      string             `bson:"text,omitempty" json:"text,omitempty"`
	CreatedAt string             `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
}

func (p *Post) ToJson() []byte {
	j, err := json.Marshal(p)
	if err != nil {
		log.Fatalf("Failed to dump post to json: %s", err.Error())
	}
	return j
}

type MongoStorage struct {
	posts *mongo.Collection
}

func (s *MongoStorage) GetPostsByUserId(
		ctx context.Context, userId *string, page *string, size int) ([]models.Post, *string, error) {

	options := options.Find()
	options.SetSort(bson.D{{"_id", -1}})
	options.SetLimit(size + 1)
	cursor, err := s.posts.Find(ctx, bson.M{{"authorId": *userId}, {"_id", bson.D{{"$lte", *page}}}} options)
	return nil, nil, nil
}

func (s *MongoStorage) AddPost(ctx context.Context, userId *string, text *string) (models.Post, error) {


	return nil, nil
}

func (s *MongoStorage) GetPost(ctx context.Context, postId *string) (models.Post, error) {
	var result Post
	err := s.posts.FindOne(ctx, bson.M{"_id": postId}).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("no document with id %v - %w", postId, storage.NotFoundError)
		}
		return nil, fmt.Errorf("somehting went wroing - %w", storage.InternalError)
	}
	return &result, nil
}

func CreateMongoStorage(dbUrl, dbName string) storage.Storage {
	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbUrl))
	if err != nil {
		panic(err)
	}

	posts := client.Database(dbName).Collection("posts")

	ensureIndexes(ctx, posts)

	return &MongoStorage{
		posts: posts,
	}
}

func ensureIndexes(ctx context.Context, posts *mongo.Collection) {
	indexModels := []mongo.IndexModel{
		{
			Keys: bsonx.Doc{
				{Key: "author_id", Value: bsonx.Int32(1)},
				{Key: "_id", Value: bsonx.Int32(1)},
			},
		},
	}
	opts := options.CreateIndexes().SetMaxTime(10 * time.Second)

	_, err := posts.Indexes().CreateMany(ctx, indexModels, opts)
	if err != nil {
		panic(fmt.Errorf("failed to ensure indexes %w", err))
	}
}
