package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

// DB credentials
const (
	host     = "localhost"
	port     = 5432
	user     = "tommychu"
	password = "ferdajekamarad"
	dbname   = "word_list"
)

// global scope DB
var db *sql.DB

func main() {

	// define database connection
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	// connect to db
	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(errors.Wrap(err, "connecting to the database"))
	}
	defer db.Close()
	// check the db connecton
	err = db.Ping()
	if err != nil {
		panic(errors.Wrap(err, "ping to DB conn"))
	}

	// insert the data into DB tables
	err = parseInDB("words/index.adj", "adjective")
	panicErr(err)

	err = parseInDB("words/index.adv", "adverb")
	panicErr(err)

	err = parseInDB("words/index.noun", "noun")
	panicErr(err)

	err = parseInDB("words/index.verb", "verb")
	panicErr(err)
}

// parseInDB parses the file
func parseInDB(filename string, category string) error {

	// log
	fmt.Printf("Parsing file %s (%s)...\n", filename, category)

	// open file
	f, err := os.Open(filename)
	if err != nil {
		return errors.Wrap(err, "filename: "+filename)
	}
	defer f.Close()

	// enable scanning
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	// range over lines
	for scanner.Scan() {

		// get the line
		ln := strings.Split(scanner.Text(), " ")
		// insert into DB
		if err = insert(ln[0], category); err != nil {
			return err
		}
	}

	return nil
}

// insert inserts the word into the database and assings the given category to it.
func insert(word string, category string) error {

	// define SQL query
	query := `
	INSERT INTO words (word, category_id)
	VALUES ($1, (SELECT id
	FROM categories
	WHERE type = $2 LIMIT 1))`
	_, err := db.Exec(query, word, category)
	if err != nil {
		return errors.Wrap(err, "error occured when inserting: "+word+" ("+category+")")
	}
	return nil
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}
