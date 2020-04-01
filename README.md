# gosql

An early PostgreSQL implementation in Go.

## Example

```bash
$ git clone git@github.com:eatonphil/gosql
$ cd gosql
$ go run cmd/main.go
Welcome to gosql.
# CREATE TABLE users (name TEXT, age INT);
ok
#  INSERT INTO users VALUES ('Stephen', 16);
ok
# SELECT name, age FROM users;
   name   | age
----------+------
  Stephen |  16
(1 result)
ok
# INSERT INTO users VALUES ('Adrienne', 23);
ok
# SELECT age + 2, name FROM users WHERE age = 23;
  age |   name
------+-----------
   25 | Adrienne
(1 result)
ok
# SELECT name FROM users;
    name
------------
  Stephen
  Adrienne
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

* Add a new operator (such as `<`, `>`, etc.) supported by PostgreSQL
* Add a new data type supported by PostgreSQL

In each case, you'll probably have to add support in the lexer,
parser, and in-memory backend. I recommend going in that order.

In all cases, make sure the code is formatted (`make fmt`) and passes
tests (`make test`). New code should have tests.

## Blog series

* [https://notes.eatonphil.com/database-basics.html](Writing a SQL database from scratch in Go)

## Further reading

Here are some similar projects written in Go.

* [go-mysql-server](https://github.com/src-d/go-mysql-server)
  * This is a MySQL frontend (with an in-memory backend for testing only).
* [ramsql](https://github.com/proullon/ramsql)
  * This is a WIP PostgreSQL-compatible in-memory database.
* [CockroachDB](https://github.com/cockroachdb/cockroach)
  * This is a production-ready PostgreSQL-compatible database.