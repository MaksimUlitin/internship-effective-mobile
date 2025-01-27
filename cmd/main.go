package main

import (
	"internship-effective-mobile/config"
	"internship-effective-mobile/internal/routes"
	"internship-effective-mobile/internal/storage/database"
	"log"
)

func main() {
	config.LoadConfigEnv()
	logger.Info("environment variables loaded")

	db := database.DbConnect()
	logger.Info("database connect success")

	database.Migrate(db)
	logger.Info("database migrate success")

	log.Fatal(routes.Router().Run(":8080"))
}
