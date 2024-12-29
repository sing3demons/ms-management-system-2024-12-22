package mongo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type EMethod string

const (
	InsertOne         EMethod = "insertOne"
	InsertMany        EMethod = "insertMany"
	UpdateOne         EMethod = "updateOne"
	UpdateMany        EMethod = "updateMany"
	DeleteOne         EMethod = "deleteOne"
	DeleteMany        EMethod = "deleteMany"
	FindOne           EMethod = "findOne"
	Find              EMethod = "find"
	FindOneAndUpdate  EMethod = "findOneAndUpdate"
	FindOneAndDelete  EMethod = "findOneAndDelete"
	FindOneAndReplace EMethod = "findOneAndReplace"
)

type ResultMongo struct {
	Err            bool        `json:"err"`
	ResultDesc     string      `json:"result_desc"`
	ResultData     interface{} `json:"result_data"`
	OutgoingDetail struct {
		Body    interface{} `json:"Body"`
		RawData string      `json:"RawData"`
	} `json:"outgoing_detail"`
	IngoingDetail struct {
		Body    interface{} `json:"Body"`
		RawData string      `json:"RawData"`
	} `json:"ingoing_detail"`
}

type TDocument struct {
	Filter     bson.M      `json:"filter"`
	New        bson.M      `json:"new"`
	Options    OptionMongo `json:"options"`
	SortItems  bson.M      `json:"sort_items"`
	Projection bson.M      `json:"projection"`
}

type InputLog struct {
	Body struct {
		Return any `json:"Return"`
	}
	RawData string `json:"RawData"`
}

var db *mongo.Database

func InitMongo(uri, dbName string) *mongo.Database {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		panic(err)
	}
	db = client.Database(dbName)
	fmt.Println("Database connected")
	return db
}

type OptionMongo struct {
	*options.InsertOneOptions
	*options.UpdateOneOptions
	*options.DeleteOneOptions
	*options.FindOneOptions
	*options.FindOptions
	*options.FindOneAndUpdateOptions
	*options.FindOneAndDeleteOptions
	*options.FindOneAndReplaceOptions
}

type MongoBody struct {
	Collection string  `json:"Collection"`
	Method     EMethod `json:"Method"`
	Query      bson.M  `json:"Query"`
	Document   bson.M  `json:"Document"`
	Options    any     `json:"options"`
	SortItems  bson.M  `json:"sort_items"`
	Projection bson.M  `json:"projection"`
}
type ProcessOutputLog struct {
	Body    MongoBody `json:"Body"`
	RawData string    `json:"RawData"`
}

type InputBody struct {
	Return any `json:"Return"`
}
type ProcessInputLog struct {
	Body    InputBody `json:"Body"`
	RawData string    `json:"RawData"`
}

func Mongo(collectionName string, method EMethod, document TDocument) ResultMongo {
	var result ResultMongo
	result.ResultDesc = "success"
	collection := db.Collection(collectionName)

	processReqLog := ProcessOutputLog{
		Body: MongoBody{
			Collection: collectionName,
			Method:     method,
			Query:      document.Filter,
			Document:   document.New,
			Options:    nil,
			SortItems:  document.SortItems,
			Projection: document.Projection,
		},
		RawData: "",
	}

	processResLog := ProcessInputLog{
		Body: InputBody{
			Return: nil,
		},
		RawData: "",
	}

	// var operationResult interface{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch method {
	case InsertOne:
		if document.New == nil {
			result.Err = true
			result.ResultDesc = "'new' is required for insertOne"
			processResLog.Body.Return = result
			result.IngoingDetail.Body = processResLog.Body
			break
		}

		operationResult, err := collection.InsertOne(ctx, document.New)
		if err != nil {
			result.Err = true
			result.ResultDesc = fmt.Sprintf("failed to insert one: %v", err)

		}
		result.ResultData = operationResult
		processResLog.Body.Return = operationResult
		result.IngoingDetail.Body = processResLog.Body

	case UpdateOne:
		if document.Filter == nil || document.New == nil {
			result.Err = true
			result.ResultDesc = "'filter' and 'new' are required for updateOne"
			processResLog.Body.Return = result
			result.IngoingDetail.Body = processResLog.Body
			break
		}

		operationResult, err := collection.UpdateOne(ctx, document.Filter, document.New)
		if err != nil {
			result.Err = true
			result.ResultDesc = fmt.Sprintf("failed to update one: %v", err)
		}

		result.ResultData = operationResult
		processResLog.Body.Return = operationResult
		result.IngoingDetail.Body = processResLog.Body
	case UpdateMany:
		if document.Filter == nil || document.New == nil {
			result.Err = true
			result.ResultDesc = "'filter' and 'new' are required for updateMany"
			processResLog.Body.Return = result
			break
		}

		operationResult, err := collection.UpdateMany(ctx, document.Filter, document.New)
		if err != nil {
			result.Err = true
			result.ResultDesc = fmt.Sprintf("failed to update many: %v", err)
		}

		result.ResultData = operationResult
		processResLog.Body.Return = operationResult
		result.IngoingDetail.Body = processResLog.Body
	case DeleteOne:
		if document.Filter == nil {
			result.Err = true
			result.ResultDesc = "'filter' is required for deleteOne"
			processResLog.Body.Return = result
			break
		}

		operationResult, err := collection.DeleteOne(ctx, document.Filter)
		if err != nil {
			result.Err = true
			result.ResultDesc = fmt.Sprintf("failed to delete one: %v", err)
			processResLog.Body.Return = result
			result.IngoingDetail.Body = processResLog.Body
		}

		result.ResultData = operationResult
		processResLog.Body.Return = operationResult
		result.IngoingDetail.Body = processResLog.Body

	case FindOne:
		if document.Filter == nil {
			result.Err = true
			result.ResultDesc = "'filter' is required for findOne"
			processResLog.Body.Return = result
			return result
		}

		operationResult := collection.FindOne(ctx, document.Filter)
		if err := operationResult.Err(); err != nil {
			result.Err = true
			result.ResultDesc = fmt.Sprintf("failed to find one: %v", err)
			processResLog.Body.Return = result

		}
		processResLog.Body.Return = operationResult
		result.IngoingDetail.Body = processResLog.Body
		result.ResultData = operationResult
	case Find:

		operationResult, err := collection.Find(ctx, document.Filter)
		if err != nil {
			result.Err = true
			result.ResultDesc = fmt.Sprintf("failed to find: %v", err)
			processResLog.Body.Return = result

		}

		operationResult.All(ctx, &result.ResultData)
		processResLog.Body.Return = operationResult
		result.IngoingDetail.Body = processResLog.Body
		result.ResultData = operationResult

	case FindOneAndUpdate:
		if document.Filter == nil || document.New == nil {
			result.Err = true
			result.ResultDesc = "'filter' and 'new' are required for findOneAndUpdate"
			processResLog.Body.Return = result
			break
		}

		operationResult := collection.FindOneAndUpdate(ctx, document.Filter, document.New)
		if err := operationResult.Err(); err != nil {
			result.Err = true
			result.ResultDesc = fmt.Sprintf("failed to find one and update: %v", err)
			processResLog.Body.Return = result
		}
		processResLog.Body.Return = operationResult
		result.IngoingDetail.Body = processResLog.Body
		result.ResultData = operationResult
	// Add cases for other methods as required...
	default:
		result.Err = true
		result.ResultDesc = fmt.Sprintf("unsupported method: %s", method)
		processResLog.Body.Return = result
		return result
	}

	result.Err = false
	result.ResultDesc = "success"
	result.OutgoingDetail.Body = processReqLog.Body
	result.OutgoingDetail.RawData = processReqLog.RawData
	result.IngoingDetail.Body = processResLog.Body
	result.IngoingDetail.RawData = processResLog.RawData

	return result
}
