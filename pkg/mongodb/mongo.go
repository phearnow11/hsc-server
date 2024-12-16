package mongodb

import (
	"context"
	"fmt"
	"log"
	"os"
	"pigdata/datadog/processingServer/pkg/utils"
	"strings"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	CLIENT_OPTION    *options.ClientOptions
	URI              = "mongodb://" + os.Getenv("MONGO_HOST") + ":" + os.Getenv("MONGO_PORT")
	mongo_msg_prefix = "MONGO:"
	db_name          = os.Getenv("MONGO_DBNAME")
)

func init() {
	credential := options.Credential{
		Username: os.Getenv("MONGO_INITDB_ROOT_USERNAME"),
		Password: os.Getenv("MONGO_INITDB_ROOT_PASSWORD"),
	}
	CLIENT_OPTION = options.Client().ApplyURI(URI).SetAuth(credential)
	fmt.Println("URI:" + URI)
}

func Init_database() {
	init_msg := mongo_msg_prefix + "INIT:"
	if !Check_exist_collection(METRICS_COLLECTION) {
		log.Println(init_msg, "Metric collection not found")
		log.Println(init_msg, init_metrics_collection())
	}
}

func connect() (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, CLIENT_OPTION)
	if err != nil {
		return nil, fmt.Errorf("Mongo connection failed %s", err)
	}
	return client, nil
}

func Metric_data_processing(data datadogV1.MetricsQueryResponse) {
	fmt.Println(data.GetQuery())
	if true {
		return
	}
	if len(data.GetSeries()) <= 0 {
		log.Println("Mongo DB: Error processing Metric data: Series is less than 0")
		return
	}
}

func Add_to_db(data []interface{}, collection string) error {
	adddb_msg := mongo_msg_prefix + "ADD:"
	client, err := connect()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	defer client.Disconnect(ctx)
	db := client.Database(db_name)

	var dbCollection *mongo.Collection
	if !Check_exist_collection(collection) {
		log.Println(adddb_msg, "Collection", collection, "not found, create collection")

		err = db.CreateCollection(ctx, collection)
		if err != nil {
			return fmt.Errorf("%s Collection failed to create %s", adddb_msg, err)
		}
		dbCollection := db.Collection(collection)
		err = createCompoundIndex(dbCollection, INDEXES_n_UPSERT[collection])
		if err != nil {
			return fmt.Errorf("%s Create indexes error: %s", adddb_msg, err)
		}
	} else {
		dbCollection = db.Collection(collection)
	}
	if dbCollection == nil {
		return fmt.Errorf("%s Collection is nil: %v", adddb_msg, err)
	}
	insertops := options.InsertMany().SetOrdered(false)
	_, err = dbCollection.InsertMany(ctx, data, insertops)
	if err != nil {
		return fmt.Errorf("%s Insert failed %s", adddb_msg, err)
	}
	return nil
}

func list_all_collection() []string {
	err_msg := "MONGO: List all collection:"
	client, err := connect()
	if err != nil {
		log.Println(err_msg, "Error", err)
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	defer client.Disconnect(ctx)
	db := client.Database(db_name)

	collection, err := db.ListCollectionNames(ctx, map[string]interface{}{})
	if err != nil {
		log.Println(err_msg, "Error:", err)
		return nil
	}
	return collection
}

func Check_exist_collection(collection string) bool {
	if utils.Contain(list_all_collection(), collection) >= 0 {
		return true
	}
	return false
}

func init_metrics_collection() error {
	metrics := []interface{}{}
	for cluster, query_map := range utils.Metrics.BusinessMetric {
		business_metric := Metric{Metric_type: "business", Cluster: cluster}
		for dashboard_name, query := range query_map {
			business_metric.Dashboard_name = dashboard_name

			query_spilt := strings.Split(query, "$$")
			query = query_spilt[0]

			business_metric.Query = query
			metrics = append(metrics, business_metric)
		}
	}
	for cluster, query_map := range utils.Metrics.ServiceMetric {
		service_metric := Metric{Metric_type: "service", Cluster: cluster}
		for dashboard_name, sub_query_map := range query_map {
			service_metric.Dashboard_name = dashboard_name
			for sub_dashboard_name, query := range sub_query_map {

				query_spilt := strings.Split(query, "$$")
				query = query_spilt[0]

				service_metric.Query = query
				service_metric.Sub_Dashboard_name = sub_dashboard_name
				metrics = append(metrics, service_metric)
			}
		}
	}
	log.Println("INIT METRICS: Metrics ready, adding to db", metrics)
	err := Add_to_db(metrics, METRICS_COLLECTION)
	if err != nil {
		return err
	}

	return nil
}

func createCompoundIndex(collection *mongo.Collection, fields []string) error {
	indexing_msg := mongo_msg_prefix + "INDEX:"
	index_name := collection.Name() + "_compound_index"
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	props := bson.D{}
	unique := false
	for _, field := range fields {
		if field == "unique" {
			unique = true
			continue
		}
		cur_prop := bson.E{Key: field, Value: 1}
		props = append(props, cur_prop)
	}

	indexModel := mongo.IndexModel{
		Keys:    props,
		Options: options.Index().SetName(index_name).SetUnique(unique),
	}
	indexName, err := collection.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return fmt.Errorf("%s %v", indexing_msg, err)
	}
	log.Println(indexing_msg, "Index created:", indexName)
	return nil
}

func Bulkwrite_upsert(data []interface{}, collection string) error {
	bulkwrite_msg := mongo_msg_prefix + "BULK WRITE:"
	client, err := connect()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	defer client.Disconnect(ctx)
	db := client.Database(db_name)

	var dbCollection *mongo.Collection
	if !Check_exist_collection(collection) {
		log.Println(bulkwrite_msg, "Collection", collection, "not found, create collection")

		err = db.CreateCollection(ctx, collection)
		if err != nil {
			return fmt.Errorf("%s Collection failed to create %s", bulkwrite_msg, err)
		}
		dbCollection := db.Collection(collection)
		err = createCompoundIndex(dbCollection, INDEXES_n_UPSERT[collection])
		if err != nil {
			return fmt.Errorf("%s Create indexes error: %s", bulkwrite_msg, err)
		}
	} else {
		dbCollection = db.Collection(collection)
	}
	var operations []mongo.WriteModel
	for _, doc := range data {
		bytes, err := bson.Marshal(doc)
		if err != nil {
			return fmt.Errorf("error marshaling to BSON: %v", err)
		}

		// Unmarshal BSON bytes into bson.M
		var docMap bson.M
		if err := bson.Unmarshal(bytes, &docMap); err != nil {
			return fmt.Errorf("error unmarshaling to bson.M: %v", err)
		}
		filter := bson.M{}
		for _, field := range INDEXES_n_UPSERT[collection] {
			filter[field] = docMap[field]
		}
		update := bson.M{"$set": docMap}
		operation := mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(true)

		operations = append(operations, operation)

	}
	bulkOpts := options.BulkWrite().SetOrdered(false)
	_, err = dbCollection.BulkWrite(ctx, operations, bulkOpts)
	if err != nil {
		return fmt.Errorf("%s error: %s", bulkwrite_msg, err)
	}
	return nil
}

func Drop_collection(collection string) error {
	client, err := connect()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer client.Disconnect(ctx)
	db := client.Database(db_name)
	return db.Collection(collection).Drop(ctx)
}

func Get_datapoints_in_range(query string, startTime, endTime int64) ([]Datapoint, error) {
	client, err := connect()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	defer client.Disconnect(ctx)
	db := client.Database(db_name)
	collection := db.Collection(DATAPOINTS_COLLECTION)

	filter := bson.M{
		"query":     query,
		"timestamp": bson.M{"$gte": startTime, "$lte": endTime},
	}

	opts := options.Find().SetSort(bson.D{{Key: "timestamp", Value: 1}})

	cursor, err := collection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var datapoints []Datapoint
	if err = cursor.All(context.TODO(), &datapoints); err != nil {
		return nil, err
	}
	return datapoints, nil
}

func StructToBsonM(s interface{}) (bson.M, error) {
	bsonBytes, err := bson.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("error marshalling struct: %v", err)
	}

	var result bson.M
	err = bson.Unmarshal(bsonBytes, &result)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling to bson.M: %v", err)
	}

	return result, nil
}

func FindLatestReportByName(name string) (*Reports, error) {
	var result Reports
	client, err := connect()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	defer client.Disconnect(ctx)
	db := client.Database(db_name)
	collection := db.Collection(REPORTS_COLLECTION)

	filter := bson.M{"name": name}
	// find latest document base on createAt
	options := options.FindOne().SetSort(bson.D{{Key: "createdAt", Value: -1}})

	err = collection.FindOne(ctx, filter, options).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &result, nil
}

func Add_status_report(name string, avail bool) error {
	log.Println("ADD STATUS REPORT", name, "AVAIL:", avail)
	stat := Stats{Failures: 0}
	report := Reports{Name: name, Stats: stat}
	if avail {
		stat.Failures = 0
	} else {
		stat.Failures = 1
	}
	data_add := []interface{}{report}
	return Bulkwrite_upsert(data_add, REPORTS_COLLECTION)
}
