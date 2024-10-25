package main

import (
    "encoding/json"
    "log"
    "net/http"
    "os"
    "strconv"
)

const apiKey = ""

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
func getStockData(symbol string) (*TimeSeriesResponse, error) {
    url := "https://www.alphavantage.co/query?apikey=" + apiKey + "&function=TIME_SERIES_DAILY&symbol=" + symbol
    resp, err := http.Get(url)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var data TimeSeriesResponse
    if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
        return nil, err
    }
    return &data, nil
}

// calculateAverage calculates the average closing price
func calculateAverage(data []TimeSeriesData) float64 {
    var total float64
    for _, entry := range data {
        total += entry.ClosePrice
    }
    return total / float64(len(data))
}

// stockHandler handles the HTTP requests
func stockHandler(w http.ResponseWriter, r *http.Request) {
    symbol := os.Getenv("STOCK_SYMBOL")
    ndays, err := strconv.Atoi(os.Getenv("NDAYS"))
    if err != nil || ndays <= 0 {
        http.Error(w, "Invalid NDAYS", http.StatusBadRequest)
        return
    }

    stockData, err := getStockData(symbol)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    var closingPrices []TimeSeriesData
    count := 0
    for date, values := range stockData.TimeSeries {
        if count >= ndays {
            break
        }
        price, _ := strconv.ParseFloat(values.Close, 64)
        closingPrices = append(closingPrices, TimeSeriesData{
            Date:       date,
            ClosePrice: price,
        })
        count++
    }

    avg := calculateAverage(closingPrices)
    response := map[string]interface{}{
        "closing_prices": closingPrices,
        "average_price":  avg,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func main() {
    http.HandleFunc("/stock", stockHandler)
    port := ":8080"
    log.Printf("Server starting on port %s...", port)
    log.Fatal(http.ListenAndServe(port, nil))
}
