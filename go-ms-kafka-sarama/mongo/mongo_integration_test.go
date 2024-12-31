package mongo_test

import (
	"context"
	"testing"

	m "github.com/sing3demons/saram-kafka/mongo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type MockClient struct {
	mock.Mock
}

func (m *MockClient) Connect(ctx context.Context, opts ...*options.ClientOptions) (*mongo.Client, error) {
	args := m.Called(ctx, opts)
	return args.Get(0).(*mongo.Client), args.Error(1)
}

// Ping(ctx context.Context, rp *readpref.ReadPref) error
func (m *MockClient) Ping(ctx context.Context, rp *readpref.ReadPref) error {
	args := m.Called(ctx, rp)
	return args.Error(0)
}

// TestInitMongo
func TestInitMongo(t *testing.T) {
	mockClient := new(MockClient)

	// Mock behavior
	mockClient.On("Connect", mock.Anything, mock.Anything).Return(&mongo.Client{}, nil)
	mockClient.On("Ping", mock.Anything, mock.Anything).Return(nil)

	// Mock URI and Database Name
	uri := "mongodb://localhost:27017"
	dbName := "testdb"

	db, err := m.InitMongo(uri, dbName)

	assert.NoError(t, err)
	assert.NotNil(t, db)
}
