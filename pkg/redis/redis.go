package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	_ "pigdata/datadog/processingServer/internal/setup"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

var client redis.Client

func connect() *redis.Client {
	//client = *redis.NewFailoverClient(&redis.FailoverOptions{
	//	MasterName:    os.Getenv("RD_MASTER_NAME"),
	//	SentinelAddrs: []string{fmt.Sprintf("%s:%s", os.Getenv("RD_SENTINEL_HOST"), os.Getenv("RD_SENTINEL_PORT"))},
	//	DB:            0,
	//})

	//err := client.Ping(context.Background()).Err()
	//if err != nil {
	//	log.Println("Connect to Redis Sentinel agent at", fmt.Sprintf("%s:%s", os.Getenv("RD_SENTINEL_HOST"), os.Getenv("RD_SENTINEL_PORT")), "failed", err, ", Retry in 1 min")
	//}

	client = *redis.NewClient(&redis.Options{
		Addr:     os.Getenv("RD_ADDR"),
		Password: os.Getenv("RD_PWD"),
		DB:       0,
	})

	err := client.Do(context.Background(), "CONFIG", "SET", "stop-writes-on-bgsave-error", "no").Err()
	if err != nil {
		log.Printf("Failed to set configuration: %v\n", err)
	}

	log.Println("Successfully disabled stop-writes-on-bgsave-error")

	err = client.Ping(context.Background()).Err()
	if err != nil {
		log.Println("Connect to Redis agent at", os.Getenv("RD_ADDR"), "failed", err, ", attempting to connect at", os.Getenv("RD_ADDR_2"))
		client = *redis.NewClient(&redis.Options{
			Addr:     os.Getenv("RD_ADDR_2"),
			Password: os.Getenv("RD_PWD_2"),
			DB:       0,
		})

	}

	// client.ConfigSet(context.Background(), "save", "900 100 300 1000 60 10000")
	return &client
}

func IsConnected() bool {
	err := client.Ping(context.Background()).Err()
	if err != nil {
		log.Println("Redis is disconnected:", err)
		return false
	}
	return true
}

func SetData(ctx context.Context, data interface{}, token string) error {
	expT, err := time.ParseDuration(os.Getenv("RD_EXP"))
	if err != nil {
		return err
	}
	err = client.Set(ctx, token, data, expT).Err()
	if err != nil {
		return err
	}

	return nil
}

func MalshalAndSetData(ctx context.Context, data interface{}, token string) error {
	expT, err := time.ParseDuration(os.Getenv("RD_EXP"))
	if err != nil {
		return err
	}
	datamarshaled, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = client.Set(ctx, token, datamarshaled, expT).Err()
	if err != nil {
		return err
	}

	return nil
}

func GetIntData(ctx context.Context, token string) (int, error) {
	return client.Get(ctx, token).Int()
}

func GetStringData(ctx context.Context, token string) string {
	return client.Get(ctx, token).Val()
}

func GetData(ctx context.Context, token string) ([]byte, error) {
	val, err := client.Get(ctx, token).Bytes()
	if err != nil {
		return nil, err
	}
	return val, nil
}

func isInRange(ctx context.Context, c_max int64, c_min int64, service string) (bool, string) {
	o_max, err := GetIntData(ctx, "maxt-"+service)
	if err != nil {
		log.Println("Max ts not found:", err)
		return false, ""
	}
	o_min, err := GetIntData(ctx, "mint-"+service)
	if err != nil {
		log.Println("Min ts not found:", err)
		return false, ""
	}
	if c_max > int64(o_max) || c_min < int64(o_min) {
		return false, ""
	}
	key := strconv.Itoa(o_min) + "_" + strconv.Itoa(o_max) + "_" + service
	return true, key
}

func CheckRangeGetData(ctx context.Context, ts_range_string string) ([]byte, error) {
	// ts_range_string format from_to_service
	split_key := strings.Split(ts_range_string, "_")
	from_s, to_s, service := split_key[0], split_key[1], split_key[2]
	// check in range of old ts range
	// key format maxts-service mins-service
	min, _ := strconv.ParseInt(from_s, 10, 64)
	max, _ := strconv.ParseInt(to_s, 10, 64)

	inRange, o_key := isInRange(ctx, max, min, service)

	if !inRange {
		return nil, fmt.Errorf("Given Timestamp is not in range, need to get new data")
	}
	log.Println("Timestamp", from_s, to_s, "of service:", service, "is in range")
	val, err := client.Get(ctx, o_key).Bytes()
	if err != nil {
		return nil, err
	}
	return val, nil
}

func CheckRangeSetData(ctx context.Context, ts_range_string string, data interface{}) error {
	split_key := strings.Split(ts_range_string, "_")
	// maxx, _ := strconv.ParseInt(split_key[1], 10, 64)
	// split_key[1] = strconv.Itoa(int(maxx) + 60)
	expT, err := time.ParseDuration(os.Getenv("RD_EXP"))
	if err != nil {
		return err
	}
	err = client.Set(ctx, "maxt-"+split_key[2], split_key[1], expT).Err()
	if err != nil {
		log.Println("Set maxt failed:", err)
	}
	err = client.Set(ctx, "mint-"+split_key[2], split_key[0], expT).Err()
	if err != nil {
		log.Println("Set mint failed:", err)
	}
	ts_range_string = strings.Join(split_key[:], "_")
	err = client.Set(ctx, ts_range_string, data, expT).Err()
	if err != nil {
		return err
	}
	return nil
}

func CheckRedisConnectionPeriodically(ctx context.Context, redisStatus *int32) {
	log.Println("Redis checker initialized")

	for {
		_, err := client.Ping(ctx).Result()
		if err != nil {
			log.Printf("Redis checker status: Redis connection failed: %v, Retry connection", err)
			atomic.StoreInt32(redisStatus, 0) // Set status to unhealthy (0)
			connect()
		} else {
			log.Println("Redis checker status: Redis connection is OK")
			atomic.StoreInt32(redisStatus, 1) // Set status to healthy (1)
		}
		time.Sleep(time.Minute)
	}
}

func ResetRedis(ctx context.Context) error {
	err := client.FlushDB(ctx).Err()
	if err != nil {
		return fmt.Errorf("Reset redis error: %v", err)
	}
	return nil
}

func LinkedListSetData(ctx context.Context, data interface{}, ts_range_string string) error {
	return nil
}
