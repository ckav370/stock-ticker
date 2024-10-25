package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func initializeRedisClient() *redis.Client {
	log.Println("Initializing Redis client...")
	client := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: "",
		DB:       0,
	})
	log.Println("Redis client initialized")
	return client
}

type TimeSeriesData struct {
	Date       string  `json:"date"`
	ClosePrice float64 `json:"close"`
}

type TimeSeriesResponse struct {
	TimeSeries map[string]struct {
		Close string `json:"4. close"`
	} `json:"Time Series (Daily)"`
}

// getStockData fetches stock data from Alpha Vantage
func getStockData(symbol string, apiKey string) (*TimeSeriesResponse, error) {
	log.Printf("Fetching stock data for symbol: %s", symbol)
	url := "https://www.alphavantage.co/query?apikey=" + apiKey + "&function=TIME_SERIES_DAILY&symbol=" + symbol
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching stock data: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var data TimeSeriesResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("Error decoding stock data response: %v", err)
		return nil, err
	}
	if len(data.TimeSeries) == 0 {
		log.Printf("No data available for symbol: %s", symbol)
		return nil, errors.New("no data available for symbol")
	}
	log.Printf("Stock data fetched successfully for symbol: %s", symbol)
	return &data, nil
}

// getCachedStockData attempts to retrieve stock data from Redis
func getCachedStockData(redisClient *redis.Client, symbol string) (*TimeSeriesResponse, error) {
	log.Printf("Checking Redis cache for symbol: %s", symbol)
	cachedData, err := redisClient.Get(ctx, symbol).Result()
	if err == redis.Nil {
		log.Println("Cache miss")
		return nil, nil // Cache miss
	} else if err != nil {
		log.Printf("Error retrieving cache data: %v", err)
		return nil, err
	}
	log.Println("Cache hit")

	var data TimeSeriesResponse
	if err := json.Unmarshal([]byte(cachedData), &data); err != nil {
		log.Printf("Error unmarshalling cached data: %v", err)
		return nil, err
	}
	return &data, nil
}

// cacheStockData caches the stock data in Redis
func cacheStockData(redisClient *redis.Client, symbol string, data *TimeSeriesResponse) error {
	log.Printf("Caching data for symbol: %s", symbol)
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling data for caching: %v", err)
		return err
	}
	if err := redisClient.Set(ctx, symbol, jsonData, time.Hour).Err(); err != nil {
		log.Printf("Error setting cache for symbol: %s, error: %v", symbol, err)
		return err
	}
	log.Printf("Data cached successfully for symbol: %s", symbol)
	return nil
}

// calculateAverage calculates the average closing price
func calculateAverage(data []TimeSeriesData) float64 {
	if len(data) == 0 {
		log.Println("No data available for average calculation")
		return 0.0
	}
	var total float64
	for _, entry := range data {
		total += entry.ClosePrice
	}
	return total / float64(len(data))
}

// stockHandler handles HTTP requests
func stockHandler(redisClient *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Handling /stock request")
		symbol := os.Getenv("SYMBOL")
		if symbol == "" {
			log.Println("SYMBOL environment variable not set")
			http.Error(w, "SYMBOL environment variable not set", http.StatusInternalServerError)
			return
		}

		ndaysStr := os.Getenv("NDAYS")
		ndays, err := strconv.Atoi(ndaysStr)
		if err != nil || ndays <= 0 {
			log.Println("Invalid NDAYS environment variable")
			http.Error(w, "Invalid NDAYS environment variable", http.StatusBadRequest)
			return
		}

		apiKey := os.Getenv("API_KEY")
		if apiKey == "" {
			log.Println("API key not set in API_KEY environment variable")
			http.Error(w, "API key not set in API_KEY environment variable", http.StatusInternalServerError)
			return
		}

		// Check Redis cache
		stockData, err := getCachedStockData(redisClient, symbol)
		if err != nil {
			log.Printf("Error checking cache: %v", err)
			http.Error(w, "Error checking cache: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if stockData == nil {
			// Fetch from Alpha Vantage and cache
			log.Printf("No cache available, fetching data for symbol: %s", symbol)
			stockData, err = getStockData(symbol, apiKey)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if err := cacheStockData(redisClient, symbol, stockData); err != nil {
				log.Printf("Error caching data for symbol %s: %v", symbol, err)
			}
		}

		// Collect all data into a slice, with dates sorted in descending order
		var closingPrices []TimeSeriesData
		for date, values := range stockData.TimeSeries {
			price, parseErr := strconv.ParseFloat(values.Close, 64)
			if parseErr != nil {
				log.Printf("Error parsing close price for date %s: %v", date, parseErr)
				continue
			}
			closingPrices = append(closingPrices, TimeSeriesData{
				Date:       date,
				ClosePrice: price,
			})
		}

		// Sort by date in descending order to ensure the latest dates are first
		sort.Slice(closingPrices, func(i, j int) bool {
			return closingPrices[i].Date > closingPrices[j].Date
		})

		// Get only the latest `NDAYS` entries
		if len(closingPrices) > ndays {
			closingPrices = closingPrices[:ndays]
		}

		// Calculate the average closing price over the last `NDAYS` entries
		avg := calculateAverage(closingPrices)
		log.Printf("Calculated average price for last %d days: %f", ndays, avg)

		// Prepare the JSON response with `closing_prices` and `average_price`
		response := map[string]interface{}{
			"closing_prices": closingPrices,
			"average_price":  avg,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Println("Failed to encode response")
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		} else {
			log.Println("Response successfully sent")
		}
	}
}

func main() {
	log.Println("Starting server...")
	redisClient := initializeRedisClient() 
	defer redisClient.Close()

	http.HandleFunc("/stock", stockHandler(redisClient))
	port := ":8080"
	log.Printf("Server running on port %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
