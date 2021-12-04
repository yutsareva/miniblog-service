package persistent

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
	"log"
	"miniblog/storage"
	"miniblog/storage/models"
	"time"
)

type Post struct {
	Id             primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	AuthorId       string             `bson:"authorId,omitempty" json:"authorId,omitempty"`
	Text           string             `bson:"text,omitempty" json:"text,omitempty"`
	CreatedAt      string             `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
	LastModifiedAt string             `bson:"lastModifiedAt,omitempty" json:"lastModifiedAt,omitempty"`
	Version        int64              `bson:"version,omitempty"`
}

type Subscription struct {
	Id             primitive.ObjectID `bson:"_id,omitempty"`
	userId       string             `bson:"authorId,omitempty"`
	SubscriptionId           string             `bson:"text,omitempty"`
}

func (p *Post) GetId() string {
	return p.Id.Hex()
}

func (p *Post) GetVersion() int64 {
	return p.Version
}

type MongoStorage struct {
	posts       *mongo.Collection
	subscriptions *mongo.Collection
	feed        *mongo.Collection
}

func (s *MongoStorage) Subscribe(ctx context.Context, userId string, subscriber string) error {
	subscription := Subscription{
		userId:       userId,
		SubscriptionId:           subscriber,
	}
	id, err := s.subscriptions.InsertOne(ctx, subscription)
	if err != nil {
		return fmt.Errorf("failed to insert subscription: %w", storage.InternalError)
	}
	log.Printf("Created subscription with id %s: %s -> %s", id, userId, subscriber)
	return nil
}

func (s *MongoStorage) GetSubscriptions(ctx context.Context, userId string) ([]string, error) {
	panic("implement me")
}

func (s *MongoStorage) GetSubscribers(ctx context.Context, userId string) ([]string, error) {
	panic("implement me")
}

func (s *MongoStorage) Feed(ctx context.Context, userId *string, page *string, size int) ([]models.Post, *string, error) {
	panic("implement me")
}

func (s *MongoStorage) PatchPost(ctx context.Context, postId string, userId string, text string) (models.Post, error) {
	var result Post
	postMongoId, err := primitive.ObjectIDFromHex(postId)
	if err != nil {
		return nil, fmt.Errorf("failed to convert provided id to Mongo object id %w", storage.NotFoundError)
	}
	filter := bson.M{"_id": postMongoId, "authorId": userId}
	update := bson.M{
		"$set": bson.M{
			"text":           text,
			"lastModifiedAt": time.Now().UTC().Format(time.RFC3339),
		},
		"$inc": bson.M{
			"version": 1,
		},
	}

	upsert := false
	after := options.After
	opt := options.FindOneAndUpdateOptions{
		ReturnDocument: &after,
		Upsert:         &upsert,
	}
	mongoResult := s.posts.FindOneAndUpdate(ctx, filter, update, &opt)
	err = mongoResult.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = s.posts.FindOne(ctx, bson.M{"_id": postMongoId}).Decode(&result)
			if err == nil {
				return nil, fmt.Errorf("post %s is owned by another user: %s %w", postId, result.AuthorId, storage.Forbidden)
			}
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, fmt.Errorf("no document with id %v: %w", postId, storage.NotFoundError)
			}
		}

		return nil, fmt.Errorf("failed to find post: %s %s %s %w", err.Error(), postMongoId, userId, storage.InternalError)
	}

	mongoResult.Decode(&result)
	return &result, nil
}

func (s *MongoStorage) GetPostsByUserId(
	ctx context.Context, userId *string, page *string, size int) ([]models.Post, *string, error) {

	options := options.Find()
	options.SetSort(bson.D{{"_id", -1}})
	options.SetLimit(int64(size + 1))

	minPage := "ffffffffffffffffffffffff"
	if page == nil {
		page = &minPage
	}
	pageMongoId, err := primitive.ObjectIDFromHex(*page)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert provided page to Mongo object id: %s, %w", err.Error(), storage.ClientError)
	}
	cursor, err := s.posts.Find(
		ctx,
		bson.D{
			{"authorId", *userId},
			{"_id", bson.D{{"$lte", pageMongoId}}},
		},
		options,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find posts by author: %s, %w", err.Error(), storage.InternalError)
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			log.Printf("Cursor closing failed: %s", err.Error())
		}
	}(cursor, ctx)

	posts := make([]models.Post, 0)
	var nextPage string
	for len(posts) != size+1 && cursor.Next(ctx) {
		var nextPost Post
		if err = cursor.Decode(&nextPost); err != nil {
			return nil, nil, fmt.Errorf("decode error: %s, %w", err, storage.InternalError)
		}
		if len(posts) == size {
			nextPage = nextPost.Id.Hex()
			return posts, &nextPage, nil
		}
		posts = append(posts, &nextPost)
	}
	if len(posts) == 0 && *page != minPage {
		return nil, nil, fmt.Errorf("provided page for non-existent user", storage.ClientError)
	}
	return posts, nil, nil
}

func (s *MongoStorage) AddPost(ctx context.Context, userId string, text string) (models.Post, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	post := Post{
		Text:           text,
		AuthorId:       userId,
		CreatedAt:      now,
		LastModifiedAt: now,
		Version:        0,
	}
	id, err := s.posts.InsertOne(ctx, post)
	if err != nil {
		return nil, fmt.Errorf("failed to insert post: %w", storage.InternalError)
	}
	post.Id = id.InsertedID.(primitive.ObjectID)
	return &post, nil
}

func (s *MongoStorage) GetPost(ctx context.Context, postId string) (models.Post, error) {
	var result Post
	postMongoId, err := primitive.ObjectIDFromHex(postId)
	if err != nil {
		return nil, fmt.Errorf("failed to convert provided id to Mongo object id %w", storage.NotFoundError)
	}
	err = s.posts.FindOne(ctx, bson.M{"_id": postMongoId}).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("no document with id %v: %w", postId, storage.NotFoundError)
		}
		return nil, fmt.Errorf("failed to find post: %s %w", err.Error(), storage.InternalError)
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
