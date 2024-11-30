package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "1234"
	dbname   = "sandbox"
)

var (
	db *sql.DB
	mu sync.Mutex
)

func initDB() {
	var err error
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		log.Fatalf("Database is not reachable: %v", err)
	}

	// Создаём таблицу, если её нет
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS public.counter (
			id SERIAL PRIMARY KEY,
			value INTEGER NOT NULL DEFAULT 0
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// Инициализация счётчика, если таблица пуста
	_, err = db.Exec(`INSERT INTO public.counter (id, value) VALUES (1, 0) ON CONFLICT DO NOTHING`)
	if err != nil {
		log.Fatalf("Failed to initialize counter: %v", err)
	}

	log.Println("Database initialized successfully")
}

func getCountHandler(c echo.Context) error {
	mu.Lock()
	defer mu.Unlock()

	var counter int
	err := db.QueryRow(`SELECT value FROM public.counter WHERE id = 1`).Scan(&counter)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch counter value")
	}

	return c.JSON(http.StatusOK, map[string]int{"current_count": counter})
}

func postCountHandler(c echo.Context) error {
	var request struct {
		Count int `json:"count"`
	}
	if err := c.Bind(&request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON format")
	}

	mu.Lock()
	defer mu.Unlock()

	// Увеличиваем значение счётчика в базе данных
	_, err := db.Exec(`UPDATE public.counter SET value = value + $1 WHERE id = 1`, request.Count)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update counter")
	}

	var newCounter int
	err = db.QueryRow(`SELECT value FROM public.counter WHERE id = 1`).Scan(&newCounter)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch updated counter value")
	}

	return c.JSON(http.StatusOK, map[string]int{"incremented_by": request.Count, "new_count": newCounter})
}

func main() {
	// Инициализация базы данных
	initDB()
	defer db.Close()

	// Создание Echo-инстанса
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())  // Логирование запросов
	e.Use(middleware.Recover()) // Обработка паник

	// Маршруты
	e.GET("/count", getCountHandler)
	e.POST("/count", postCountHandler)

	// Запуск сервера
	log.Println("Server is running on http://localhost:8082")
	if err := e.Start(":8082"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// curl -X GET http://localhost:8082/count
// curl -X POST -H "Content-Type: application/json" -d '{"count": 5}' http://localhost:8082/count
