package client

import (
	"context"
	"net/http"
	"server/db"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

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
}

var (
	mongoClient *mongo.Client
	database    string
	collection  string
)

func HandleDynamicReport(c *gin.Context) {
	var data ServerDynamicData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Check if the node is bound or unbound
	bind, err := checkNodeStatus(data.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check node status"})
		return
	}

	if !bind {
		// Bind the node
		if err := bindNode(data.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to bind node"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Node not found or invalid"})
		return
	}
	// 添加时间戳
	data.Timestamp = time.Now()

	// 插入数据到 MongoDB
	err = insertDynamicData(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data received and stored successfully"})
}

func insertDynamicData(data ServerDynamicData) error {
	collection := db.MG.CC("vps", "dynamic")
	_, err := collection.InsertOne(context.TODO(), data)
	return err
}

func HandleStaticReport(c *gin.Context) {
	var data ServerStaticData
	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if the node is bound or unbound
	bind, err := checkNodeStatus(data.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check node status"})
		return
	}

	if !bind {
		// Bind the node
		if err := bindNode(data.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to bind node"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Node not found or invalid"})
		return
	}

	// 添加或更新最后报告时间
	data.LastReportTime = time.Now()

	// 插入或更新数据到 MongoDB
	err = upsertStaticData(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upsert data"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Static data received and stored successfully"})
}

// New function to check node status
func checkNodeStatus(nodeID string) (bool, error) {
	cc := db.MG.CC("prob", "node")
	var result struct {
		Bound bool `bson:"bound"`
	}
	err := cc.FindOne(context.TODO(), bson.M{"token": nodeID}).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}
	return result.Bound, nil
}

// New function to bind a node
func bindNode(nodeID string) error {
	cc := db.MG.CC("prob", "node")
	_, err := cc.UpdateOne(
		context.TODO(),
		bson.M{"token": nodeID},
		bson.M{"$set": bson.M{"status": true}},
	)
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
