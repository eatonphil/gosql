# gosql

An early PostgreSQL implementation in Go (WIP)

## Example

```bash
$ git clone git@github.com:eatonphil/gosql
$ cd gosql
$ go run cmd/main.go
Welcome to gosql.
# CREATE TABLE users (id INT, name TEXT);
ok
# INSERT INTO users VALUES (1, 'Phil');
ok
# SELECT id, name FROM users;
| id | name |
====================
| 1 |  Phil |
ok
# INSERT INTO users VALUES (2, 'Kate');
ok
# SELECT name, id FROM users;
| name | id |
====================
| Phil |  1 |
| Kate |  2 |
ok
```
