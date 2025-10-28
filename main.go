package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"github.com/joho/godotenv"
	"golang.org/x/image/font/gofont/goregular"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Country model
type Country struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	Name            string    `gorm:"uniqueIndex;not null" json:"name"`
	Capital         string    `json:"capital"`
	Region          string    `json:"region"`
	Population      int64     `gorm:"not null" json:"population"`
	CurrencyCode    *string   `json:"currency_code"`
	ExchangeRate    *float64  `json:"exchange_rate"`
	EstimatedGDP    *float64  `json:"estimated_gdp"`
	FlagURL         string    `json:"flag_url"`
	LastRefreshedAt time.Time `json:"last_refreshed_at"`
}

// External API response structures
type RestCountry struct {
	Name       string              `json:"name"`
	Capital    string              `json:"capital"`
	Region     string              `json:"region"`
	Population int64               `json:"population"`
	Flag       string              `json:"flag"`
	Currencies []map[string]string `json:"currencies"`
}

type ExchangeRates struct {
	Rates map[string]float64 `json:"rates"`
}

var db *gorm.DB

func main() {
	// Load environment variables
	godotenv.Load()

	// Initialize database
	initDB()

	// Create cache directory
	os.MkdirAll("cache", 0755)

	// Setup Gin router
	r := gin.Default()

	// Routes
	r.POST("/countries/refresh", refreshCountries)
	r.GET("/countries", getCountries)
	r.GET("/countries/image", getCountryImage)
	r.GET("/countries/:name", getCountry)
	r.DELETE("/countries/:name", deleteCountry)
	r.GET("/status", getStatus)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}

func formatDatabaseURL(url string) string {
	// If URL starts with mysql://, convert it to GORM format
	if strings.HasPrefix(url, "mysql://") {
		// Remove mysql:// prefix
		url = strings.TrimPrefix(url, "mysql://")

		// Split into user:pass and host:port/db
		parts := strings.Split(url, "@")
		if len(parts) != 2 {
			return url
		}

		userPass := parts[0]
		hostDB := parts[1]

		// Format: user:pass@tcp(host:port)/db?params
		return fmt.Sprintf("%s@tcp(%s)?charset=utf8mb4&parseTime=True&loc=Local", userPass, hostDB)
	}
	return url
}

func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		// Try POSTGRES_URL as fallback
		dsn = os.Getenv("POSTGRES_URL")
	}
	if dsn == "" {
		log.Fatal("DATABASE_URL or POSTGRES_URL environment variable is required")
	}

	// For PostgreSQL, the DSN can be used as is
	dsn = formatDatabaseURL(dsn)

	var err error
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate
	db.AutoMigrate(&Country{})
}

func refreshCountries(c *gin.Context) {
	// Fetch countries
	countries, err := fetchCountries()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "External data source unavailable",
			"details": "Could not fetch data from restcountries.com",
		})
		return
	}

	// Fetch exchange rates
	rates, err := fetchExchangeRates()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "External data source unavailable",
			"details": "Could not fetch data from open.er-api.com",
		})
		return
	}

	now := time.Now()

	// Process and save countries
	for _, rc := range countries {
		country := Country{
			Name:            rc.Name,
			Capital:         rc.Capital,
			Region:          rc.Region,
			Population:      rc.Population,
			FlagURL:         rc.Flag,
			LastRefreshedAt: now,
		}

		// Handle currency
		if len(rc.Currencies) > 0 && rc.Currencies[0] != nil {
			if code, ok := rc.Currencies[0]["code"]; ok && code != "" {
				country.CurrencyCode = &code
			}
		}

		// Get exchange rate if currency code exists
		if country.CurrencyCode != nil {
			if rate, ok := rates[*country.CurrencyCode]; ok {
				country.ExchangeRate = &rate

				// Calculate estimated GDP
				multiplier := rand.Float64()*(2000-1000) + 1000
				gdp := float64(country.Population) * multiplier / rate
				country.EstimatedGDP = &gdp
			} else {
				// Rate not found, exchange_rate null (already nil), estimated_gdp null
			}
		} else {
			// No currency, set estimated_gdp to 0
			zero := 0.0
			country.EstimatedGDP = &zero
		}

		// Update or create
		var existing Country
		result := db.Where("LOWER(name) = LOWER(?)", country.Name).First(&existing)
		if result.Error == nil {
			// Update existing
			country.ID = existing.ID
			db.Save(&country)
		} else {
			// Create new
			db.Create(&country)
		}
	}

	// Generate summary image
	if err := generateSummaryImage(); err != nil {
		log.Printf("Failed to generate image: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":           "Countries refreshed successfully",
		"last_refreshed_at": now,
	})
}

func getCountries(c *gin.Context) {
	var countries []Country
	query := db

	// Filters
	if region := c.Query("region"); region != "" {
		query = query.Where("region = ?", region)
	}
	if currency := c.Query("currency"); currency != "" {
		query = query.Where("currency_code = ?", currency)
	}

	// Sorting
	sort := c.Query("sort")
	switch sort {
	case "gdp_desc":
		query = query.Order("estimated_gdp DESC NULLS LAST")
	case "gdp_asc":
		query = query.Order("estimated_gdp ASC NULLS FIRST")
	case "population_desc":
		query = query.Order("population DESC")
	case "population_asc":
		query = query.Order("population ASC")
	default:
		query = query.Order("name ASC")
	}

	query.Find(&countries)
	c.JSON(http.StatusOK, countries)
}

func getCountry(c *gin.Context) {
	name := c.Param("name")
	var country Country

	if err := db.Where("LOWER(name) = LOWER(?)", name).First(&country).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Country not found"})
		return
	}

	c.JSON(http.StatusOK, country)
}

func deleteCountry(c *gin.Context) {
	name := c.Param("name")
	var country Country

	if err := db.Where("LOWER(name) = LOWER(?)", name).First(&country).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Country not found"})
		return
	}

	db.Delete(&country)
	c.JSON(http.StatusOK, gin.H{"message": "Country deleted successfully"})
}

func getStatus(c *gin.Context) {
	var count int64
	var lastRefresh time.Time

	db.Model(&Country{}).Count(&count)
	db.Model(&Country{}).Select("COALESCE(MAX(last_refreshed_at), '0001-01-01T00:00:00Z')").Scan(&lastRefresh)

	c.JSON(http.StatusOK, gin.H{
		"total_countries":   count,
		"last_refreshed_at": lastRefresh,
	})
}

func getCountryImage(c *gin.Context) {
	if _, err := os.Stat("cache/summary.png"); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Summary image not found"})
		return
	}

	c.File("cache/summary.png")
}

func fetchCountries() ([]RestCountry, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://restcountries.com/v2/all?fields=name,capital,region,population,flag,currencies")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var countries []RestCountry
	if err := json.NewDecoder(resp.Body).Decode(&countries); err != nil {
		return nil, err
	}

	return countries, nil
}

func fetchExchangeRates() (map[string]float64, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://open.er-api.com/v6/latest/USD")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var rates ExchangeRates
	if err := json.NewDecoder(resp.Body).Decode(&rates); err != nil {
		return nil, err
	}

	return rates.Rates, nil
}

func generateSummaryImage() error {
	// Get total countries
	var totalCountries int64
	db.Model(&Country{}).Count(&totalCountries)

	// Get top 5 by GDP
	var topCountries []Country
	db.Where("estimated_gdp IS NOT NULL").
		Order("estimated_gdp DESC").
		Limit(5).
		Find(&topCountries)

	// Get last refresh time
	var lastRefresh time.Time
	db.Model(&Country{}).Select("COALESCE(MAX(last_refreshed_at), '0001-01-01T00:00:00Z')").Scan(&lastRefresh)

	// Create image
	img := image.NewRGBA(image.Rect(0, 0, 800, 600))

	// Fill background
	for y := 0; y < 600; y++ {
		for x := 0; x < 800; x++ {
			img.Set(x, y, color.RGBA{240, 248, 255, 255})
		}
	}

	// Load font
	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		return err
	}

	c := freetype.NewContext()
	c.SetDPI(72)
	c.SetFont(font)
	c.SetFontSize(24)
	c.SetClip(img.Bounds())
	c.SetDst(img)
	c.SetSrc(image.NewUniform(color.RGBA{0, 0, 0, 255}))

	// Draw title
	pt := freetype.Pt(50, 80)
	c.DrawString("Country Data Summary", pt)

	// Draw total countries
	c.SetFontSize(18)
	pt = freetype.Pt(50, 140)
	c.DrawString(fmt.Sprintf("Total Countries: %d", totalCountries), pt)

	// Draw top 5 countries
	pt = freetype.Pt(50, 200)
	c.DrawString("Top 5 Countries by Estimated GDP:", pt)

	c.SetFontSize(14)
	y := 240
	for i, country := range topCountries {
		pt = freetype.Pt(70, y)
		gdp := "N/A"
		if country.EstimatedGDP != nil {
			gdp = fmt.Sprintf("$%.2f", *country.EstimatedGDP)
		}
		c.DrawString(fmt.Sprintf("%d. %s - %s", i+1, country.Name, gdp), pt)
		y += 40
	}

	// Draw timestamp
	c.SetFontSize(16)
	pt = freetype.Pt(50, 500)
	c.DrawString(fmt.Sprintf("Last Refreshed: %s", lastRefresh.Format(time.RFC3339)), pt)

	// Save image
	file, err := os.Create("cache/summary.png")
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}
