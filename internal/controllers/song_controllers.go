package controllers

import (
	"effectiveMobileTask/config"
	"effectiveMobileTask/internal/models"
	"effectiveMobileTask/internal/storage/database"
	"effectiveMobileTask/lib/logger"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type SongEnriched struct {
	ReleaseDate string `json:"release_date"`
	Group       string `json:"group"`
	Song        string `json:"song"`
	Text        string `json:"text"`
	Link        string `json:"link"`
}

type songRequest struct {
	Group string `json:"group" example:"Muse"` // Пример значения
	Title string `json:"song" example:"Supermassive Black Hole"`
}

// AddSongInfo godoc
// @Summary Get song information
// @Description Retrieve song information by group and title
// @Tags Songs
// @Accept json
// @Produce json
// @Param request body songRequest true "Request Body"
// @Success 200 {object} models.SongDetail "Song details successfully retrieved"
// @Failure 400 {object} map[string]string "Bad request - missing or invalid parameters"
// @Failure 404 {object} map[string]string "Song not found"
// @Failure 500 {object} map[string]string "Internal server error - database or API error"
// @Router /info [post]
func AddSongInfo(c *gin.Context) {
	var requestBody songRequest

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		logger.Error("invalid request body", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	groupName := requestBody.Group
	songTitle := requestBody.Title

	db := database.DbConnect()

	// Find or create the artist
	var artist models.Artist
	if err := db.Where("name = ?", groupName).FirstOrCreate(&artist, models.Artist{Name: groupName}).Error; err != nil {
		logger.Error("failed to find or create artist", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		return
	}

	// Find the song
	var song models.Song
	if err := db.Where("artist_id = ? AND title = ?", artist.ID, songTitle).First(&song).Error; err != nil {
		logger.Info("song not found", slog.Any("params", map[string]string{"group": groupName, "song": songTitle}))
		songDetail, boolReturn := GetSongDetailAPI(groupName, songTitle, c)
		if boolReturn {
			return
		}

		// Parse the release date
		releaseDate, err := time.Parse("02.01.2006", songDetail.ReleaseDate)
		if err != nil {
			logger.Error("failed to parse release date", slog.Any("error", err))
			releaseDate = time.Now()
		}

		// Create a new song
		newSong := models.Song{
			ArtistID:    artist.ID,
			Title:       songTitle,
			ReleaseDate: releaseDate,
			Text:        songDetail.Text,
			Link:        songDetail.Link,
		}

		if err := db.Create(&newSong).Error; err != nil {
			logger.Error("failed to add new song", slog.Any("error", err), slog.Any("params", map[string]string{"group": groupName, "song": songTitle}))
			c.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			return
		}
		logger.Info("added new song", slog.Any("params", map[string]string{"group": groupName, "song": songTitle}))
		song = newSong
	}

	// Format the release date
	releaseDateStr := song.ReleaseDate.Format("02.01.2006")

	songDetail := models.SongDetail{
		ReleaseDate: releaseDateStr,
		Text:        song.Text,
		Link:        song.Link,
	}

	songEnrichFromJSON(&songDetail, groupName, songTitle)
	c.JSON(http.StatusOK, songDetail)
}

func GetSongDetailAPI(group, song string, c *gin.Context) (models.SongDetail, bool) {
	encodedGroup := url.QueryEscape(group)
	encodedSong := url.QueryEscape(song)

	urlAPI := fmt.Sprintf("%s%s?group=%s&song=%s",
		config.AppConfig.ExternalAPI.BaseURL,
		config.AppConfig.ExternalAPI.InfoURL,
		encodedGroup,
		encodedSong)

	resp, err := http.Get(urlAPI)
	if err != nil {
		return handleAPIError(c, "failed to get song detail", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return handleAPIError(c, "failed to get song detail with status code", resp.StatusCode)
	}

	var dataAPI models.SongDetail
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return handleAPIError(c, "failed to read song detail", err)
	}

	if err := json.Unmarshal(body, &dataAPI); err != nil {
		return handleAPIError(c, "failed to unmarshal song detail", err)
	}

	return dataAPI, false
}

func handleAPIError(c *gin.Context, message string, err interface{}) (models.SongDetail, bool) {
	logger.Error(message, slog.Any("error", err))
	c.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
	return models.SongDetail{}, true
}

func GetSongDetailJSON(group, song string) (models.SongDetail, error) {
	jsonFileEnrich, err := os.Open("enrichInfoSong.json")
	if err != nil {
		logger.Error("could not open song enrichment file", slog.Any("error", err))
		return models.SongDetail{}, err
	}
	defer jsonFileEnrich.Close()

	jsonVal, err := io.ReadAll(jsonFileEnrich)

	if err != nil {
		logger.Error("failed to read song enrichment file", slog.Any("error", err))
		return models.SongDetail{}, err
	}

	var enrichmentData SongEnriched
	if err := json.Unmarshal(jsonVal, &enrichmentData); err != nil {
		logger.Error("failed to unmarshal song enrichment file", slog.Any("error", err))
		return models.SongDetail{}, err
	}

	if enrichmentData.Group == group && enrichmentData.Song == song {
		return models.SongDetail{
			ReleaseDate: enrichmentData.ReleaseDate,
			Text:        enrichmentData.Text,
			Link:        enrichmentData.Link,
		}, nil
	}

	return models.SongDetail{}, errors.New("invalid song enrichment")
}

func songEnrichFromJSON(songDetail *models.SongDetail, group, song string) {
	jsonFileEnrich, err := os.Open("enrichInfoSong.json")
	if err != nil {
		logger.Error("failed to open enrichInfoSong.json", slog.Any("error", err))
		return
	}
	defer jsonFileEnrich.Close()

	jsonVal, err := io.ReadAll(jsonFileEnrich)

	if err != nil {
		logger.Error("failed to read enrichInfoSong.json", slog.Any("error", err))
	}

	var enrichmentData SongEnriched
	if err := json.Unmarshal(jsonVal, &enrichmentData); err != nil {
		logger.Error("failed to unmarshal enrichInfoSong.json", slog.Any("error", err))

	} else {
		if enrichmentData.Group == group && enrichmentData.Song == song {
			songDetail.ReleaseDate = enrichmentData.ReleaseDate
			songDetail.Text = enrichmentData.Text
			songDetail.Link = enrichmentData.Link
		}
	}
}

// GetSongs godoc
// @Summary List songs with optional filtering
// @Description Retrieve a list of songs with optional filtering and pagination
// @Tags Songs
// @Accept json
// @Produce json
// @Param group query string false "Filter by Group Name"
// @Param song query string false "Filter by Song Title"
// @Param releaseDate query string false "Filter by Release Date (format: DD.MM.YYYY)"
// @Param text query string false "Filter by Text"
// @Param link query string false "Filter by Link"
// @Param page query int false "Page number for pagination" default(1)
// @Param limit query int false "Number of items per page" default(10)
// @Success 200 {array} models.Song "Songs retrieved successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid parameters"
// @Failure 404 {object} map[string]string "No songs found matching criteria"
// @Failure 500 {object} map[string]string "Internal server error - database error"
// @Router /songs [get]
func GetSongs(c *gin.Context) {
	db := database.DbConnect()
	var songs []models.Song

	group := c.Query("group")
	song := c.Query("song")
	releaseDate := c.Query("release_date")
	text := c.Query("text")
	link := c.Query("link")
	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")

	pageNumber, err := strconv.Atoi(page)
	if err != nil || pageNumber < 1 {
		pageNumber = 1
	}

	limitNumber, err := strconv.Atoi(limit)
	if err != nil || limitNumber < 1 {
		limitNumber = 10
	}

	query := db.Model(&models.Song{}).Joins("JOIN artists ON songs.artist_id = artists.id")

	if group != "" {
		query = query.Where("artists.name ILIKE ?", "%"+group+"%")
	}

	if song != "" {
		query = query.Where("songs.title ILIKE ?", "%"+song+"%")
	}

	if releaseDate != "" {
		// Parse the filter date
		date, err := time.Parse("02.01.2006", releaseDate)
		if err == nil {
			query = query.Where("songs.release_date = ?", date)
		}
	}

	if text != "" {
		query = query.Where("songs.text ILIKE ?", "%"+text+"%")
	}

	if link != "" {
		query = query.Where("songs.link ILIKE ?", "%"+link+"%")
	}

	offset := (pageNumber - 1) * limitNumber
	query = query.Offset(offset).Limit(limitNumber)

	if err := query.Find(&songs).Error; err != nil {
		logger.Error("failed to query songs", slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		return
	}

	logger.Info("Songs retrieved successfully", slog.Int("count", len(songs)))
	c.JSON(http.StatusOK, songs)
}

// GetSongText godoc
// @Summary Get song text by ID with pagination
// @Description Retrieve song text for a specific song ID with pagination support
// @Tags Songs
// @Accept json
// @Produce json
// @Param id path int true "Song ID"
// @Param page query int false "Page number for text pagination" default(1)
// @Param limit query int false "Number of text lines per page" default(10)
// @Success 200 {object} map[string]interface{} "Song text retrieved successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid ID format"
// @Failure 404 {object} map[string]string "Song or page not found"
// @Failure 500 {object} map[string]string "Internal server error - database error"
// @Router /songs/{id}/text [get]
func GetSongText(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Error("invalid song ID format", slog.Any("id", c.Param("id")))
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid song ID format"})
		return
	}

	db := database.DbConnect()
	var song models.Song

	if err := db.Unscoped().First(&song, id).Error; err != nil {
		logger.Error("failed to query song", slog.Any("id", id))
		c.JSON(http.StatusNotFound, gin.H{"message": "song not found"})
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit < 1 {
		limit = 10
	}

	text := strings.Split(song.Text, "\n\n")

	totalText := len(text)
	if totalText == 0 {
		logger.Error("no text found for song with id ", slog.Any("id", id))
		c.JSON(http.StatusNotFound, gin.H{"message": "text not found"})
		return
	}

	beginOfIndex := (page - 1) * limit
	endOfIndex := beginOfIndex + limit

	if beginOfIndex >= totalText {
		logger.Error("page out of range for song id", slog.Any("id", id), slog.Any("page", page))
		c.JSON(http.StatusNotFound, gin.H{"message": "no text found for requested page"})
		return
	}

	if endOfIndex > totalText {
		endOfIndex = totalText
	}

	selectText := text[beginOfIndex:endOfIndex]
	resp := map[string]interface{}{
		"songId":    id,
		"page":      page,
		"text":      selectText,
		"total":     totalText,
		"limit":     limit,
		"totalPage": (totalText + limit - 1) / limit,
	}

	logger.Info("retrieved text for song id ", slog.Any("id", id), slog.Any("page", page))
	c.JSON(http.StatusOK, resp)
}

// UpdateSong godoc
// @Summary Update an existing song
// @Description Update song information by ID (supports partial updates)
// @Tags Songs
// @Accept json
// @Produce json
// @Param id path int true "Song ID"
// @Param song body models.SongUpdate true "Song Update Information (supports partial updates)"
// @Success 200 {object} map[string]string "Song updated successfully"
// @Failure 400 {object} map[string]string "Invalid song data or ID format"
// @Failure 404 {object} map[string]string "Song not found"
// @Failure 500 {object} map[string]string "Internal server error - database error"
// @Router /songs/{id} [patch]
func UpdateSong(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Error("invalid song ID format", slog.Any("id", c.Param("id")))
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid song ID format"})
		return
	}

	db := database.DbConnect()
	var song models.Song
	if err := db.First(&song, id).Error; err != nil {
		logger.Error("song not found", slog.Any("id", id))
		c.JSON(http.StatusNotFound, gin.H{"message": "song not found"})
		return
	}

	var updateData models.SongUpdate
	if err := c.ShouldBindJSON(&updateData); err != nil {
		logger.Error("invalid song update data", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
		return
	}

	if updateData.Title != nil {
		song.Title = *updateData.Title
	}
	if updateData.ReleaseDate != nil {
		song.ReleaseDate = *updateData.ReleaseDate
		logger.Error("invalid release date format", slog.Any("error", err))
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid release date format"})
		return

	}
	if updateData.Text != nil {
		song.Text = *updateData.Text
	}
	if updateData.Link != nil {
		song.Link = *updateData.Link
	}

	if err := db.Save(&song).Error; err != nil {
		logger.Error("failed to update song", slog.Any("id", id), slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		return
	}

	logger.Info("song updated successfully", slog.Any("id", id))
	c.JSON(http.StatusOK, gin.H{"message": "song updated successfully"})
}

// DeleteSong godoc
// @Summary Delete a song
// @Description Delete a song by its ID
// @Tags Songs
// @Accept json
// @Produce json
// @Param id path int true "Song ID"
// @Success 200 {object} map[string]string "Song deleted successfully"
// @Failure 400 {object} map[string]string "Invalid song ID format"
// @Failure 404 {object} map[string]string "Song not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /songs/{id} [delete]
func DeleteSong(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		logger.Error("invalid song ID format", slog.Any("id", c.Param("id")))
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid song ID format"})
		return
	}

	db := database.DbConnect()
	if err := db.Delete(&models.Song{}, id).Error; err != nil {
		logger.Error("failed to delete song", slog.Any("id", id), slog.Any("error", err))
		c.JSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		return
	}

	logger.Info("song deleted successfully", slog.Any("id", id))
	c.JSON(http.StatusOK, gin.H{"message": "song deleted successfully"})
}
