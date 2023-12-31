package main

import (
	"employees/controller"
	"employees/repository"
	"employees/routes"
	"employees/service"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/avast/retry-go"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"
	"time"
)

const maxRetries = 5
const retryDelay = 1 * time.Second

func main() {
	app := fiber.New()
	app.Use(cors.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: os.Getenv("ALLOWED_ORIGINS"),
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	var db *gorm.DB
	err := retry.Do(
		func() error {
			db = initializeDatabaseConnection()
			err := db.Raw("SELECT 1").Scan(&struct{}{}).Error
			if err != nil {
				log.Println("Error connecting to the database:", err)
				return err
			}
			return nil
		},
		retry.Attempts(maxRetries),
		retry.Delay(retryDelay),
	)

	if err != nil {
		log.Fatal("Failed to connect to the database after multiple attempts.")
	}

	repository.RunMigrations(db)
	employeeRepository := repository.NewEmployeeRepository(db)
	employeeService := service.NewEmployeeService(employeeRepository)
	employeeController := controller.NewEmployeeController(employeeService)
	routes.RegisterRoute(app, employeeController)

	err = app.Listen(":8080")
	if err != nil {
		log.Fatalln(fmt.Sprintf("error starting the server %s", err.Error()))
	}
}

func initializeDatabaseConnection() *gorm.DB {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  createDsn(),
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		log.Println(fmt.Sprintf("error connecting with database %s", err.Error()))
	}
	return db
}

func createDsn() string {
	dsnFormat := "host=%s user=%s password=%s dbname=%s port=%s sslmode=disable"
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	return fmt.Sprintf(dsnFormat, dbHost, dbUser, dbPassword, dbName, dbPort)
}
