package routes

import (
	"github.com/gin-gonic/gin"
	"internship-effective-mobile/internal/controllers"
)

func Router() *gin.Engine {
	r := gin.Default()
	r.POST("/info", controllers.AddSongInfo)
	r.GET("/songs", controllers.GetSongs)
	r.GET("/songs/:id/text", controllers.GetSongText)
	r.PUT("/songs/:id", controllers.UpdateSong)
	r.DELETE("/songs/:id", controllers.DeleteSong)

	return r
}
