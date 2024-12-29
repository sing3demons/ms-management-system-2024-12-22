//go:build integration
// +build integration

package repository

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func TestNewRepository(t *testing.T) {
	// Setup MongoDB client and collection for test using mongo.Connect
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Connect to the MongoDB server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Ping(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Get the test collection
	collection := client.Database("testdb").Collection("testcoll")

	// Create a new repository instance
	repo := NewRepository[string](collection)

	// Assert the repository is not nil and the collection is correctly set
	assert.NotNil(t, repo)
	assert.Equal(t, collection, repo.collection)

	defer func() {
		// clean up the test collection
		_, err := collection.DeleteMany(ctx, bson.M{})
		if err != nil {
			t.Fatalf("Failed to clean up test collection: %v", err)
		}

		// drop the test collection
		err = collection.Drop(ctx)
		if err != nil {
			t.Fatalf("Failed to drop test collection: %v", err)
		}

		// drop the test database
		err = client.Database("testdb").Drop(ctx)
		if err != nil {
			t.Fatalf("Failed to drop test database: %v", err)
		}

		// Disconnect from the MongoDB server
		err = client.Disconnect(ctx)
		if err != nil {
			t.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}

	}()
}

func TestInsert(t *testing.T) {
	// Setup MongoDB client and collection for test using mongo.Connect
	client, err := mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Connect to the MongoDB server
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err = client.Ping(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Get the test collection
	collection := client.Database("testdb").Collection("testcoll")

	// Create a new repository instance
	repo := NewRepository[string](collection)

	// Insert a new document
	insertResult, err := repo.collection.InsertOne(ctx, bson.M{"name": "test", "age": 30})
	if err != nil {
		t.Fatalf("Failed to insert document: %v", err)
	}

	// Verify insert result
	assert.NotNil(t, insertResult.InsertedID)

	// Clean up the inserted document (optional)
	_, err = repo.collection.DeleteOne(ctx, bson.M{"_id": insertResult.InsertedID})
	if err != nil {
		t.Fatalf("Failed to clean up test document: %v", err)
	}

	defer func() {
		// clean up the test collection
		_, err := collection.DeleteMany(ctx, bson.M{})
		if err != nil {
			t.Fatalf("Failed to clean up test collection: %v", err)
		}

		// drop the test collection
		err = collection.Drop(ctx)
		if err != nil {
			t.Fatalf("Failed to drop test collection: %v", err)
		}

		// drop the test database
		err = client.Database("testdb").Drop(ctx)
		if err != nil {
			t.Fatalf("Failed to drop test database: %v", err)
		}

		// Disconnect from the MongoDB server
		err = client.Disconnect(ctx)
		if err != nil {
			t.Fatalf("Failed to disconnect from MongoDB: %v", err)
		}

	}()
}
