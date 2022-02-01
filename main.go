package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"

	// Add database drivers here
	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	dbType   string
	filename string
	db       *sql.DB
}

func GetDatabase() Database {
	return Database{
		dbType:   "sqlite3",
		filename: "db.sqlite",
	}
}

func (db *Database) Connect() error {
	db.db, _ = sql.Open(db.dbType, db.filename)
	return nil
}

func (db *Database) Disconnect() error {
	db.db.Close()
	return nil
}

func (db *Database) CreateTable() error {
	db.Connect()
	defer db.Disconnect()
	createSQL := `
		CREATE TABLE IF NOT EXISTS url (
			"hash" TEXT NOT NULL PRIMARY KEY,
			"url" TEXT NOT NULL
		);
	`
	statement, err := db.db.Prepare(createSQL)
	if err != nil {
		return err
	}
	_, err = statement.Exec()
	return err
}

func get_url_hash(url string) string {
	h := sha256.New()
	h.Write([]byte("salt_that_should_be_generated_per_instance" + url))
	return hex.EncodeToString(h.Sum(nil))
}

func add_url_to_db(hash string, url string) error {
	db := GetDatabase()
	db.Connect()
	defer db.Disconnect()
	insertStatement := `
		INSERT INTO url(hash, url)
		VALUES (?, ?)
	`
	statement, err := db.db.Prepare(insertStatement)
	if err != nil {
		return err
	}
	_, err = statement.Exec(hash, url)
	if err != nil {
		return err
	}
	return nil

}

func get_url_from_db(hash string) (string, error) {
	db := GetDatabase()
	db.Connect()
	defer db.Disconnect()
	queryStatement := `
		SELECT url FROM url WHERE hash=? LIMIT 1
	`
	statement, err := db.db.Prepare(queryStatement)
	if err != nil {
		return "", err
	}
	row, err := statement.Query(hash)
	if err != nil {
		return "", err
	}
	var url string
	row.Scan(&url)
	return url, nil
}

func everything_handler(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Path
	url, err := get_url_from_db(hash)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
	} else {
		http.Redirect(w, r, url, http.StatusMovedPermanently)
	}
}

func add_url_handler(w http.ResponseWriter, r *http.Request) {
	url := "" // TODO get this from the request body
	hash := get_url_hash(url)

	str, err := get_url_from_db(hash)
	if err != nil {
		w.Write([]byte("Error getting from db: " + err.Error() + "\n"))
	} else if str != "" {
		// Url already in db
		w.Write([]byte("Error: url already in db\n"))
		return
	} else {
		err = add_url_to_db(hash, url)
		if err != nil {
			w.Write([]byte("Error adding to db: " + err.Error() + "\n"))
		} else {
			w.Write([]byte(`
				<html>
					<head>
						<title>Shortened URL</title>
					</head>
					<body>
						<p>Your shortened url is: <a href="` + str + `">` + str + `</a></p>
					</body>
				</html>
			`))
		}
	}
}

func main() {
	db := GetDatabase()
	err := db.Connect()
	if err != nil {
		fmt.Println("Connection failed")
		os.Exit(1)
	}
	err = db.CreateTable()
	if err != nil {
		fmt.Println("Table creation failed: " + err.Error())
		os.Exit(1)
	}
	db.Disconnect()
	fmt.Println("Database setup, starting server")
	http.HandleFunc("/", everything_handler)
	http.HandleFunc("/add_url", add_url_handler)
	http.ListenAndServe(":8080", nil)
}
