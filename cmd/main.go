package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	_ "github.com/mattn/go-sqlite3"
)

type ErrorResponse struct {
	Error            string `json:"error"`
	DeveloperDetails string `json:"developer_details,omitempty"`
}

type SuccessResponse struct {
	Data interface{}   `json:"data"`
	Meta *MetaResponse `json:"meta,omitempty"`
}

type MetaResponse struct {
	Pagination PaginationResponse `json:"pagination"`
}

type PaginationResponse struct {
	Page      int `json:"page"`
	PageSize  int `json:"page_size"`
	PageCount int `json:"page_count"`
	Total     int `json:"total"`
}

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	// Connect to SQLite database
	db, err := sql.Open("sqlite3", "database.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create users table if it doesn't exist
	createTable(db)

	// Initialize Echo
	e := echo.New()

	// Routes
	e.GET("/users", getUsers(db))
	e.GET("/users/:id", getUser(db))
	e.POST("/users", createUser(db))
	e.PUT("/users/:id", updateUser(db))
	e.DELETE("/users/:id", deleteUser(db))

	// Start the server
	e.Logger.Fatal(e.Start(":8080"))
}

func createTable(db *sql.DB) {
	query := `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			age INTEGER NOT NULL
		)
	`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

func getUsers(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		page, _ := strconv.Atoi(c.QueryParam("page"))
		limit, _ := strconv.Atoi(c.QueryParam("limit"))

		if page < 1 {
			page = 1
		}

		if limit < 1 {
			limit = 10
		}

		offset := (page - 1) * limit

		var total int
		db.QueryRow("SELECT COUNT(*) FROM users").Scan(&total)

		rows, err := db.Query("SELECT * FROM users LIMIT ? OFFSET ?", limit, offset)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		users := []User{}
		for rows.Next() {
			var user User
			err := rows.Scan(&user.ID, &user.Name, &user.Age)
			if err != nil {
				log.Print(err)

				errResponse := ErrorResponse{
					Error:            "No users found",
					DeveloperDetails: err.Error(),
				}
				return c.JSON(http.StatusNotFound, errResponse)
			}
			users = append(users, user)
		}

		pageCount := total / limit

		if total%limit > 0 {
			pageCount++
		}

		pagination := PaginationResponse{
			Page:      page,
			PageSize:  limit,
			PageCount: pageCount,
			Total:     total,
		}

		meta := MetaResponse{
			Pagination: pagination,
		}

		successResponse := SuccessResponse{
			Data: users,
			Meta: &meta,
		}

		return c.JSON(http.StatusOK, successResponse)
	}
}

func getUser(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")

		var user User
		err := db.QueryRow("SELECT * FROM users WHERE id = ?", id).Scan(&user.ID, &user.Name, &user.Age)
		if err != nil {
			log.Print(err)

			errResponse := ErrorResponse{
				Error:            "User not found",
				DeveloperDetails: err.Error(),
			}
			return c.JSON(http.StatusNotFound, errResponse)
		}

		return c.JSON(http.StatusOK, SuccessResponse{Data: user})
	}
}

func createUser(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		user := new(User)
		if err := c.Bind(user); err != nil {
			return err
		}

		result, err := db.Exec("INSERT INTO users (name, age) VALUES (?, ?)", user.Name, user.Age)
		if err != nil {
			log.Fatal(err)
		}

		lastInsertId, _ := result.LastInsertId()
		user.ID = int(lastInsertId)

		return c.JSON(http.StatusCreated, user)
	}
}

func updateUser(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")

		user := new(User)
		if err := c.Bind(user); err != nil {
			return err
		}

		_, err := db.Exec("UPDATE users SET name = ?, age = ? WHERE id = ?", user.Name, user.Age, id)
		if err != nil {
			log.Fatal(err)
		}

		return c.NoContent(http.StatusOK)
	}
}

func deleteUser(db *sql.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		id := c.Param("id")

		_, err := db.Exec("DELETE FROM users WHERE id = ?", id)
		if err != nil {
			log.Fatal(err)
		}

		return c.NoContent(http.StatusOK)
	}
}
