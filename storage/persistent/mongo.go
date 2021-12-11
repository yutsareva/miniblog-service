package persistent

import (
	"context"
	"errors"
	"fmt"
	"github.com/RichardKnop/machinery/v1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"miniblog/storage"
	"miniblog/storage/models"
	"miniblog/utils"
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
	SubscriptionId string             `bson:"subscriptionId,omitempty"`
	UserId         string             `bson:"userId,omitempty"`
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

type MongoStorageWithBroker struct {
	mongo  *MongoStorage
	broker *machinery.Server
}

func (s *MongoStorageWithBroker) Subscribe(ctx context.Context, userId string, subscriber string) error {
	subscription := Subscription{
		UserId:         subscriber,
		SubscriptionId: userId,
	}
	upsertSubscribtion := bson.D{{"$set", subscription}}
	queryOptions := options.Update().SetUpsert(true)
	id, err := s.mongo.subscriptions.UpdateOne(ctx, subscription, upsertSubscribtion, queryOptions)
	if err != nil {
		return fmt.Errorf("failed to insert subscription: %w", storage.InternalError)
	}
	log.Printf("Created subscription with id %s: %s -> %s", id, subscriber, userId)

	task := createAddSubscriptionTask(userId, subscriber)
	_, err = s.broker.SendTaskWithContext(context.Background(), &task)
	if err != nil {
		return fmt.Errorf("could not send task: %s %w", err.Error(), storage.InternalError)
	}
	//results, err := asyncResult.Get(time.Duration(1 * time.Second))
	//if err != nil {
	//	return fmt.Errorf("getting task result failed with error: %s", err.Error())
	//}
	//log.Printf("%v\n", tasks.HumanReadableResults(results))
	return nil
}

func (s *MongoStorageWithBroker) GetSubscriptions(ctx context.Context, userId string) ([]string, error) {
	cursor, err := s.mongo.subscriptions.Find(
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

	subscriptions := make([]string, 0)
	for cursor.Next(ctx) {
		var nextSubscription Subscription
		if err = cursor.Decode(&nextSubscription); err != nil {
			return nil, fmt.Errorf("decode error: %s, %w", err, storage.InternalError)
		}
		subscriptions = append(subscriptions, nextSubscription.SubscriptionId)
	}
	return subscriptions, nil
}

func (s *MongoStorageWithBroker) GetSubscribers(ctx context.Context, userId string) ([]string, error) {
	cursor, err := s.mongo.subscriptions.Find(
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

	subscribers := make([]string, 0)
	for cursor.Next(ctx) {
		var nextSubscription Subscription
		if err = cursor.Decode(&nextSubscription); err != nil {
			return nil, fmt.Errorf("decode error: %s, %w", err, storage.InternalError)
		}
		subscribers = append(subscribers, nextSubscription.UserId)
	}
	return subscribers, nil
}

func (s *MongoStorageWithBroker) Feed(ctx context.Context, userId *string, page *string, size int) ([]models.Post, *string, error) {
	queryOptions := options.Find()
	queryOptions.SetSort(bson.D{{"postId", -1}})
	queryOptions.SetLimit(int64(size + 1))

	minPage := "ffffffffffffffffffffffff"
	if page == nil {
		page = &minPage
	}
	pageMongoId, err := primitive.ObjectIDFromHex(*page)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert provided page to Mongo object id: %s, %w", err.Error(), storage.ClientError)
	}
	cursor, err := s.mongo.feed.Find(
		ctx,
		bson.D{
			{"userId", userId},
			{"postId", bson.D{{"$lte", pageMongoId}}},
		},
		queryOptions,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("feed: failed to find posts for user: %s, %w", err.Error(), storage.InternalError)
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

func (s *MongoStorageWithBroker) PatchPost(ctx context.Context, postId string, userId string, text string) (models.Post, error) {
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
	mongoResult := s.mongo.posts.FindOneAndUpdate(ctx, filter, update, &opt)
	err = mongoResult.Err()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = s.mongo.posts.FindOne(ctx, bson.M{"_id": postMongoId}).Decode(&result)
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

	task := createPatchPostTask(result.Id)
	_, err = s.broker.SendTaskWithContext(context.Background(), &task)
	if err != nil {
		return nil, fmt.Errorf("could not send task: %s %w", err.Error(), storage.InternalError)
	}
	return &result, nil
}

func (s *MongoStorageWithBroker) GetPostsByUserId(
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
	cursor, err := s.mongo.posts.Find(
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

func (s *MongoStorageWithBroker) AddPost(ctx context.Context, userId string, text string) (models.Post, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	post := Post{
		Text:           text,
		AuthorId:       userId,
		CreatedAt:      now,
		LastModifiedAt: now,
		Version:        0,
	}
	id, err := s.mongo.posts.InsertOne(ctx, post)
	if err != nil {
		return nil, fmt.Errorf("failed to insert post: %s %w", err.Error(), storage.InternalError)
	}
	post.Id = id.InsertedID.(primitive.ObjectID)

	task := createAddPostTask(post.Id, post.AuthorId)
	_, err = s.broker.SendTaskWithContext(context.Background(), &task)
	if err != nil {
		return nil, fmt.Errorf("could not send task: %s %w", err.Error(), storage.InternalError)
	}
	//results, err := asyncResult.Get(time.Duration(1 * time.Second))
	//if err != nil {
	//	return fmt.Errorf("getting task result failed with error: %s", err.Error())
	//}
	//log.Printf("%v\n", tasks.HumanReadableResults(results))
	return &post, nil
}

func (s *MongoStorageWithBroker) GetPost(ctx context.Context, postId string) (models.Post, error) {
	var result Post
	postMongoId, err := primitive.ObjectIDFromHex(postId)
	if err != nil {
		return nil, fmt.Errorf("failed to convert provided id to Mongo object id %w", storage.NotFoundError)
	}
	err = s.mongo.posts.FindOne(ctx, bson.M{"_id": postMongoId}).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("no document with id %v: %w", postId, storage.NotFoundError)
		}
		return nil, fmt.Errorf("failed to find post: %s %w", err.Error(), storage.InternalError)
	}
	return &result, nil
}

func (s *MongoStorageWithBroker) UpdateFeedNewSubscription(ctx context.Context, userId string, posts []models.Post) error {
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
	if len(feedItems) == 0 {
		log.Printf("Update feed: nothing to insert")
		return nil
	}
	// TODO: upsert instead of insert
	_, err := s.mongo.feed.InsertMany(ctx, feedItems)
	if err != nil {
		return fmt.Errorf("failed to insert post: %s %w", err.Error(), storage.InternalError)
	}
	return nil
}

func (s *MongoStorageWithBroker) UpdateFeedNewPost(ctx context.Context, postId string, subscribers []string) (int, error) {
	post, err := s.GetPost(ctx, postId)
	if err != nil {
		return 0, fmt.Errorf("update feed: failed to get post by id: %s %w", err.Error(), storage.InternalError)
	}

	var feedItems []interface{}
	for _, subscriber := range subscribers {
		postId, _ := primitive.ObjectIDFromHex(post.GetId())
		feedItem := FeedItem{
			UserId:         subscriber,
			PostId:         postId,
			Text:           post.GetText(),
			AuthorId:       post.GetAuthorId(),
			CreatedAt:      post.GetCreatedAt(),
			LastModifiedAt: post.GetLastModifiedAt(),
		}
		feedItems = append(feedItems, feedItem)
	}
	if len(feedItems) == 0 {
		log.Printf("Update feed: nothing to insert")
		return 0, nil
	}
	// TODO: upsert instead of insert
	ids, err := s.mongo.feed.InsertMany(ctx, feedItems)
	log.Printf("update feed - added post: Inserted %s feedItems", ids)
	if err != nil {
		return 0, fmt.Errorf("failed to insert post: %s %w", err.Error(), storage.InternalError)
	}
	return len(feedItems), nil
}

func (s *MongoStorageWithBroker) UpdateFeedPatchPost(ctx context.Context, postId string) (int, error) {
	post, err := s.GetPost(ctx, postId)
	if err != nil {
		return 0, fmt.Errorf("update feed: failed to get post by id: %s %w", err.Error(), storage.InternalError)
	}
	postObjId, _ := primitive.ObjectIDFromHex(postId)
	filter := bson.M{"postId": postObjId}
	updateInfo := bson.M{
		"$set": bson.M{
			"text":           post.GetText(),
			"lastModifiedAt": post.GetLastModifiedAt(),
		},
	}
	ids, err := s.mongo.feed.UpdateMany(ctx, filter, updateInfo)
	log.Printf("update feed - patched post: Updated %d feedItems: %s", ids.ModifiedCount, ids)
	if err != nil {
		return 0, fmt.Errorf("failed to patch post: %s %w", err.Error(), storage.InternalError)
	}
	return int(ids.ModifiedCount), nil
}

var mongoStorage *MongoStorage
var mongoStorageWithoutBroker *MongoStorageWithBroker
var onceMongo sync.Once
var onceStorage sync.Once

func CreateMongoStorage(dbUrl, dbName string) *MongoStorage {
	// make singleton
	onceMongo.Do(func() {
		ctx := context.Background()
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbUrl))
		if err != nil {
			panic(err)
		}
		posts := client.Database(dbName).Collection("posts")
		subscriptions := client.Database(dbName).Collection("subscriptions")
		feed := client.Database(dbName).Collection("feed")
		ensurePostsIndexes(ctx, posts)
		ensureFeedIndexes(ctx, feed)
		ensureSubscriptionsIndexes(ctx, subscriptions)
		mongoStorage = &MongoStorage{
			posts:         posts,
			subscriptions: subscriptions,
			feed:          feed,
		}
	})
	return mongoStorage
}

func CreateMongoStorageWithBroker(dbUrl, dbName, brokerUrl string) *MongoStorageWithBroker {
	broker, err := startBroker(brokerUrl)
	if err != nil {
		panic("Failed to start broker: " + err.Error())
	}

	return &MongoStorageWithBroker{
		mongo:  CreateMongoStorage(dbUrl, dbName),
		broker: broker,
	}
}

func GetMongoStorageWithoutBroker() *MongoStorageWithBroker {
	onceStorage.Do(func() {
		mongoUrl := utils.GetEnvVar("MONGO_URL")
		mongoDbName := utils.GetEnvVar("MONGO_DBNAME")
		mongoStorageWithoutBroker = &MongoStorageWithBroker{
			mongo:  CreateMongoStorage(mongoUrl, mongoDbName),
			broker: nil,
		}
	})
	return mongoStorageWithoutBroker
}
