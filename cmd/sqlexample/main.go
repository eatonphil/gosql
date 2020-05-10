package main

import (
	"database/sql"
	"fmt"

	_ "github.com/eatonphil/gosql"
)

func main() {
	db, err := sql.Open("postgres", "")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Query("CREATE TABLE users (name TEXT, age INT);")
	if err != nil {
		panic(err)
	}

	_, err = db.Query("INSERT INTO users VALUES ('Terry', 45);")
	if err != nil {
		panic(err)
	}

	_, err = db.Query("INSERT INTO users VALUES ('Anette', 57);")
	if err != nil {
		panic(err)
	}

	rows, err := db.Query("SELECT name, age FROM users;")
	if err != nil {
		panic(err)
	}

	var name string
	var age uint64
	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&name, &age)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Name: %s, Age: %d\n", name, age)
	}

	if err = rows.Err(); err != nil {
		panic(err)
	}
}
