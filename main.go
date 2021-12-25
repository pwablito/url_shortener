package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"

	// Add database drivers here
	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	dbType   string
	filename string
	db       *sql.DB
}

func (db *Database) Connect() error {
	db.db, _ = sql.Open(db.dbType, db.filename)
	return nil
}

func (db *Database) Disconnect() error {
	db.db.Close()
	return nil
}

func get_url_hash(url string) string {
	h := sha256.New()
	h.Write([]byte(url))
	return hex.EncodeToString(h.Sum(nil))
}

func add_url_to_db(hash string, url string) error {
	return errors.New("not implemented")
}

func get_from_db(hash string) (string, error) {
	return "", nil // TODO implement
}

func everything_handler(w http.ResponseWriter, r *http.Request) {
	hash := r.URL.Path
	url, err := get_from_db(hash)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
	} else {
		http.Redirect(w, r, url, http.StatusMovedPermanently)
	}
}

func add_url_handler(w http.ResponseWriter, r *http.Request) {
	url := "" // TODO get this from the request body
	hash := get_url_hash(url)

	str, err := get_from_db(hash)
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
	http.HandleFunc("/", everything_handler)
	http.HandleFunc("/add_url", add_url_handler)
	http.ListenAndServe(":8080", nil)
}
