package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	MAIN_SERVICE_URL = "http://192.168.0.5:8000/api/consumption-calc/result/"
	SECRET_TOKEN     = "12345678"
)

type DeviceData struct {
	ID          int     `json:"id"`
	Consumption float64 `json:"consumption"`
}

type DeviceInRequest struct {
	Device   DeviceData `json:"device"`
	Quantity int        `json:"quantity"`
}

type RequestData struct {
	RequestID  int                `json:"request_id"`
	Residents  int                `json:"residents"`
	Temperature int               `json:"temperature"`
	Devices    []DeviceInRequest  `json:"devices"`
}

type ResultData struct {
	Token     string `json:"token"`
	RequestID int    `json:"request_id"`
	Result    int    `json:"result"`
}

func calculateResult(data RequestData) int {
	baseConsumption := 0.0
	for _, deviceInRequest := range data.Devices {
		baseConsumption += deviceInRequest.Device.Consumption * float64(deviceInRequest.Quantity)
	}

	tempDiff := 20 - data.Temperature
	if tempDiff < 0 {
		tempDiff = -tempDiff
	}
	temperatureEffect := float64(tempDiff) * 0.01 * baseConsumption

	residentsEffect := float64(data.Residents-1) * 0.3 * baseConsumption

	totalConsumption := baseConsumption + temperatureEffect + residentsEffect

	rand.Seed(time.Now().UnixNano())
	randomFactor := 0.9 + rand.Float64()*0.2
	
	return int(totalConsumption * randomFactor)
}

func sendResult(requestID int, result int) error {
	resultData := ResultData{
		Token:     SECRET_TOKEN,
		RequestID: requestID,
		Result:    result,
	}

	jsonData, err := json.Marshal(resultData)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s%d/", MAIN_SERVICE_URL, requestID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func processRequest(data RequestData) {
	time.Sleep(5 * time.Second)

	result := calculateResult(data)

	err := sendResult(data.RequestID, result)
	if err != nil {
		fmt.Printf("Error sending result for request %d: %v\n", data.RequestID, err)
		return
	}

	fmt.Printf("Result sent successfully for request %d: %d\n", data.RequestID, result)
}

func main() {
	router := gin.Default()

	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	router.POST("/api/calculate", func(c *gin.Context) {
		var data RequestData
		if err := c.ShouldBindJSON(&data); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
			return
		}

		if data.RequestID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "request_id is required"})
			return
		}

		go processRequest(data)

		c.JSON(http.StatusOK, gin.H{"status": "ok", "message": "Calculation started"})
	})

	fmt.Println("Async service started on :8080")
	router.Run(":8080")
}

