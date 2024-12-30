package repository

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sing3demons/logger-kp/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MockCollection struct {
	mock.Mock
}

func (m *MockCollection) FindOne(ctx context.Context, filter interface{}, opts ...options.Lister[options.FindOneOptions]) *mongo.SingleResult {
	args := m.Called(ctx, filter, opts)
	var result any
	args.Get(0).(*mongo.SingleResult).Decode(result)

	return args.Get(0).(*mongo.SingleResult)
}

func (m *MockCollection) InsertOne(ctx context.Context, document interface{}, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error) {
	args := m.Called(ctx, document, opts)
	return args.Get(0).(*mongo.InsertOneResult), args.Error(1)
}

func (m *MockCollection) Find(ctx context.Context, filter interface{}, opts ...options.Lister[options.FindOptions]) (*mongo.Cursor, error) {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(*mongo.Cursor), args.Error(1)
}

func (m *MockCollection) UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error) {
	args := m.Called(ctx, filter, update, opts)
	return args.Get(0).(*mongo.UpdateResult), args.Error(1)
}

func (m *MockCollection) DeleteOne(ctx context.Context, filter interface{}, opts ...options.Lister[options.DeleteOneOptions]) (*mongo.DeleteResult, error) {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(*mongo.DeleteResult), args.Error(1)
}

func (m *MockCollection) Name() string {
	return "mock"
}

type TestDocument struct {
	ID   string `bson:"_id"`
	Name string `bson:"name"`
}

// TestGetInvoke tests the getInvoke function
func TestGetInvoke(t *testing.T) {
	invoke := getInvoke()

	// Check if the result starts with "mongo:"
	if !strings.HasPrefix(invoke, MongoPrefix) {
		t.Errorf("expected string to start with 'mongo:', got %s", invoke)
	}

	// Check if the remaining part is a valid UUID
	_, err := uuid.Parse(invoke[len(MongoPrefix):])
	if err != nil {
		t.Errorf("expected a valid UUID after 'mongo:', got %s", invoke[len(MongoPrefix):])
	}
}

func TestGetModel(t *testing.T) {
	// Call the function under test
	result := getModel("example", "find")

	// Assert the result is as expected
	if result != "db.example.find" {
		t.Errorf("expected result: %v, got: %v", "db.example.find", result)
	}
}

func TestGenerateInvoke(t *testing.T) {
	// Call the function under test
	result := generateInvoke()

	// Check if the result starts with "mongo:"
	if !strings.HasPrefix(result, MongoPrefix) {
		t.Errorf("expected string to start with 'mongo:', got %s", result)
	}

	// Check if the remaining part is a valid UUID
	// This is done by checking if the string after "mongo:" is a valid UUID
	_, err := uuid.Parse(result[len(MongoPrefix):])
	if err != nil {
		t.Errorf("expected a valid UUID after 'mongo:', got %s", result[len(MongoPrefix):])
	}
}

func mapErrMsg(str, v string) string {
	return fmt.Sprintf("%s: %s", str, v)
}

func TestFindOneSuccess(t *testing.T) {
	ctx := context.TODO()
	filter := bson.M{"_id": "12345"}

	// Create a mock collection
	mockCollection := new(MockCollection)

	// Mock SingleResult to simulate the response
	mockResult := mongo.NewSingleResultFromDocument(&TestDocument{
		ID:   "12345",
		Name: "Test Name",
	}, nil, nil)

	// Set up the mock behavior
	mockCollection.On("FindOne", ctx, filter, mock.Anything).Return(mockResult)

	// Inject the mock collection into the repository
	repo := &Repository[TestDocument]{
		collection: mockCollection,
	}

	// Call the method under test
	result, err := repo.FindOne(ctx, filter, logger.NewDetailLog("mock", "mock", "mock"), logger.NewSummaryLog("mock", "mock", "mock"))
	if err != nil {
		t.Error(mapErrMsg("unexpected error", err.Error()))
	}

	assert.NotNil(t, result)

	// Verify all expectations
	mockCollection.AssertExpectations(t)
}

func TestNewRepository(t *testing.T) {
	// Create a mock collection
	mockCollection := new(MockCollection)

	// Create a new repository with the mock collection
	repo := NewRepository[TestDocument](mockCollection)

	// Assert that the repository is not nil
	if repo == nil {
		t.Errorf("expected repository to be non-nil")
	}

	// Assert that the collection in the repository is the mock collection
	if repo.collection != mockCollection {
		t.Errorf("expected collection: %v, got: %v", mockCollection, repo.collection)
	}
}

func TestInsertOneSuccess(t *testing.T) {
	ctx := context.TODO()
	document := &TestDocument{
		ID:   "12345",
		Name: "Test Name",
	}

	// Create a mock collection
	mockCollection := new(MockCollection)

	// Mock the InsertOne method to simulate the response
	mockResult := &mongo.InsertOneResult{
		InsertedID: "12345",
	}

	// Set up the mock behavior
	mockCollection.On("InsertOne", ctx, document, mock.Anything).Return(mockResult, nil)

	// Inject the mock collection into the repository
	repo := &Repository[TestDocument]{
		collection: mockCollection,
	}

	// Call the method under test
	result, err := repo.Create(ctx, document, logger.NewDetailLog("mock", "mock", "mock"), logger.NewSummaryLog("mock", "mock", "mock"))
	if err != nil {
		t.Error(mapErrMsg("unexpected error", err.Error()))
	}

	assert.NotNil(t, result)

	// Verify all expectations
	mockCollection.AssertExpectations(t)
}

func TestDeleteOneSuccess(t *testing.T) {
	ctx := context.TODO()
	filter := bson.M{"_id": "12345"}

	// Create a mock collection
	mockCollection := new(MockCollection)

	// Mock the UpdateOne method to simulate the response
	mockResult := &mongo.UpdateResult{
		MatchedCount:  1,
		ModifiedCount: 1,
	}

	// Set up the mock behavior
	mockCollection.On("UpdateOne", ctx, filter, mock.Anything, mock.Anything).Return(mockResult, nil)

	// Inject the mock collection into the repository
	repo := &Repository[TestDocument]{
		collection: mockCollection,
	}

	// Call the method under test
	err := repo.DeleteOne(ctx, filter, logger.NewDetailLog("mock", "mock", "mock"), logger.NewSummaryLog("mock", "mock", "mock"))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify all expectations
	mockCollection.AssertExpectations(t)
}

func TestRemoveEmptyFields(t *testing.T) {
	type TestStruct struct {
		Field1 string                 `json:"field1,omitempty"`
		Field2 int                    `json:"field2,omitempty"`
		Field3 []string               `json:"field3,omitempty"`
		Field4 map[string]interface{} `json:"field4,omitempty"`
		Field5 *string                `json:"field5,omitempty"`
	}

	tests := []struct {
		name     string
		input    TestStruct
		expected map[string]interface{}
	}{
		{
			name: "All fields empty",
			input: TestStruct{
				Field1: "",
				Field2: 0,
				Field3: nil,
				Field4: nil,
				Field5: nil,
			},
			expected: map[string]interface{}{},
		},
		{
			name: "Some fields empty",
			input: TestStruct{
				Field1: "value1",
				Field2: 0,
				Field3: nil,
				Field4: map[string]interface{}{"key": "value"},
				Field5: nil,
			},
			expected: map[string]interface{}{
				"field1": "value1",
				"field4": map[string]interface{}{"key": "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RemoveEmptyFields(tt.input)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected: %v, got: %v", tt.expected, result)
			}
		})
	}
}
