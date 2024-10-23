package web

import (
	"context"
	"net/http"
	"server/db"
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
	ID   string `bson:"_id,omitempty" json:"id"`
	Name string `bson:"name" json:"name"`
	IP   string `bson:"ip" json:"ip"`
}

func ProbeList(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.MG.CC("prob", "probes")

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取探针列表失败"})
		return
	}
	defer cursor.Close(ctx)

	var probes []Probe = make([]Probe, 0)
	if err = cursor.All(ctx, &probes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "解析探针数据失败"})
		return
	}

	c.JSON(http.StatusOK, probes)
}

func ProbeAdd(c *gin.Context) {
	var newProbe Probe
	if err := c.ShouldBindJSON(&newProbe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.MG.CC("prob", "probes")

	result, err := collection.InsertOne(ctx, newProbe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加探针失败"})
		return
	}

	newProbe.ID = result.InsertedID.(primitive.ObjectID).Hex()
	c.JSON(http.StatusCreated, newProbe)
}

func ProbeDelete(c *gin.Context) {
	id := c.Param("id")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	collection := db.MG.CC("prob", "probes")

	result, err := collection.DeleteOne(ctx, bson.M{"id": id})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除探针失败"})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到指定探针"})
		return
	}

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
