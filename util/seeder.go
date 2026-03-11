package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var db *sql.DB

func openDB() {
	var err error
	dsn := os.Getenv("DB_DSN")
	db, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	openDB()
	defer db.Close()
	addRandomPlans()
	//autoCategorize()
	//setALLLikeToZero()
	//setLikeToZero(1)
}

func addRandomPlans() {

	places := []string{"新宿", "渋谷", "横浜", "自宅", "公園", "水族館", "夜の海", "知らない駅"}
	actions := []string{"散歩する", "アイスを食べる", "夜景を見る", "ゲームをする", "深海魚を眺める", "お弁当を食べる"}
	categories := []string{"Romantic", "Adventurous", "Budget-Friendly", "Foodie", "Indoor"}
	fmt.Println("Started inserting data...")
	start := time.Now()

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err := tx.Prepare(`INSERT INTO datePlans(title, content, category, "like") VALUES($1, $2, $3, 0)`)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	for i := 1; i <= 1000000; i++ {
		title := fmt.Sprintf("%sで%s No.%d", places[rand.Intn(len(places))], actions[rand.Intn(len(actions))], i)
		body := "これはテスト用のデートプラン詳細テキストです。大量のデータの中でも爆速で動くか検証中。"

		_, err = stmt.Exec(title, body, categories[rand.Intn(len(categories))])
		if err != nil {
			tx.Rollback()
			log.Fatal(err)
		}
		if i%10000 == 0 {
			fmt.Printf("%d cases finished... \n", i)
		}
	}

	tx.Commit()

	fmt.Printf("Transaction completed; total time: %d \n", time.Since(start))
}

func autoCategorize() {
	categories := []string{"Romantic", "Adventurous", "Budget-Friendly", "Foodie", "Indoor"}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback()
	// 1. Get all IDs from the database
	var rows *sql.Rows
	rows, err = tx.Query(`SELECT id FROM datePlans`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// 2. Prepare the update statement
	var stmt *sql.Stmt
	stmt, err = tx.Prepare("UPDATE datePlans SET category = ? WHERE id = ?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	for rows.Next() {
		var id int
		rows.Scan(&id)

		// Pick a random category from our list
		randomCat := categories[rand.Intn(len(categories))]

		// Apply it to this specific ID
		_, err = stmt.Exec(randomCat, id)
		if err != nil {
			log.Fatal(err)
		}
	}
	tx.Commit()
}

func setALLLikeToZero() {
	query := `UPDATE datePlans SET "like" = 0 WHERE "like" IS NULL;`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

func setLikeToZero(id int) {
	query := `UPDATE datePlans SET like = 0 WHERE id = ?`
	_, err := db.Exec(query, id)
	if err != nil {
		log.Fatal(err)
	}
}
