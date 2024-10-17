package web

import (
	"context"
	"net/http"
	"server/client"
	"server/db"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ServerData struct {
	Id              string     `json:"id"`
	ServerName      string     `json:"serverName"`
	AreaCode        string     `json:"areaCode"`
	AreaFlagUrl     string     `json:"areaFlagUrl"`
	OsIconUrl       string     `json:"osIconUrl"`
	OsName          string     `json:"OsName"`
	Vendor          string     `json:"vendor"`
	VendorIconUrl   string     `json:"vendorIconUrl"`
	CpuUsed         int        `json:"cpuUsed"`
	CpuTotal        int        `json:"cpuTotal"`
	MemoryUsed      int        `json:"memoryUsed"`
	MemoryTotal     int        `json:"memoryTotal"`
	DiskUsed        int        `json:"diskUsed"`
	DiskTotal       int        `json:"diskTotal"`
	SwapUsed        int        `json:"swapUsed"`
	SwapTotal       int        `json:"swapTotal"`
	NetDownload     int        `json:"netDownload"`
	NetUpload       int        `json:"netUpload"`
	TrafficDownload int        `json:"trafficDownload"`
	TrafficUpload   int        `json:"trafficUpload"`
	Load            [3]float32 `json:"load"`
	TcpCount        int        `json:"tcpCount"`
	UdpCount        int        `json:"udpCount"`
	ProcessCount    int        `json:"processCount"`
	ThreadCount     int        `json:"threadCount"`
	OnlineDuration  int        `json:"onlineDuration"`
	MonthlyTraffic  int        `json:"monthlyTraffic"`
	TotalTraffic    int        `json:"totalTraffic"`
	OnlineStatus    string     `json:"onlineStatus"`
	Ipv4Supported   bool       `json:"ipv4Supported"`
	Ipv6Supported   bool       `json:"ipv6Supported"`
}

func getUniqueServerIDs(collection *mongo.Collection) ([]string, error) {
	// 使用 distinct 命令获取所有唯一的 server_id
	ids, err := collection.Distinct(context.TODO(), "id", bson.M{})
	if err != nil {
		return nil, err
	}

	// 将 interface{} 类型转换为 []string
	serverIDs := make([]string, len(ids))
	for i, id := range ids {
		serverIDs[i] = id.(string)
	}

	return serverIDs, nil
}

func getServerDataFromMongo() ([]ServerData, error) {
	staticCollection := db.MG.CC("vps", "static")
	dynamicCollection := db.MG.CC("vps", "dynamic")

	// 获取所有唯一的 server_id
	serverIDs, err := getUniqueServerIDs(staticCollection.Collection)
	if err != nil {
		return nil, err
	}

	var serverDataList []ServerData

	for _, id := range serverIDs {
		// 获取最新的静态数据
		var staticData client.ServerStaticData
		err := staticCollection.FindOne(
			context.TODO(),
			bson.M{"id": id},
		).Decode(&staticData)
		if err != nil {
			continue // 如果没有找到静态数据，跳过这个服务器
		}

		// 获取最新的动态数据
		var dynamicData client.ServerDynamicData
		err = dynamicCollection.FindOne(
			context.TODO(),
			bson.M{"id": id},
			options.FindOne().SetSort(bson.M{"timestamp": -1}),
		).Decode(&dynamicData)
		if err != nil {
			continue // 如果没有找到动态数据，跳过这个服务器
		}

		serverData := ServerData{
			Id:              id,
			ServerName:      staticData.HostName,
			AreaCode:        staticData.CountryCode,
			OsName:          staticData.OSName,
			Vendor:          staticData.VendorName,
			CpuUsed:         int(dynamicData.CPUUsage),
			CpuTotal:        100, // 假设CPU总量为100%
			MemoryUsed:      int(dynamicData.MemoryUsed),
			MemoryTotal:     parseMemoryTotal(staticData.MemoryTotal),
			DiskUsed:        int(dynamicData.DiskUsed),
			DiskTotal:       parseDiskTotal(staticData.DiskTotal),
			SwapUsed:        0, // 需要添加到动态数据中
			SwapTotal:       parseSwapTotal(staticData.SwapTotal),
			NetDownload:     int(dynamicData.NetworkDownload),
			NetUpload:       int(dynamicData.NetworkUpload),
			TrafficDownload: int(dynamicData.TrafficDownload),
			TrafficUpload:   int(dynamicData.TrafficUpload),
			Load:            [3]float32{float32(dynamicData.Load[0]), float32(dynamicData.Load[1]), float32(dynamicData.Load[2])},
			TcpCount:        dynamicData.TCPCount,
			UdpCount:        dynamicData.UDPCount,
			ProcessCount:    dynamicData.ProcessCount,
			ThreadCount:     dynamicData.ThreadCount,
			OnlineDuration:  int(time.Since(staticData.LastReportTime).Seconds()),
			OnlineStatus:    "online", // 假设所有报告的服务器都在线
			Ipv4Supported:   staticData.IPv4Supported,
			Ipv6Supported:   staticData.IPv6Supported,
			// 其他字段可以根据需要添加或修改
		}

		serverDataList = append(serverDataList, serverData)
	}

	return serverDataList, nil
}

func parseMemoryTotal(memoryTotal string) int {
	return parseSize(memoryTotal)
}

func parseDiskTotal(diskTotal string) int {
	return parseSize(diskTotal)
}

func parseSwapTotal(swapTotal string) int {
	return parseSize(swapTotal)
}

func parseSize(size string) int {
	size = strings.TrimSpace(size)
	if size == "" {
		return 0
	}

	// 将大小转换为小写以统一处理
	size = strings.ToLower(size)

	// 分离数字和单位
	var numStr string
	var unit string
	for i, c := range size {
		if c < '0' || c > '9' {
			numStr = size[:i]
			unit = size[i:]
			break
		}
	}

	// 如果没有找到单位，假设整个字符串都是数字
	if unit == "" {
		numStr = size
	}

	// 解析数字部分
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	// 根据单位转换为字节
	switch unit {
	case "b", "bytes":
		return int(num)
	case "k", "kb", "kib":
		return int(num * 1024)
	case "m", "mb", "mib":
		return int(num * 1024 * 1024)
	case "g", "gb", "gib":
		return int(num * 1024 * 1024 * 1024)
	case "t", "tb", "tib":
		return int(num * 1024 * 1024 * 1024 * 1024)
	default:
		return int(num)
	}
}

func Status(c *gin.Context) {
	serverDataList, err := getServerDataFromMongo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve server data"})
		return
	}

	c.JSON(http.StatusOK, serverDataList)
}
