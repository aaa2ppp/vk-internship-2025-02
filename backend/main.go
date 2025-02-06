package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

type PingResult struct {
	IP       string    `json:"ip"`
	PingTime time.Time `json:"ping_time"`
	Success  bool      `json:"success"`
}

var db *sql.DB

func initDB() {
	var err error
	connStr := "user=postgres dbname=monitoring sslmode=disable password=postgres host=db"
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
}

func getPingResults(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT ip, ping_time, success FROM ping_results")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []PingResult
	for rows.Next() {
		var result PingResult
		if err := rows.Scan(&result.IP, &result.PingTime, &result.Success); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func addPingResult(w http.ResponseWriter, r *http.Request) {
	var result PingResult
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := db.Exec("INSERT INTO ping_results (ip, ping_time, success) VALUES ($1, $2, $3)",
		result.IP, result.PingTime, result.Success)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func main() {
	initDB()
	http.HandleFunc("/ping-results", getPingResults)
	http.HandleFunc("/add-ping-result", addPingResult)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
