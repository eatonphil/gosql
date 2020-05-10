package gosql

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
)

type Rows struct {
	columns []ResultColumn
	index   uint64
	rows    [][]Cell
}

func (r *Rows) Columns() []string {
	columns := []string{}
	for _, c := range r.columns {
		columns = append(columns, c.Name)
	}

	return columns
}

func (r *Rows) Close() error {
	r.index = uint64(len(r.rows))
	return nil
}

func (r *Rows) Next(dest []driver.Value) error {
	if r.index >= uint64(len(r.rows)) {
		return io.EOF
	}

	row := r.rows[r.index]

	for idx, cell := range row {
		typ := r.columns[idx].Type
		switch typ {
		case IntType:
			i := cell.AsInt()
			if i == nil {
				dest[idx] = i
			} else {
				dest[idx] = *i
			}
		case TextType:
			s := cell.AsText()
			if s == nil {
				dest[idx] = s
			} else {
				dest[idx] = *s
			}
		case BoolType:
			b := cell.AsBool()
			if b == nil {
				dest[idx] = b
			} else {
				dest[idx] = b
			}
		}
	}

	r.index++
	return nil
}

type Conn struct {
	bkd Backend
}

func (dc *Conn) Query(query string, args []driver.Value) (driver.Rows, error) {
	if len(args) > 0 {
		// TODO: support parameterization
		panic("Parameterization not supported")
	}

	parser := Parser{}
	ast, err := parser.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("Error while parsing: %s", err)
	}

	// NOTE: ignorning all but the first statement
	stmt := ast.Statements[0]
	switch stmt.Kind {
	case CreateIndexKind:
		err = dc.bkd.CreateIndex(stmt.CreateIndexStatement)
		if err != nil {
			return nil, fmt.Errorf("Error adding index on table: %s", err)
		}
	case CreateTableKind:
		err = dc.bkd.CreateTable(stmt.CreateTableStatement)
		if err != nil {
			return nil, fmt.Errorf("Error creating table: %s", err)
		}
	case DropTableKind:
		err = dc.bkd.DropTable(stmt.DropTableStatement)
		if err != nil {
			return nil, fmt.Errorf("Error dropping table: %s", err)
		}
	case InsertKind:
		err = dc.bkd.Insert(stmt.InsertStatement)
		if err != nil {
			return nil, fmt.Errorf("Error inserting values: %s", err)
		}
	case SelectKind:
		results, err := dc.bkd.Select(stmt.SelectStatement)
		if err != nil {
			return nil, err
		}

		return &Rows{
			rows:    results.Rows,
			columns: results.Columns,
			index:   0,
		}, nil
	}

	return nil, nil
}

func (dc *Conn) Prepare(query string) (driver.Stmt, error) {
	panic("Prepare not implemented")
}

func (dc *Conn) Begin() (driver.Tx, error) {
	panic("Begin not implemented")
}

func (dc *Conn) Close() error {
	return nil
}

type Driver struct {
	bkd Backend
}

func (d *Driver) Open(name string) (driver.Conn, error) {
	return &Conn{d.bkd}, nil
}

func init() {
	sql.Register("postgres", &Driver{NewMemoryBackend()})
}
