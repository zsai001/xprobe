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
