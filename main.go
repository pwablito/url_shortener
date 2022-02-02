package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
	return hex.EncodeToString(h.Sum(nil))[:8]
}

func add_url_to_db(hash string, url string) error {
	db := GetDatabase()
	db.Connect()
	defer db.Disconnect()
	insertStatement := `
		INSERT OR IGNORE INTO url(hash, url)
		VALUES (?, ?)
	`
	statement, err := db.db.Prepare(insertStatement)
	if err != nil {
		return err
	}
	_, err = statement.Exec(hash, url)
	return err

}

func get_url_from_db(hash string) (string, error) {
	db := GetDatabase()
	db.Connect()
	defer db.Disconnect()
	queryStatement := `
		SELECT url FROM url WHERE hash=? LIMIT 1
	`
	row := db.db.QueryRow(queryStatement, hash)
	var url string
	err := row.Scan(&url)
	if err != nil {
		return "", err
	}
	return url, nil
}

func shortcut_handler(w http.ResponseWriter, r *http.Request) {
	hash := strings.TrimPrefix(r.URL.Path, "/url/")
	url, err := get_url_from_db(hash)
	if err != nil {
		fmt.Println(hash + " not found")
		http.Error(w, "Not found", http.StatusNotFound)
	} else {
		fmt.Println("Redirecting " + hash + " to " + url)
		http.Redirect(w, r, url, http.StatusMovedPermanently)
	}
}

func render_page_handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Serving " + r.URL.Path)
	contents, err := ioutil.ReadFile("public" + r.URL.Path)
	if err != nil {
		w.Write([]byte("Something went wrong"))
	}
	w.Write(contents)
}

func index_redirect_handler(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/index.html", http.StatusMovedPermanently)
}

func write_shortened_url_response(w http.ResponseWriter, url string) {
	w.Write([]byte(`
		<html>
			<head>
				<title>Shortened URL</title>
			</head>
			<body>
				<p>Your shortened url is: <a href="` + url + `">` + url + `</a></p>
			</body>
		</html>
	`))
}

func add_url_handler(w http.ResponseWriter, r *http.Request) {
	url := r.FormValue("url")
	hash := get_url_hash(url)

	err := add_url_to_db(hash, url)
	if err != nil {
		w.Write([]byte("Error adding to db: " + err.Error() + "\n"))
	} else {
		write_shortened_url_response(w, "http://"+r.Host+"/url/"+hash)
	}
}

func not_found_handler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Not found: " + r.URL.Path)
	w.Write([]byte("Page Not found"))
}

func other_handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		index_redirect_handler(w, r)
	} else {
		not_found_handler(w, r)
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
	fmt.Println("Database up")
	err = filepath.Walk("./public", func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			http.HandleFunc(strings.TrimPrefix(path, "public"), render_page_handler)
		}
		return nil
	})
	if err != nil {
		fmt.Println("Failed to register static handlers")
		os.Exit(1)
	}
	http.HandleFunc("/url/", shortcut_handler)
	http.HandleFunc("/add_url", add_url_handler)
	http.HandleFunc("/", other_handler)
	fmt.Println("Server up")
	err = http.ListenAndServe(":80", nil)
	if err != nil {
		fmt.Println("Failed to start server")
		os.Exit(1)
	}
}
