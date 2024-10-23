package db

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

type PingNode struct {
	ID       int    `bson:"id" json:"id"`
	Name     string `bson:"name" json:"name"`
	Address  string `bson:"address" json:"address"`
	Enable   bool   `bson:"enable" json:"enable"`
	Interval int    `bson:"interval" json:"interval"`
}

type PingConfig struct {
	Version string     `bson:"version" json:"version"`
	Nodes   []PingNode `bson:"nodes" json:"nodes"`
}

func AddPingNode(name, address string) (PingNode, error) {
	collection := MG.CC("prob", "ping_node")
	//find the max id one, get the id
	cursor, err := collection.Find(context.TODO(), bson.M{}, options.Find().SetSort(bson.M{"id": -1}))
	if err != nil {
		return PingNode{}, err
	}
	var add PingNode = PingNode{ID: 0, Name: name, Address: address, Enable: true, Interval: 5}
	maxId := 0
	for cursor.Next(context.TODO()) {
		var node PingNode
		err := cursor.Decode(&node)
		if err != nil {
			return PingNode{}, err
		}
		if node.ID > maxId {
			maxId = node.ID
		}
	}
	add.ID = maxId + 1
	_, err = collection.InsertOne(context.TODO(), add)
	return add, err
}

func RemovePingNode(node PingNode) error {
	collection := MG.CC("prob", "ping_node")
	//update enable to false
	_, err := collection.UpdateOne(context.TODO(), bson.M{"id": node.ID}, bson.M{"$set": bson.M{"enable": false}})
	return err
}

func GetPingConfig() (PingConfig, error) {
	collection := MG.CC("prob", "ping_node")
	//load all nodes
	cursor, err := collection.Find(context.TODO(), bson.M{"enable": true}, options.Find().SetSort(bson.M{"id": 1}))
	if err != nil {
		return PingConfig{}, err
	}
	defer cursor.Close(context.TODO())
	ret := PingConfig{}
	for cursor.Next(context.TODO()) {
		var node PingNode
		err := cursor.Decode(&node)
		if err != nil {
			return PingConfig{}, err
		}
		ret.Nodes = append(ret.Nodes, node)
	}
	//generate version with node_id join with .
	version := ""
	for _, node := range ret.Nodes {
		version += fmt.Sprintf("%d.", node.ID)
	}
	if len(version) > 0 {
		version = version[:len(version)-1] //remove the last .
	}
	ret.Version = version
	return ret, nil
}

type PingData struct {
	NodeId    string    `bson:"node_id"`
	NodeName  string    `json:"node_name"`
	Address   string    `json:"address"`
	Latency   float64   `json:"latency"`
	Timestamp time.Time `json:"timestamp"`
}

var lastOptimizationTime time.Time
var optimizationMutex sync.Mutex
var once sync.Once

func InsertPingResult(data []PingData) error {
	collection := MG.CC("prob", "ping_result")
	once.Do(func() {
		CreatePingDataIndexes()
	})
	documents := make([]interface{}, len(data))
	for i, d := range data {
		documents[i] = d
	}
	_, err := collection.InsertMany(context.TODO(), documents)

	// 检查是否需要进行优化
	if time.Now().Sub(lastOptimizationTime) >= 24*time.Hour {
		if optimizationMutex.TryLock() {
			now := time.Now()
			go func() {
				defer optimizationMutex.Unlock()
				log.Println("Starting ping data optimization...")
				if err := OptimisePingData(); err != nil {
					log.Printf("Error optimizing ping data: %v", err)
				} else {
					log.Println("Ping data optimization completed successfully")
					lastOptimizationTime = now
				}
			}()
		}
	}
	return err
}

// optmise the ping data, for last 24 hours, every 5 seconds
// for last 7 days, every 1 minutes
// for last 30 days, every 5 minutes
// for last 90 days, every 15 minutes
// for last 1 year, every 1 hour
func OptimisePingData() error {
	collection := MG.CC("prob", "ping_result")
	now := time.Now()
	intervals := []struct {
		duration time.Duration
		interval time.Duration
	}{
		{24 * time.Hour, 5 * time.Second},
		{7 * 24 * time.Hour, 1 * time.Minute},
		{30 * 24 * time.Hour, 5 * time.Minute},
		{90 * 24 * time.Hour, 15 * time.Minute},
		{365 * 24 * time.Hour, 1 * time.Hour},
	}

	for _, interval := range intervals {
		startTime := now.Add(-interval.duration)
		endTime := startTime.Add(interval.duration)
		if err := optimizeInterval(collection.Collection, startTime, endTime, interval.interval); err != nil {
			return fmt.Errorf("failed to optimize interval %v: %v", interval.duration, err)
		}
	}

	// 清理超过一年的数据
	if err := cleanOldData(collection.Collection, now.AddDate(-1, 0, 0)); err != nil {
		return fmt.Errorf("failed to clean old data: %v", err)
	}

	return nil
}

func optimizeInterval(collection *mongo.Collection, startTime, endTime time.Time, intervalDuration time.Duration) error {
	pipeline := []bson.M{
		{"$match": bson.M{"time": bson.M{"$gte": startTime, "$lt": endTime}}},
		{"$group": bson.M{
			"_id": bson.M{
				"id":     "$id",
				"bucket": bson.M{"$toDate": bson.M{"$subtract": []interface{}{"$time", bson.M{"$mod": []interface{}{bson.M{"$toLong": "$time"}, intervalDuration.Milliseconds()}}}}},
			},
			"maxNumber": bson.M{"$max": "$number"},
			"maxTime":   bson.M{"$max": "$time"},
		}},
		{"$project": bson.M{
			"_id":    0,
			"id":     "$_id.id",
			"time":   "$maxTime",
			"number": "$maxNumber",
		}},
	}

	cursor, err := collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return fmt.Errorf("aggregation failed: %v", err)
	}
	defer cursor.Close(context.TODO())

	var optimizedData []interface{}
	for cursor.Next(context.TODO()) {
		var result PingData
		if err := cursor.Decode(&result); err != nil {
			return fmt.Errorf("decoding result failed: %v", err)
		}
		optimizedData = append(optimizedData, result)
	}

	if len(optimizedData) > 0 {
		// 删除原始数据
		_, err = collection.DeleteMany(context.TODO(), bson.M{
			"time": bson.M{"$gte": startTime, "$lt": endTime},
		})
		if err != nil {
			return fmt.Errorf("deleting original data failed: %v", err)
		}

		// 插入优化后的数据
		_, err = collection.InsertMany(context.TODO(), optimizedData)
		if err != nil {
			return fmt.Errorf("inserting optimized data failed: %v", err)
		}
	}

	return nil
}

func cleanOldData(collection *mongo.Collection, cutoffTime time.Time) error {
	_, err := collection.DeleteMany(
		context.TODO(),
		bson.M{"time": bson.M{"$lt": cutoffTime}},
	)
	if err != nil {
		return fmt.Errorf("failed to delete old data: %v", err)
	}
	return nil
}

func CreatePingDataIndexes() error {
	resultCollection := MG.CC("prob", "ping_result")
	optimizedCollection := MG.CC("prob", "ping_result_optimized")

	// Create index for ping_result collection
	_, err := resultCollection.Indexes().CreateOne(
		context.TODO(),
		mongo.IndexModel{
			Keys: bson.D{
				{Name: "time", Value: 1},
				{Name: "id", Value: 1},
			},
			Options: options.Index().SetName("time_id_index"),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create index for ping_result: %v", err)
	}

	// Create index for ping_result_optimized collection
	_, err = optimizedCollection.Indexes().CreateOne(
		context.TODO(),
		mongo.IndexModel{
			Keys: bson.D{
				{Name: "time", Value: 1},
				{Name: "id", Value: 1},
			},
			Options: options.Index().SetName("time_id_index"),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create index for ping_result_optimized: %v", err)
	}

	return nil
}

type PeriodType int

const (
	Day PeriodType = iota
	Week
	Month
	Year
)

type Period struct {
	Duration time.Duration
	Interval time.Duration
}

var periodMap = map[PeriodType]Period{
	Day:   {Duration: 24 * time.Hour, Interval: 5 * time.Minute},
	Week:  {Duration: 7 * 24 * time.Hour, Interval: 1 * time.Hour},
	Month: {Duration: 30 * 24 * time.Hour, Interval: 6 * time.Hour},
	Year:  {Duration: 365 * 24 * time.Hour, Interval: 24 * time.Hour},
}

type RetPingData struct {
	Time   time.Time `bson:"time"`
	Number float64   `bson:"number"`
}

type NodePingData struct {
	NodeID string
	Data   []RetPingData
}

func GetAllNodesPingData(period PeriodType) ([]NodePingData, error) {
	collection := MG.CC("prob", "ping_result")
	now := time.Now()

	p, ok := periodMap[period]
	if !ok {
		return nil, fmt.Errorf("invalid period type: %v", period)
	}

	nodeIDs, err := getAllNodeIDs(collection.Collection)
	if err != nil {
		return nil, err
	}

	var result []NodePingData

	for _, nodeID := range nodeIDs {
		data, err := getDataForRange(collection.Collection, nodeID, now.Add(-p.Duration), now, p.Interval)
		if err != nil {
			return nil, err
		}

		nodePingData := NodePingData{
			NodeID: nodeID,
			Data:   data,
		}

		result = append(result, nodePingData)
	}

	return result, nil
}

func getAllNodeIDs(collection *mongo.Collection) ([]string, error) {
	pipeline := []bson.M{
		{"$group": bson.M{"_id": "$id"}},
		{"$project": bson.M{"_id": 0, "id": "$_id"}},
	}

	cursor, err := collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var results []struct {
		ID string `bson:"id"`
	}
	if err = cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}

	var nodeIDs []string
	for _, result := range results {
		nodeIDs = append(nodeIDs, result.ID)
	}

	return nodeIDs, nil
}

func getDataForRange(collection *mongo.Collection, nodeID string, startTime, endTime time.Time, interval time.Duration) ([]RetPingData, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"id":   nodeID,
				"time": bson.M{"$gte": startTime, "$lt": endTime},
			},
		},
		{
			"$group": bson.M{
				"_id": bson.M{
					"bucket": bson.M{
						"$toDate": bson.M{
							"$subtract": []interface{}{
								"$time",
								bson.M{"$mod": []interface{}{bson.M{"$toLong": "$time"}, interval.Milliseconds()}},
							},
						},
					},
				},
				"maxNumber": bson.M{"$max": "$number"},
				"maxTime":   bson.M{"$max": "$time"},
			},
		},
		{
			"$project": bson.M{
				"_id":    0,
				"time":   "$maxTime",
				"number": "$maxNumber",
			},
		},
		{
			"$sort": bson.M{"time": 1},
		},
	}

	cursor, err := collection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var results []RetPingData
	if err = cursor.All(context.TODO(), &results); err != nil {
		return nil, err
	}

	return results, nil
}
