package web

import (
	"context"
	"fmt"
	"net/http"
	"server/db"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Node struct {
	ID        string    `bson:"_id,omitempty"`
	Token     string    `bson:"token"`
	CreatedAt time.Time `bson:"createdAt"`
	Bound     bool      `bson:"bound"`
	// 其他节点信息字段
}

type Probe struct {
	ID   string `bson:"id,omitempty" json:"id,omitempty"`
	Name string `bson:"name" json:"name"`
	IP   string `bson:"ip" json:"ip"`
}

func ProbeList(c *gin.Context) {
	probes, err := db.GetPingConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取探针列表失败"})
		return
	}
	fmt.Println("get probes: ", probes.Nodes, err)
	if probes.Nodes == nil {
		probes.Nodes = []db.PingNode{}
	}
	// data, _ := json.Marshal(probes.Nodes)
	c.JSON(http.StatusOK, probes.Nodes)
}

func ProbeAdd(c *gin.Context) {
	var newProbe Probe
	if err := c.ShouldBindJSON(&newProbe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}
	node, err := db.AddPingNode(newProbe.Name, newProbe.IP)
	if err != nil {
		fmt.Println("添加探针失败: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加探针失败"})
		return
	}
	c.JSON(http.StatusCreated, node)
}

func ProbeDelete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}
	//convert id to int
	idInt, err := strconv.Atoi(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"error": "无效的探针ID"})
		return
	}
	db.RemovePingNode(db.PingNode{ID: idInt})
	c.JSON(http.StatusOK, gin.H{"message": "探针删除成功"})
}

func DeleteNode(c *gin.Context) {
	// 从URL参数中获取节点ID
	nodeID := c.Param("id")

	// 验证nodeID是否为有效的ObjectID
	_, err := primitive.ObjectIDFromHex(nodeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid node ID"})
		return
	}

	// 获取MongoDB集合
	collection := db.MG.CC("vps", "nodes")

	// 执行删除操作
	result, err := collection.DeleteOne(context.TODO(), bson.M{"_id": nodeID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete node"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Node deleted successfully"})
}
