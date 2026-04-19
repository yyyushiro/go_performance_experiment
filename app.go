package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Plan struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Category string `json:"category"`
	Like     string `json:"like"`
}

type addPlanRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type deletePlanRequest struct {
	Id int `json:"id"`
}

var db *sql.DB
var sizeOfRow int

// initDB initializes the database.
func initDB() {
	var err error
	// Specify the destination of database. It uses network, so the form is like URL, not a path.
	dsn := os.Getenv("DB_DSN")
	db, err = sql.Open("pgx", dsn)
	db.SetMaxOpenConns(25)                 // 同時に開く接続の最大数
	db.SetMaxIdleConns(25)                 // アイドル状態で保持する接続数
	db.SetConnMaxLifetime(5 * time.Minute) // 接続の最大生存時間

	statement := `CREATE TABLE IF NOT EXISTS datePlans (
		id SERIAL PRIMARY KEY,
		title TEXT,
		content TEXT,
		category TEXT DEFAULT 'General',
		"like" INTEGER
	);`

	_, err = db.Exec(statement)
	if err != nil {
		log.Fatal(err)
	}
}

// getSizeOfRow gets the size of row in current database.
func getSizeOfRow(db *sql.DB) error {
	query := `SELECT MAX(id) FROM datePlans`
	err := db.QueryRow(query).Scan(&sizeOfRow)
	if err != nil {
		log.Println("Database Error: ", err)
	}
	return err
}

// getRandomPlan gets a data plan randomly.
func getRandomPlan(w http.ResponseWriter, r *http.Request) {
	randomId := rand.Intn(sizeOfRow)
	query := `SELECT id, title, content, category, like FROM datePlans WHERE id >= $1 ORDER BY id ASC LIMIT 1`
	var p Plan
	err := db.QueryRow(query, randomId).Scan(&p.ID, &p.Title, &p.Content, &p.Category, &p.Like)
	if err != nil {
		query = `SELECT id, title, content, category, like FROM datePlans WHERE id <= $1 ORDER BY id DESC LIMIT 1`
		log.Println("Second sql issued")
		err = db.QueryRow(query, randomId).Scan(&p.ID, &p.Title, &p.Content, &p.Category, &p.Like)
		if err != nil {
			log.Printf("SQL Error: %v", err)
			renderJSONError(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}
	renderJSON(w, &p)
}

// getPlan gets a data plan with the specified id.
func getPlan(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.URL.Path[len("/datePlan/"):])
	if err != nil {
		log.Println(err)
		renderJSONError(w, "Invalid id", http.StatusBadRequest)
		return
	}

	query := `SELECT id, title, content, category, "like" FROM datePlans WHERE id = $1`
	var p Plan
	err = db.QueryRow(query, id).Scan(&p.ID, &p.Title, &p.Content, &p.Category, &p.Like)
	if err != nil {
		log.Println(err)
		renderJSONError(w, "Internal server error", http.StatusBadRequest)
		return
	}

	renderJSON(w, &p)
}

// likePlan likes a plan.
func likePlan(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		log.Println(err)
		renderJSONError(w, "invalid id", http.StatusBadRequest)
		return
	}

	query := `UPDATE datePlans SET "like" = "like" + 1 WHERE id = $1 RETURNING "like"`
	var like int
	err = db.QueryRow(query, id).Scan(&like)

	if err != nil {
		log.Println(err)
		renderJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	// Make anonymous maps.
	response := map[string]any{
		"id":   id,
		"like": like,
	}
	renderJSON(w, response)
}

// addPlan adds a data plan.
func addPlan(w http.ResponseWriter, r *http.Request) {
	var newPlan addPlanRequest
	err := json.NewDecoder(r.Body).Decode(&newPlan)
	if err != nil {
		log.Println("Decode error: ", err.Error())
		renderJSONError(w, "Decoding failed", http.StatusBadRequest)
		return
	}
	// Check if the fields are empty.
	if newPlan.Title == "" {
		log.Println("Incomplete request: missing title")
		renderJSONError(w, "missing title", http.StatusBadRequest)
		return
	}
	if newPlan.Content == "" {
		log.Println("Incomplete request: missing content")
		renderJSONError(w, "missing content", http.StatusBadRequest)
		return
	}
	query := `INSERT INTO datePlans (title, content) VALUES ($1, $2) RETURNING id, title, content`
	var p Plan
	err = db.QueryRow(query, newPlan.Title, newPlan.Content).Scan(&p.ID, &p.Title, &p.Content, &p.Category, &p.Like)
	if err != nil {
		log.Println("Database error: ", err)
		renderJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	// Increment the number of rows in the database after making sure query succeeded.
	sizeOfRow++
	fmt.Printf("Id: %v, Title: %s, Content: %s, Category: %s, Like: %s", p.ID, p.Title, p.Content, p.Category, p.Like)
	renderJSON(w, &p)
}

// deletePlan deletes a plan.
func deletePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		log.Println("method not allowed")
		renderJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req deletePlanRequest
	json.NewDecoder(r.Body).Decode(&req)
	query := `DELETE FROM datePlans WHERE id = $1 RETURNING id, title, content, category, like`
	var p Plan
	err := db.QueryRow(query, req.Id).Scan(&p.ID, &p.Title, &p.Content, &p.Category, &p.Like)
	if err != nil {
		log.Println("Database error: ", err)
		renderJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	fmt.Printf("Id: %v, Title: %s, Content: %s, Category: %s, Like: %s", p.ID, p.Title, p.Content, p.Category, p.Like)
	renderJSON(w, &p)
}

// renderJSON renders the date plan into JSON.
func renderJSON(w http.ResponseWriter, p any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(p)
}

// renderJSONError renders the error message into JSON.
func renderJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func main() {
	initDB()
	defer db.Close()
	// gets the size of row of database only once.
	getSizeOfRow(db)

	http.HandleFunc("GET /datePlan/", getRandomPlan)
	http.HandleFunc("GET /datePlan/{id}", getPlan)
	http.HandleFunc("POST /datePlan/{id}/like", likePlan)
	http.HandleFunc("POST /datePlan/", addPlan)
	http.HandleFunc("DELETE /datePlan/", deletePlan)
	log.Println("Server started at :8080.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
