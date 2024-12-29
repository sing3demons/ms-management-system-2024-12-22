package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sing3demons/logger-kp/logger"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	ErrUserNotFound = "user not found"
	ErrIdInvalid    = "invalid ID"
)

type CollectionInterface interface {
	Name() string
	FindOne(ctx context.Context, filter interface{}, opts ...options.Lister[options.FindOneOptions]) *mongo.SingleResult
	InsertOne(ctx context.Context, document interface{}, opts ...options.Lister[options.InsertOneOptions]) (*mongo.InsertOneResult, error)
	Find(ctx context.Context, filter interface{}, opts ...options.Lister[options.FindOptions]) (*mongo.Cursor, error)
	UpdateOne(ctx context.Context, filter interface{}, update interface{}, opts ...options.Lister[options.UpdateOneOptions]) (*mongo.UpdateResult, error)
}

type Repository[T any] struct {
	collection CollectionInterface
}

func NewRepository[T any](collection CollectionInterface) *Repository[T] {
	return &Repository[T]{
		collection: collection,
	}
}

type MongoBody struct {
	Collection string         `json:"Collection"`
	Method     string         `json:"Method"`
	Query      any            `json:"Query"`
	Document   any            `json:"Document"`
	Options    any            `json:"options"`
	SortItems  map[string]any `json:"sort_items"`
	Projection map[string]any `json:"projection"`
}
type ProcessOutputLog struct {
	Body    MongoBody `json:"Body"`
	RawData string    `json:"RawData"`
}

const (
	MongoPrefix = "mongo:"
)

func generateInvoke() string {
	return MongoPrefix + uuid.New().String()
}

func getModel(name, method string) string {
	return fmt.Sprintf("db.%s.%s", name, method)
}

func getInvoke() string {
	id, err := uuid.NewUUID()
	if err != nil {
		return uuid.NewString()
	}
	return MongoPrefix + id.String()
}

type Document[T any] struct {
	Filter     any
	New        T
	Options    map[string]any
	SortItems  map[string]any
	Projection map[string]any
}

func (r *Repository[T]) Create(ctx context.Context, doc *T, detailLog logger.DetailLog, summaryLog logger.SummaryLog) (*T, error) {
	node := "mongo"
	cmd := "insertOne"
	invoke := generateInvoke()
	collectionName := r.collection.Name()
	method := "insertOne"
	model := getModel(collectionName, method)

	v := reflect.ValueOf(doc).Elem()
	idField := v.FieldByName("ID")
	if idField.Kind() == reflect.String && idField.String() == "" {
		idField.SetString(uuid.New().String())
	}

	if !idField.IsValid() {
		idField.SetString(uuid.New().String())
	}

	if idField.IsZero() {
		idField.SetString(uuid.New().String())
	}

	rawDataNew, _ := json.Marshal(doc)
	rawDataStr := strings.ReplaceAll(string(rawDataNew), "\"", "'")

	rawData := fmt.Sprintf("%s(%s", model, rawDataStr)
	rawData += ")"

	processReqLog := ProcessOutputLog{
		Body: MongoBody{
			Collection: collectionName,
			Method:     method,
			Query:      nil,
			Document:   doc,
			Options:    nil,
			SortItems:  nil,
			Projection: nil,
		},
		RawData: rawData,
	}

	detailLog.AddOutputRequest(node, cmd, invoke, processReqLog.RawData, processReqLog.Body)
	detailLog.End()
	insertOneResult, err := r.collection.InsertOne(ctx, doc)
	if err != nil {
		summaryLog.AddError(node, cmd, invoke, err.Error())
		detailLog.AddInputRequest(node, cmd, invoke, "", err)
		return nil, fmt.Errorf("failed to insert document: %w", err)
	}

	summaryLog.AddSuccess(node, cmd, invoke, "success")
	detailLog.AddInputRequest(node, cmd, invoke, "", map[string]interface{}{
		"Return": insertOneResult,
	})
	return doc, nil
}

func (r *Repository[T]) Find(ctx context.Context, doc Document[T], detailLog logger.DetailLog, summaryLog logger.SummaryLog) ([]*T, error) {
	node := "mongo"
	cmd := "findAll"
	collectionName := r.collection.Name()
	method := "find"
	invoke := fmt.Sprintf("mongo:%s", uuid.New().String())
	model := getModel(collectionName, method)

	if doc.Filter == nil {
		doc.Filter = make(map[string]any)
	}
	rawDataFilter, _ := json.Marshal(doc.Filter)
	rawData := fmt.Sprintf("%s(%s", model, strings.ReplaceAll(string(rawDataFilter), "\"", "'"))

	if doc.Projection != nil {
		rawDataProjection, _ := json.Marshal(doc.Projection)
		rawData += fmt.Sprintf(", %s", strings.ReplaceAll(string(rawDataProjection), "\"", "'"))
	}

	rawData += ")"

	if doc.SortItems != nil {
		rawDataSort, _ := json.Marshal(doc.SortItems)
		rawData = fmt.Sprintf("%s.sort(%s).toArray()", rawData, strings.ReplaceAll(string(rawDataSort), "\"", "'"))
	}

	processReqLog := ProcessOutputLog{
		Body: MongoBody{
			Collection: collectionName,
			Method:     method,
			Query:      doc.Filter,
			Document:   nil,
			Options:    nil,
			SortItems:  doc.SortItems,
			Projection: doc.Projection,
		},
		RawData: rawData,
	}

	detailLog.AddOutputRequest(node, cmd, invoke, processReqLog.RawData, processReqLog.Body)
	detailLog.End()

	opts := &options.FindOptionsBuilder{}

	if doc.SortItems != nil {
		// opts.Sort = doc.SortItems
		opts.SetSort(doc.SortItems)
	}

	if doc.Projection != nil {
		// opts.Projection = doc.Projection
		opts.SetProjection(doc.Projection)
	}

	cursor, err := r.collection.Find(ctx, doc.Filter, opts)
	if err != nil {
		summaryLog.AddError(node, cmd, invoke, err.Error())
		detailLog.AddInputRequest(node, cmd, invoke, "", err)
		return nil, fmt.Errorf("failed to find documents: %w", err)
	}

	defer cursor.Close(ctx)

	var results []*T
	if err = cursor.All(ctx, &results); err != nil {
		summaryLog.AddError(node, cmd, invoke, err.Error())
		detailLog.AddInputRequest(node, cmd, invoke, "", err)
		return nil, fmt.Errorf("failed to decode documents: %w", err)
	}

	summaryLog.AddSuccess(node, cmd, invoke, "success")
	detailLog.AddInputRequest(node, cmd, invoke, "", map[string]interface{}{
		"Return": results,
	})
	return results, nil
}

func (r *Repository[T]) FindOne(ctx context.Context, filter any, detailLog logger.DetailLog, summaryLog logger.SummaryLog) (*T, error) {
	node := "mongo"
	cmd := "findOne"
	collectionName := r.collection.Name()
	method := "findOne"
	invoke := fmt.Sprintf("mongo:%s", uuid.New().String())

	model := getModel(collectionName, method)

	rawDataFilter, _ := json.Marshal(filter)
	rawData := fmt.Sprintf("%s(%s", model, strings.ReplaceAll(string(rawDataFilter), "\"", "'"))
	rawData += ")"

	processReqLog := ProcessOutputLog{
		Body: MongoBody{
			Collection: collectionName,
			Method:     method,
			Query:      filter,
			Document:   nil,
			Options:    nil,
			SortItems:  nil,
			Projection: nil,
		},
		RawData: rawData,
	}

	detailLog.AddOutputRequest(node, cmd, invoke, processReqLog.RawData, processReqLog.Body)
	detailLog.End()

	var doc T
	err := r.collection.FindOne(ctx, filter).Decode(&doc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			summaryLog.AddError(node, cmd, invoke, ErrUserNotFound)
			detailLog.AddInputRequest(node, cmd, invoke, "", err)
			return nil, fmt.Errorf("failed to find document: %w", err)
		}

		summaryLog.AddError(node, cmd, invoke, err.Error())
		detailLog.AddInputRequest(node, cmd, invoke, "", err)
		return nil, fmt.Errorf("failed to find document: %w", err)
	}

	summaryLog.AddSuccess(node, cmd, invoke, "success")
	detailLog.AddInputRequest(node, cmd, invoke, "", doc)
	return &doc, nil
}

func (r *Repository[T]) UpdateOne(ctx context.Context, doc Document[T], detailLog logger.DetailLog, summaryLog logger.SummaryLog) error {
	node := "mongo"
	cmd := "updateOne"
	collectionName := r.collection.Name()
	method := "updateOne"
	invoke := getInvoke()

	model := getModel(collectionName, method)

	rawDataFilter, _ := json.Marshal(doc.Filter)

	// delete empty fields in the document
	newDoc, err := RemoveEmptyFields(doc.New)
	if err != nil {
		return fmt.Errorf("failed to update document: %w", err)
	}
	newDoc["update_at"] = time.Now()

	update := bson.M{
		"$set": newDoc,
	}
	rawDataNew, _ := json.Marshal(update)
	rawData := fmt.Sprintf("%s(%s, %s", model, strings.ReplaceAll(string(rawDataFilter), "\"", "'"), strings.ReplaceAll(string(rawDataNew), "\"", "'"))
	rawData += ")"

	processReqLog := ProcessOutputLog{
		Body: MongoBody{
			Collection: collectionName,
			Method:     method,
			Query:      doc.Filter,
			Document:   update,
			Options:    doc.Options,
			SortItems:  nil,
			Projection: nil,
		},
		RawData: rawData,
	}

	detailLog.AddOutputRequest(node, cmd, invoke, processReqLog.RawData, processReqLog.Body)
	detailLog.End()

	opts := &options.UpdateOneOptionsBuilder{}

	if doc.Options != nil {
		if doc.Options["upsert"] != nil {
			upsert := doc.Options["upsert"].(bool)
			if upsert {
				opts.SetUpsert(upsert)
			}
		}
	}

	updateResult, err := r.collection.UpdateOne(ctx, doc.Filter, update, opts)
	if err != nil {
		summaryLog.AddError(node, cmd, invoke, err.Error())
		detailLog.AddInputRequest(node, cmd, invoke, "", map[string]interface{}{
			"Return": err.Error(),
		})
		return fmt.Errorf("failed to update document: %w", err)
	}

	summaryLog.AddSuccess(node, cmd, invoke, "success")
	detailLog.AddInputRequest(node, cmd, invoke, "", map[string]interface{}{
		"Return": updateResult,
	})
	return nil
}

func RemoveEmptyFields[T any](input T) (map[string]interface{}, error) {
	data, err := json.Marshal(input) // Convert to JSON (respects omitempty)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	for key, value := range result {
		if isEmpty(value) {
			delete(result, key)
		}
	}

	return result, nil
}

func isEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		// Use reflection for other types
		return reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface())
	}
}
