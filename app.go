package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"

	_ "modernc.org/sqlite"
)

type Plan struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
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
		content TEXT
	);`

	_, err = db.Exec(statement)
	if err != nil {
		log.Fatal(err)
	}
}

// getSizeOfRow gets the size of row in current database.
func getSizeOfRow(db *sql.DB) error {
	query := `SELECT COUNT(id) FROM datePlans`
	err := db.QueryRow(query).Scan(&sizeOfRow)
	if err != nil {
		log.Println("Database Error", err.Error())
	}
	return err
}

// getRandomPlan randomly gets one of date plans from database.
// It needs improvement because getting from DB every time is inefficient.
func getRandomPlan(w http.ResponseWriter, r *http.Request) {
	randomId := rand.Intn(sizeOfRow + 1)
	query := `SELECT id, title, content FROM datePlans WHERE id = ?`
	var p Plan
	err := db.QueryRow(query, randomId).Scan(&p.ID, &p.Title, &p.Content)
	if err != nil {
		log.Println("Database Error:", err.Error())
		renderJSONError(w, "Internal server error", http.StatusInternalServerError)
		return
	}
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

	http.HandleFunc("/datePlan/", getRandomPlan)
	log.Println("Server started at :8080.")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
