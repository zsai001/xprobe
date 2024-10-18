package web

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"net/http"
	db2 "server/db"
	"time"
)

type Setting struct {
	Display   DisplaySetting `json:"display"`
	Add       AddSetting     `json:"add"`
	About     AboutSetting   `json:"about"`
	UpdatedAt time.Time      `bson:"updatedAt" json:"updatedAt"`
}

type MenuItem struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type AboutSetting struct {
	SiteTitle string     `json:"siteTitle"`
	MenuItems []MenuItem `json:"menuItems"`
}

type AddSetting struct {
	Windows string `bson:"windows" json:"windows"`
	Linux   string `bson:"linux" json:"linux"`
	MacOS   string `bson:"macos" json:"macos"`
}

type DisplaySetting struct {
	IsGrouped bool   `json:"isGrouped"`
	GroupBy   string `json:"groupBy"`
	IsSorted  bool   `json:"isSorted"`
	SortBy    string `json:"sortBy"`
	SortOrder string `json:"sortOrder"`
}

func getDefaultSettings(c *gin.Context) Setting {
	return Setting{
		Display: DisplaySetting{
			IsGrouped: false,
			GroupBy:   "status",
			IsSorted:  true,
			SortBy:    "name",
			SortOrder: "asc",
		},
		About: AboutSetting{
			SiteTitle: "XProb",
			MenuItems: []MenuItem{},
		},
		UpdatedAt: time.Now(),
	}
}

// SettingGet 处理设置设置的请求
func SettingGet(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cc := db2.MG.CC("prob", "setting")
	// 从数据库获取最新的设置
	var setting Setting
	err := cc.FindOne(ctx, bson.M{}).Decode(&setting)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// 如果没有找到设置，返回默认设置
			setting = getDefaultSettings(c)
			setting.Add = GetAddSetting(c)
			c.JSON(http.StatusOK, setting)
			return
		}
		// 其他错误情况
		log.Printf("Error fetching settings: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch settings"})
		return
	}
	setting.Add = GetAddSetting(c)
	// 返回找到的设置
	c.JSON(http.StatusOK, setting)
}

func SettingSet(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var newSetting Setting
	if err := c.ShouldBindJSON(&newSetting); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// 验证 DisplaySettings
	if err := validateDisplaySettings(newSetting.Display); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置更新时间
	newSetting.UpdatedAt = time.Now()

	// 尝试更新现有文档，如果不存在则插入新文档
	opts := options.FindOneAndUpdate().SetUpsert(true)
	filter := bson.M{} // 空 filter 意味着更新或插入唯一的文档
	update := bson.M{"$set": newSetting}

	cc := db2.MG.CC("prob", "setting")

	var updatedSetting Setting
	err := cc.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedSetting)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// 这意味着进行了插入操作
			c.JSON(http.StatusCreated, gin.H{"message": "Settings created successfully", "settings": newSetting})
		} else {
			log.Printf("Error updating settings: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Settings updated successfully", "settings": newSetting})
}

// validateDisplaySettings 验证显示设置的有效性
func validateDisplaySettings(ds DisplaySetting) error {
	// 这里可以添加更复杂的验证逻辑
	validGroupBy := map[string]bool{"status": true, "vendor": true, "country": true}
	validSortBy := map[string]bool{"name": true, "status": true, "uptime": true}
	validSortOrder := map[string]bool{"asc": true, "desc": true}

	if ds.IsGrouped && !validGroupBy[ds.GroupBy] {
		return fmt.Errorf("invalid groupBy value")
	}
	if ds.IsSorted && (!validSortBy[ds.SortBy] || !validSortOrder[ds.SortOrder]) {
		return fmt.Errorf("invalid sortBy or sortOrder value")
	}
	return nil
}
