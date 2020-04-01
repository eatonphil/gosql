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
