package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

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

var db *sql.DB

func initDB() {
	var err error
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
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
		CREATE TABLE IF NOT EXISTS public.users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) UNIQUE NOT NULL
		)
	`)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	log.Println("Database initialized successfully")
}

// Структура для парсинга имени пользователя из URL запроса
type UserRequest struct {
	Name string `json:"name"`
}

// Обработчик запроса
func userHandler(c echo.Context) error {
	// Получаем параметр 'name' из query-параметра
	name := c.QueryParam("name")
	if name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Parameter 'name' is required")
	}

	// Проверяем, существует ли пользователь в базе данных
	var exists bool
	err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM public.users WHERE name = $1)`, name).Scan(&exists)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Database error")
	}

	if !exists {
		return echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("User '%s' not found", name))
	}

	// Возвращаем успешный ответ
	response := fmt.Sprintf("Hello, %s!", name)
	return c.String(http.StatusOK, response)
}

// Обработчик для создания пользователя (для теста)
func createUserHandler(c echo.Context) error {
	// Создаём структуру для парсинга JSON
	req := new(UserRequest)

	// Пробуем распарсить JSON
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON format")
	}

	// Вставляем пользователя в базу данных
	_, err := db.Exec(`INSERT INTO public.users (name) VALUES ($1)`, req.Name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create user")
	}

	// Возвращаем успешный ответ
	return c.JSON(http.StatusCreated, map[string]string{"status": "user created", "name": req.Name})
}

func main() {
	// Инициализация базы данных
	initDB()
	defer db.Close()

	// Создание нового Echo инстанса
	e := echo.New()

	// Middleware для логирования и обработки ошибок
	e.Use(middleware.Logger())  // Логирование запросов
	e.Use(middleware.Recover()) // Восстановление после паники

	// Роутинг
	e.GET("/api/user", userHandler)
	e.POST("/api/user", createUserHandler)

	// Запуск сервера
	log.Println("Server is running on http://localhost:8083")
	if err := e.Start(":8083"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// curl -X POST http://localhost:8083/api/user -H "Content-Type: application/json" -d '{"name": "Men-fish"}'
// curl "http://localhost:8083/api/user?name=Men-fish"
