package web

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type ProbeLatencyData struct {
	Name string    `json:"name"`
	Data []float64 `json:"data"`
}

type ProbeLatencyResponse struct {
	Probes []ProbeLatencyData `json:"probes"`
	Times  []string           `json:"times"`
}

func GetProbeLatency(c *gin.Context) {
	//serverID := c.Param("id")
	//rangeType := c.DefaultQuery("range", "day")

	// 这里应该是从数据库获取数据的逻辑
	// 现在我们用模拟数据代替
	//response := generateMockLatencyData(serverID, rangeType)

	c.JSON(http.StatusOK, ProbeLatencyResponse{})
}

func generateMockLatencyData(id, rangeType string) ProbeLatencyResponse {
	probes := []string{"Probe1", "Probe2", "Probe3"}
	var times []string
	var probeData []ProbeLatencyData

	now := time.Now()
	var dataPoints int
	var interval time.Duration

	switch rangeType {
	case "day":
		dataPoints = 24
		interval = time.Hour
	case "month":
		dataPoints = 30
		interval = 24 * time.Hour
	case "year":
		dataPoints = 12
		interval = 30 * 24 * time.Hour
	default:
		dataPoints = 24
		interval = time.Hour
	}

	for _, probe := range probes {
		data := make([]float64, dataPoints)
		for i := range data {
			data[i] = 50 + rand.Float64()*100 // Random latency between 50-150ms
		}
		probeData = append(probeData, ProbeLatencyData{Name: probe, Data: data})
	}

	for i := dataPoints - 1; i >= 0; i-- {
		t := now.Add(-time.Duration(i) * interval)
		times = append(times, t.Format("2006-01-02 15:04:05"))
	}

	return ProbeLatencyResponse{
		Probes: probeData,
		Times:  times,
	}
}
