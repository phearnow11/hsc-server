package mongodb

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	METRICS_COLLECTION     string = "metrics"
	DATAPOINTS_COLLECTION  string = "data_points"
	REPORTS_COLLECTION     string = "reports"
	STATUS_DATA_COLLECTION string = "status_data"
	// IMPORTANT: if want to set unique index add "unique" last
	INDEXES_n_UPSERT = map[string][]string{
		METRICS_COLLECTION:     {"metric_type", "cluster", "dashboard_name", "query"},
		DATAPOINTS_COLLECTION:  {"query", "timestamp"},
		STATUS_DATA_COLLECTION: {"query", "timestamp"},
	}
)

type Metric struct {
	Metric_type        string       `bson:"metrict_type"`
	Cluster            string       `bson:"cluster"`
	Dashboard_name     string       `bson:"dashboard_name"`
	Sub_Dashboard_name string       `bson:"sub_dashboard_name"`
	Query              string       `bson:"query"`
	Unit               []MetricUnit `bson:"unit"`
}

type MetricUnit struct {
	Family    string `bson:"family,omitempty"`
	Name      string `bson:"name,omitempty"`
	Plural    string `bson:"plural,omitempty"`
	ShortName string `bson:"short_name,omitempty"`
}

type Datapoint struct {
	Query     string  `bson:"query"`
	Timestamp int64   `bson:"timestamp"`
	Data      float64 `bson:"data"`
}

type TagsetDatapoint struct {
	Tagset    string  `bson:"tagset"`
	Timestamp int64   `bson:"timestamp"`
	Data      float64 `bson:"data"`
}

type Stats struct {
	Suites   int    `bson:"suites"`
	Tests    int    `bson:"tests"`
	Passes   int    `bson:"passes"`
	Pending  int    `bson:"pending"`
	Failures int    `bson:"failures"`
	Start    string `bson:"start"`
	End      string `bson:"end"`
	Duration int    `bson:"duration"`
}

type Test struct {
	Title     string             `bson:"title"`
	Type      string             `bson:"type"`
	Zone      string             `bson:"zone"`
	State     string             `bson:"state"`
	Duration  int                `bson:"duration"`
	TestError interface{}        `bson:"testError"`
	ID        primitive.ObjectID `bson:"_id"`
}

type Reports struct {
	ID        primitive.ObjectID `bson:"_id"`
	Name      string             `bson:"name"`
	Version   int                `bson:"__v"`
	CreatedAt time.Time          `bson:"createdAt"`
	FileName  string             `bson:"fileName"`
	Stats     Stats              `bson:"stats"`
	Tests     []Test             `bson:"tests"`
}
