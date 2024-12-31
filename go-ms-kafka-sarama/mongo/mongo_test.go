package mongo

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MockMongoCollection struct {
	mock.Mock
}

func (m *MockMongoCollection) Name() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockMongoCollection) FindOne(ctx context.Context, filter interface{}, opts ...options.Lister[options.FindOneOptions]) *mongo.SingleResult {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(*mongo.SingleResult)
}

func (m *MockMongoCollection) Find(ctx context.Context, filter interface{}, opts ...options.Lister[options.FindOptions]) (*mongo.Cursor, error) {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(*mongo.Cursor), args.Error(1)
}

func (m *MockMongoCollection) InsertOne(ctx context.Context, document interface{}, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
	args := m.Called(ctx, document, opts)
	return args.Get(0).(*mongo.InsertOneResult), args.Error(1)
}

func (m *MockMongoCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error) {
	args := m.Called(ctx, filter, update, opts)
	return args.Get(0).(*mongo.UpdateResult), args.Error(1)
}

func (m *MockMongoCollection) DeleteOne(ctx context.Context, filter interface{}, opts ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error) {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(*mongo.DeleteResult), args.Error(1)
}

func TestCollectionName(t *testing.T) {
	mockCollection := new(MockMongoCollection)
	mockCollection.On("Name").Return("testCollection")

	c := NewCollection(mockCollection)
	assert.Equal(t, "testCollection", c.Name())

	mockCollection.AssertExpectations(t)
}

func TestCollectionFindOne(t *testing.T) {
	ctx := context.TODO()
	filter := bson.M{"_id": "12345"}
	mockCollection := new(MockMongoCollection)
	mockResult := &mongo.SingleResult{}

	mockCollection.On("FindOne", ctx, filter, mock.Anything).Return(mockResult)

	c := NewCollection(mockCollection)
	result := c.FindOne(ctx, filter)

	assert.Equal(t, mockResult, result)
	mockCollection.AssertExpectations(t)
}

func TestCollectionFind(t *testing.T) {
	ctx := context.TODO()
	filter := bson.M{"name": "test"}
	mockCollection := new(MockMongoCollection)
	mockCursor := &mongo.Cursor{}

	mockCollection.On("Find", ctx, filter, mock.Anything).Return(mockCursor, nil)

	c := NewCollection(mockCollection)
	cursor, err := c.Find(ctx, filter)

	assert.Nil(t, err)
	assert.Equal(t, mockCursor, cursor)
	mockCollection.AssertExpectations(t)
}

func TestCollectionInsertOne(t *testing.T) {
	ctx := context.TODO()
	document := bson.M{"name": "test"}
	mockCollection := new(MockMongoCollection)
	mockResult := &mongo.InsertOneResult{InsertedID: "12345"}

	mockCollection.On("InsertOne", ctx, document, mock.Anything).Return(mockResult, nil)

	c := NewCollection(mockCollection)
	result, err := c.InsertOne(ctx, document)

	assert.Nil(t, err)
	assert.Equal(t, mockResult, result)
	mockCollection.AssertExpectations(t)
}

func TestCollectionUpdateOne(t *testing.T) {
	ctx := context.TODO()
	filter := bson.M{"_id": "12345"}
	update := bson.M{"$set": bson.M{"name": "updated"}}
	mockCollection := new(MockMongoCollection)
	mockResult := &mongo.UpdateResult{MatchedCount: 1, ModifiedCount: 1}

	mockCollection.On("UpdateOne", ctx, filter, update, mock.Anything).Return(mockResult, nil)

	c := NewCollection(mockCollection)
	result, err := c.UpdateOne(ctx, filter, update)

	assert.Nil(t, err)
	assert.Equal(t, mockResult, result)
	mockCollection.AssertExpectations(t)
}

func TestCollectionDeleteOne(t *testing.T) {
	ctx := context.TODO()
	filter := bson.M{"_id": "12345"}
	mockCollection := new(MockMongoCollection)
	mockResult := &mongo.DeleteResult{DeletedCount: 1}

	mockCollection.On("DeleteOne", ctx, filter, mock.Anything).Return(mockResult, nil)

	c := NewCollection(mockCollection)
	result, err := c.DeleteOne(ctx, filter)

	assert.Nil(t, err)
	assert.Equal(t, mockResult, result)
	mockCollection.AssertExpectations(t)
}
