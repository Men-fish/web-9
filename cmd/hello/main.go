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

type Handlers struct {
	dbProvider DatabaseProvider
}

type DatabaseProvider struct {
	db *sql.DB
}

// Обработчики HTTP-запросов
func (h *Handlers) GetHello(c echo.Context) error {
	msg, err := h.dbProvider.SelectHello()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.String(http.StatusOK, msg)
}

func (h *Handlers) PostHello(c echo.Context) error {
	input := struct {
		Msg string `json:"msg"`
	}{}

	if err := c.Bind(&input); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid JSON format")
	}

	if err := h.dbProvider.InsertHello(input.Msg); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusCreated, map[string]string{"status": "message inserted"})
}

// Методы для работы с базой данных
func (dp *DatabaseProvider) SelectHello() (string, error) {
	var msg string

	// Получаем одно сообщение из таблицы hello, отсортированной в случайном порядке
	row := dp.db.QueryRow("SELECT message FROM hello ORDER BY RANDOM() LIMIT 1")
	err := row.Scan(&msg)
	if err != nil {
		return "", err
	}

	return msg, nil
}

func (dp *DatabaseProvider) InsertHello(msg string) error {
	_, err := dp.db.Exec("INSERT INTO public.hello (message) VALUES (ARRAY[$1])", msg)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	// Формирование строки подключения для postgres
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Создание соединения с сервером postgres
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}
	defer db.Close()

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		log.Fatalf("Database is not reachable: %v", err)
	}

	// Создаем провайдер для БД
	dp := DatabaseProvider{db: db}
	// Создаем экземпляр обработчиков
	h := Handlers{dbProvider: dp}

	// Создание Echo-инстанса
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())  // Логирование запросов
	e.Use(middleware.Recover()) // Обработка паник

	// Маршруты
	e.GET("/get", h.GetHello)
	e.POST("/post", h.PostHello)

	// Запуск сервера
	log.Println("Server is running on http://127.0.0.1:8081")
	if err := e.Start(":8081"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// curl -X POST -H "Content-Type: application/json" -d '{"msg": "Привет, мир!"}' http://127.0.0.1:8081/post
// curl -X GET http://127.0.0.1:8081/get
