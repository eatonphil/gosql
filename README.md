# gosql

An early PostgreSQL implementation in Go.

[![gosql](https://circleci.com/gh/eatonphil/gosql.svg?style=svg)](https://circleci.com/gh/eatonphil/gosql)

## Example

```bash
$ git clone git@github.com:eatonphil/gosql
$ cd gosql
$ go run cmd/main.go
Welcome to gosql.
# CREATE TABLE users (id INT PRIMARY KEY, name TEXT, age INT);
ok
# \d users
Table "users"
  Column |  Type   | Nullable
---------+---------+-----------
  id     | integer | not null
  name   | text    |
  age    | integer |
Indexes:
        "users_pkey" PRIMARY KEY, rbtree ("id")

# INSERT INTO users VALUES (1, 'Corey', 34);
ok
# INSERT INTO users VALUES (1, 'Max', 29);
Error inserting values: Duplicate key value violates unique constraint
# INSERT INTO users VALUES (2, 'Max', 29);
ok
# SELECT * FROM users WHERE id = 2;
  id | name | age
-----+------+------
   2 | Max  |  29
(1 result)
ok
# SELECT id, name, age + 3 FROM users WHERE id = 2 OR id = 1;
  id | name  | ?column?
-----+-------+-----------
   1 | Corey |       37
   2 | Max   |       32
(2 results)
ok
```

## Architecture

* [cmd/main.go](./cmd/main.go)
  * Contains the REPL and high-level interface to the project
  * Dataflow is: user input -> lexer -> parser -> in-memory backend
* [lexer.go](./lexer.go)
  * Handles breaking user input into tokens for the parser
* [parser.go](./parser.go)
  * Matches a list of tokens into an AST or fails if the user input is not a valid program
* [memory.go](./memory.go)
  * An example, in-memory backend supporting the Backend interface (defined in backend.go)

## Contributing

* Add a new operator (such as `-`, `*`, etc.)
* Add a new data type (such as `VARCHAR(n)``)

In each case, you'll probably have to add support in the lexer,
parser, and in-memory backend. I recommend going in that order.

In all cases, make sure the code is formatted (`make fmt`), linted
(`make lint`) and passes tests (`make test`). New code should have
tests.

## Blog series

* [Writing a SQL database from scratch in Go](https://notes.eatonphil.com/database-basics.html)
* [Binary expressions and WHERE filters](https://notes.eatonphil.com/database-basics-expressions-and-where.html)
* [Indexes](https://notes.eatonphil.com/database-basics-indexes.html)

## Further reading

Here are some similar projects written in Go.

* [go-mysql-server](https://github.com/src-d/go-mysql-server)
  * This is a MySQL frontend (with an in-memory backend for testing only).
* [ramsql](https://github.com/proullon/ramsql)
  * This is a WIP PostgreSQL-compatible in-memory database.
* [CockroachDB](https://github.com/cockroachdb/cockroach)
  * This is a production-ready PostgreSQL-compatible database.
