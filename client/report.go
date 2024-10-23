package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"server/db"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

var Version = "1.0.1"

type ServerDynamicData struct {
	ID              string     `json:"id" bson:"id"`
	Load            [3]float64 `json:"load" bson:"load"`
	CPUUsage        float64    `json:"cpuUsage" bson:"cpuUsage"`
	MemoryUsed      uint64     `json:"memoryUsed" bson:"memoryUsed"`
	DiskUsed        uint64     `json:"diskUsed" bson:"diskUsed"`
	NetworkDownload uint64     `json:"networkDownload" bson:"networkDownload"`
	NetworkUpload   uint64     `json:"networkUpload" bson:"networkUpload"`
	TrafficDownload uint64     `json:"trafficDownload" bson:"trafficDownload"`
	TrafficUpload   uint64     `json:"trafficUpload" bson:"trafficUpload"`
	TCPCount        int        `json:"tcpCount" bson:"tcpCount"`
	UDPCount        int        `json:"udpCount" bson:"udpCount"`
	ProcessCount    int        `json:"processCount" bson:"processCount"`
	ThreadCount     int        `json:"threadCount" bson:"threadCount"`
	Timestamp       time.Time  `json:"timestamp" bson:"timestamp"`
}

type ServerStaticData struct {
	ID             string    `json:"id" bson:"id"`
	HostName       string    `json:"hostName" bson:"hostName"`
	OSName         string    `json:"osName" bson:"osName"`
	NAT            bool      `json:"nat" bson:"nat"`
	OSVersion      string    `json:"osVersion" bson:"osVersion"`
	Architecture   string    `json:"osArchitecture" bson:"osArchitecture"`
	Virtualization string    `json:"virtualization" bson:"virtualization"`
	PublicIPV4     string    `json:"publicIPV4" bson:"publicIPV4"`
	PublicIPV6     string    `json:"publicIPV6" bson:"publicIPV6"`
	Isp            string    `json:"isp" bson:"isp"`
	VendorName     string    `json:"vendorName" bson:"vendorName"`
	CountryCode    string    `json:"countryCode" bson:"countryCode"`
	IPv4Supported  bool      `json:"ipv4Supported" bson:"ipv4Supported"`
	IPv6Supported  bool      `json:"ipv6Supported" bson:"ipv6Supported"`
	SwapTotal      string    `json:"swapTotal" bson:"swapTotal"`
	MemoryTotal    string    `json:"memoryTotal" bson:"memoryTotal"`
	DiskTotal      string    `json:"diskTotal" bson:"diskTotal"`
	UpDateTime     string    `json:"upDateTime" bson:"upDateTime"`
	LastReportTime time.Time `json:"lastReportTime" bson:"lastReportTime"`
	Version        string    `json:"version" bson:"version"`
}

type Report map[string]any

var (
	mongoClient *mongo.Client
	database    string
	collection  string
)

type Action struct {
	Name   string `json:"name"`
	Topic  string `json:"topic"`
	Data   string `json:"data"`
	Pority int    `json:"pority"`
}

func HandleReport(c *gin.Context) {
	var report Report
	if err := c.ShouldBindJSON(&report); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// header["X-Node-Id"] = cfg.NodeID
	// header["X-Agent-Version"] = cfg.Version

	// nodeId := c.GetHeader("X-Node-Id")
	version := c.GetHeader("X-Agent-Version")

	actions := []*Action{}
	if version != Version {
		actions = append(actions, &Action{Name: "upgrade"})
	}

	action := HandleDynamicReport(c, report)
	if action != nil {
		actions = append(actions, action)
	}
	action = HandleStaticReport(c, report)
	if action != nil {
		actions = append(actions, action)
	}
	action = HandlePingReport(c, report)

	if action != nil {
		actions = append(actions, action)
	}
	if len(actions) > 0 {
		c.JSON(http.StatusOK, gin.H{"actions": actions})
		return
	}
	// log.Println("report", report)
	c.JSON(http.StatusOK, gin.H{"message": "Report received and stored successfully"})
}

type PingReport struct {
	Data    []db.PingData `json:"data"`
	Version string        `json:"version"`
}

func HandlePingReport(c *gin.Context, report Report) *Action {
	data := PingReport{}
	if content, ok := report["ping"]; ok {
		ConvertContent(content, &data)
		fmt.Println("ping report", data)
	} else {
		fmt.Println("no ping report")
		return nil
	}
	nodeId := c.GetHeader("X-Node-Id")
	fmt.Println("nodeId", nodeId)
	if nodeId == "" {
		return nil
	}
	if len(data.Data) != 0 {
		for i := range data.Data {
			data.Data[i].NodeId = nodeId
		}

		db.InsertPingResult(data.Data)
	}

	pingConfig, err := db.GetPingConfig()
	if err != nil {
		return nil
	}

	if pingConfig.Version != data.Version {
		cfg, err := db.GetPingConfig()
		if err != nil {
			return nil
		}
		data, _ := json.Marshal(cfg)
		return &Action{Name: "config", Topic: "ping", Data: string(data)}
	}
	return nil
}

func ConvertContent(content any, data interface{}) {
	tmp, _ := json.Marshal(content)
	json.Unmarshal(tmp, data)
}

func HandleDynamicReport(c *gin.Context, report Report) *Action {
	allData := []ServerDynamicData{}
	if content, ok := report["dynamic"]; ok {
		ConvertContent(content, &allData)
	} else {
		return nil
	}
	// allData, ok := content.([]ServerDynamicData)
	// if !ok {
	// 	return
	// }
	// Check if the node is bound or unbound
	// bind, err := checkNodeStatus(data.ID)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check node status"})
	// 	return
	// }
	// fmt.Println("check dynamic vps bind with", data.ID, bind)
	// if bind != "unbind" && bind != "bind" {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Node not found or invalid"})
	// 	return
	// }
	// if bind == "unbind" {
	// 	bindNode(data.ID)
	// }

	// 添加时间戳
	data := allData[0]
	data.Timestamp = time.Now()
	log.Println("report dynamic with", data)

	// 插入数据到 MongoDB
	err := upsertDynamicData(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert data"})
		return nil
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data received and stored successfully"})
	return nil
}

func upsertDynamicData(data ServerDynamicData) error {
	collection := db.MG.CC("vps", "dynamic")

	filter := bson.M{"id": data.ID}
	update := bson.M{"$set": data}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(context.TODO(), filter, update, opts)
	return err
}

func insertDynamicData(data ServerDynamicData) error {
	collection := db.MG.CC("vps", "dynamic")
	_, err := collection.InsertOne(context.TODO(), data)
	return err
}

func HandleStaticReport(c *gin.Context, report Report) *Action {
	allData := []ServerStaticData{}
	if content, ok := report["static"]; ok {
		ConvertContent(content, &allData)
	} else {
		return nil
	}
	data := allData[0]

	// Check if the node is bound or unbound
	// bind, err := checkNodeStatus(data.ID)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check node status"})
	// 	return
	// }

	// if bind != "unbind" && bind != "bind" {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Node not found or invalid"})
	// 	return
	// }

	// fmt.Println("check static vps bind with", data.ID, bind)

	// if bind == "unbind" {
	// 	bindNode(data.ID)
	// }

	// 添加或更新最后报告时间
	data.LastReportTime = time.Now()
	log.Println("report static with", data)
	// 插入或更新数据到 MongoDB
	err := upsertStaticData(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upsert data"})
		return nil
	}

	c.JSON(http.StatusOK, gin.H{"message": "Static data received and stored successfully"})
	return nil
}

// New function to check node status
func checkNodeStatus(nodeID string) (string, error) {
	cc := db.MG.CC("prob", "node")
	var result struct {
		Bound bool `bson:"bound"`
	}
	err := cc.FindOne(context.TODO(), bson.M{"token": nodeID}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "unknown", nil
		}
		return "error", err
	}
	if result.Bound {
		return "bind", nil
	} else {
		return "unbind", nil
	}
}

// New function to bind a node
func bindNode(nodeID string) error {
	cc := db.MG.CC("prob", "node")
	result, err := cc.UpdateOne(
		context.TODO(),
		bson.M{"token": nodeID},
		bson.M{"$set": bson.M{"bound": true}},
	)
	fmt.Println("bind node result", result, nodeID)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("Node not found or invalid")
	}
	return err
}

func upsertStaticData(data ServerStaticData) error {
	collection := db.MG.CC("vps", "static")

	filter := bson.M{"id": data.ID}
	update := bson.M{"$set": data}
	opts := options.Update().SetUpsert(true)

	_, err := collection.UpdateOne(context.TODO(), filter, update, opts)
	return err
}
