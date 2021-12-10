package persistent

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"miniblog/storage"
	"miniblog/storage/models"
	"os"
	"sync"
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
	userId         string             `bson:"authorId,omitempty"`
	SubscriptionId string             `bson:"text,omitempty"`
}

// TODO: rm json

type FeedItem struct {
	Id             primitive.ObjectID `bson:"_id,omitempty"`
	UserId         string             `bson:"userId,omitempty"`
	PostId         primitive.ObjectID `bson:"postId,omitempty" json:"id,omitempty"`
	AuthorId       string             `bson:"authorId,omitempty" json:"authorId,omitempty"`
	Text           string             `bson:"text,omitempty" json:"text,omitempty"`
	CreatedAt      string             `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
	LastModifiedAt string             `bson:"lastModifiedAt,omitempty" json:"lastModifiedAt,omitempty"`
}

func (p *Post) GetId() string {
	return p.Id.Hex()
}

func (p *Post) GetVersion() int64 {
	return p.Version
}

func (p *Post) GetAuthorId() string {
	return p.AuthorId
}

func (p *Post) GetText() string {
	return p.Text
}

func (p *Post) GetCreatedAt() string {
	return p.CreatedAt
}

func (p *Post) GetLastModifiedAt() string {
	return p.LastModifiedAt
}

type MongoStorage struct {
	posts         *mongo.Collection
	subscriptions *mongo.Collection
	feed          *mongo.Collection
}

func (s *MongoStorage) Subscribe(ctx context.Context, userId string, subscriber string) error {
	subscription := Subscription{
		userId:         userId,
		SubscriptionId: subscriber,
	}
	id, err := s.subscriptions.InsertOne(ctx, subscription)
	if err != nil {
		return fmt.Errorf("failed to insert subscription: %w", storage.InternalError)
	}
	log.Printf("Created subscription with id %s: %s -> %s", id, userId, subscriber)
	return nil
}

func (s *MongoStorage) GetSubscriptions(ctx context.Context, userId string) ([]string, error) {
	cursor, err := s.subscriptions.Find(
		ctx,
		bson.D{
			{"userId", userId},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find subscriptions for user: %s, %w", err.Error(), storage.InternalError)
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			log.Printf("Cursor closing failed: %s", err.Error())
		}
	}(cursor, ctx)

	var subscriptions []string
	for cursor.Next(ctx) {
		var nextSubscription Subscription
		if err = cursor.Decode(&nextSubscription); err != nil {
			return nil, fmt.Errorf("decode error: %s, %w", err, storage.InternalError)
		}
		subscriptions = append(subscriptions, nextSubscription.SubscriptionId)
	}
	return subscriptions, nil
}

func (s *MongoStorage) GetSubscribers(ctx context.Context, userId string) ([]string, error) {
	cursor, err := s.subscriptions.Find(
		ctx,
		bson.D{
			{"subscriptionId", userId},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to find subscribers for user: %s, %w", err.Error(), storage.InternalError)
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			log.Printf("Cursor closing failed: %s", err.Error())
		}
	}(cursor, ctx)

	var subscribers []string
	for cursor.Next(ctx) {
		var nextSubscription Subscription
		if err = cursor.Decode(&nextSubscription); err != nil {
			return nil, fmt.Errorf("decode error: %s, %w", err, storage.InternalError)
		}
		subscribers = append(subscribers, nextSubscription.userId)
	}
	return subscribers, nil
}

func (s *MongoStorage) Feed(ctx context.Context, userId *string, page *string, size int) ([]models.Post, *string, error) {
	queryOptions := options.Find()
	queryOptions.SetSort(bson.D{{"postId", -1}})
	queryOptions.SetLimit(int64(size + 1))

	minPage := "ffffffffffffffffffffffff"
	if page == nil {
		page = &minPage
	}
	pageMongoId, err := primitive.ObjectIDFromHex(*page)
	if err != nil {
		return nil, nil, fmt.Errorf("feed: failed to convert provided page to Mongo object id: %s, %w", err.Error(), storage.ClientError)
	}
	cursor, err := s.feed.Find(
		ctx,
		bson.D{
			{"authorId", *userId},
			{"postId", bson.D{{"$lte", pageMongoId}}},
		},
		queryOptions,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("feed: failed to find posts by author: %s, %w", err.Error(), storage.InternalError)
	}
	defer func(cursor *mongo.Cursor, ctx context.Context) {
		err := cursor.Close(ctx)
		if err != nil {
			log.Printf("feed: cursor closing failed: %s", err.Error())
		}
	}(cursor, ctx)

	posts := make([]models.Post, 0)
	var nextPage string
	for len(posts) != size+1 && cursor.Next(ctx) {
		var nextFeedItem FeedItem
		if err = cursor.Decode(&nextFeedItem); err != nil {
			return nil, nil, fmt.Errorf("decode error: %s, %w", err, storage.InternalError)
		}
		var nextPost = Post{
			Id:             nextFeedItem.PostId,
			AuthorId:       nextFeedItem.AuthorId,
			Text:           nextFeedItem.Text,
			CreatedAt:      nextFeedItem.CreatedAt,
			LastModifiedAt: nextFeedItem.LastModifiedAt,
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

	queryOptions := options.Find()
	queryOptions.SetSort(bson.D{{"_id", -1}})
	queryOptions.SetLimit(int64(size + 1))

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
		queryOptions,
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

func (s *MongoStorage) UpdateFeed(ctx context.Context, userId string, posts []models.Post) error {
	var feedItems []interface{}
	for _, post := range posts {
		postId, _ := primitive.ObjectIDFromHex(post.GetId())
		feedItem := FeedItem{
			UserId:         userId,
			PostId:         postId,
			Text:           post.GetText(),
			AuthorId:       post.GetAuthorId(),
			CreatedAt:      post.GetCreatedAt(),
			LastModifiedAt: post.GetLastModifiedAt(),
		}
		feedItems = append(feedItems, feedItem)
	}

	_, err := s.feed.InsertMany(ctx, feedItems)
	if err != nil {
		return fmt.Errorf("failed to insert post: %w", storage.InternalError)
	}
	return nil
}

var mongoStorage *MongoStorage
var once sync.Once

func CreateMongoStorage(dbUrl, dbName string) *MongoStorage {
	// make singleton
	once.Do(func() {
		ctx := context.Background()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbUrl))
		if err != nil {
			panic(err)
		}
		posts := client.Database(dbName).Collection("posts")
		subscriptions := client.Database(dbName).Collection("subscriptions")
		feed := client.Database(dbName).Collection("feed")
		ensureIndexes(ctx, posts)
		mongoStorage = &MongoStorage{
			posts:         posts,
			subscriptions: subscriptions,
			feed:          feed,
		}
	})
	return mongoStorage
}

func GetMongoStorage() *MongoStorage {
	once.Do(func() {
		mongoUrl, found := os.LookupEnv("MONGO_URL")
		if !found {
			panic("'MONGO_URL' not specified")
		}
		mongoDbName, found := os.LookupEnv("MONGO_DBNAME")
		if !found {
			panic("'MONGO_DBNAME' not specified")
		}
		mongoStorage = CreateMongoStorage(mongoUrl, mongoDbName)
	})
	return mongoStorage
}
