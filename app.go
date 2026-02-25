package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"math/rand"
	"net/http"
	"strconv"

	_ "modernc.org/sqlite"
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
	db, err = sql.Open("sqlite", "datePlans.db")
	if err != nil {
		log.Fatal(err)
	}

	statement := `CREATE TABLE IF NOT EXISTS datePlans (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		content TEXT,
		category TEXT DEFAULT 'General',
		like INTEGER
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

// getRandomPlan randomly gets one of date plans from database.
// It needs improvement because getting from DB every time is inefficient.
// func getRandomPlan(w http.ResponseWriter, r *http.Request) {
// 	randomId := rand.Intn(sizeOfRow + 1)
// 	query := `SELECT id, title, content FROM datePlans WHERE id = ?`
// 	var p Plan
// 	err := db.QueryRow(query, randomId).Scan(&p.ID, &p.Title, &p.Content)
// 	if err != nil {
// 		log.Println("Database Error:", err.Error())
// 		renderJSONError(w, "Internal server error", http.StatusInternalServerError)
// 		return
// 	}
// 	renderJSON(w, &p)
// }

func getRandomPlan(w http.ResponseWriter, r *http.Request) {
	randomId := rand.Intn(sizeOfRow)
	query := `SELECT id, title, content, category, like FROM datePlans WHERE id >= ? ORDER BY id ASC LIMIT 1`
	var p Plan
	err := db.QueryRow(query, randomId).Scan(&p.ID, &p.Title, &p.Content, &p.Category, &p.Like)
	if err != nil {
		query = `SELECT id, title, content, category, like FROM datePlans WHERE id <= ? ORDER BY id DESC LIMIT 1`
		log.Println("Second sql issued")
		err = db.QueryRow(query, randomId).Scan(&p.ID, &p.Title, &p.Content, &p.Category, &p.Like)
		if err != nil {
			log.Printf("SQL Error: %v", err)
			renderJSONError(w, "Internal server error", http.StatusInternalServerError)
		}
	}
	renderJSON(w, &p)
}

func addPlan(w http.ResponseWriter, r *http.Request) {
	var newPlan addPlanRequest
	err := json.NewDecoder(r.Body).Decode(&newPlan)
	if err != nil {
		log.Println("Decode error: ", err.Error())
		renderJSONError(w, "Decoding failed", http.StatusBadRequest)
		return
	}
	query := `INSERT INTO datePlans (title, content) VALUES (?, ?) RETURNING id, title, content`
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

func deletePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		log.Println("method not allowed")
		renderJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
	var req deletePlanRequest
	json.NewDecoder(r.Body).Decode(&req)
	query := `DELETE FROM datePlans WHERE id = ? RETURNING id, title, content`
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
func renderJSON(w http.ResponseWriter, p *Plan) {
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
	http.HandleFunc("POST /datePlan/", addPlan)
	http.HandleFunc("DELETE /datePlan/", deletePlan)
	log.Println("Server started at :8080.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
