package stdlib_test

import (
	"database/sql"
	_ "github.com/JackC/pgx/stdlib"
	"testing"
)

func openDB(t *testing.T) *sql.DB {
	db, err := sql.Open("pgx", "postgres://pgx_md5:secret@localhost:5432/pgx_test")
	if err != nil {
		t.Fatalf("sql.Open failed: %v", err)
	}

	return db
}

func closeDB(t *testing.T, db *sql.DB) {
	err := db.Close()
	if err != nil {
		t.Fatalf("db.Close unexpectedly failed: %v", err)
	}
}

func TestNormalLifeCycle(t *testing.T) {
	db := openDB(t)
	defer closeDB(t, db)

	stmt, err := db.Prepare("select 'foo', n from generate_series($1::int, $2::int) n")
	if err != nil {
		t.Fatalf("db.Prepare unexpectedly failed: %v", err)
	}
	defer func() {
		err = stmt.Close()
		if err != nil {
			t.Fatalf("stmt.Close unexpectedly failed: %v", err)
		}
	}()

	rows, err := stmt.Query(int32(1), int32(10))
	if err != nil {
		t.Fatalf("stmt.Query unexpectedly failed: %v", err)
	}

	rowCount := int64(0)

	for rows.Next() {
		rowCount++

		var s string
		var n int64
		if err := rows.Scan(&s, &n); err != nil {
			t.Fatalf("rows.Scan unexpectedly failed: %v", err)
		}
		if s != "foo" {
			t.Errorf(`Expected "foo", received "%v"`, s)
		}
		if n != rowCount {
			t.Errorf("Expected %d, received %d", rowCount, n)
		}
	}
	err = rows.Err()
	if err != nil {
		t.Fatalf("rows.Err unexpectedly is: %v", err)
	}
	if rowCount != 10 {
		t.Fatalf("Expected to receive 10 rows, instead received %d", rowCount)
	}

	err = rows.Close()
	if err != nil {
		t.Fatalf("rows.Close unexpectedly failed: %v", err)
	}
}

func TestQueryCloseRowsEarly(t *testing.T) {
	db := openDB(t)
	defer closeDB(t, db)

	stmt, err := db.Prepare("select 'foo', n from generate_series($1::int, $2::int) n")
	if err != nil {
		t.Fatalf("db.Prepare unexpectedly failed: %v", err)
	}
	defer func() {
		err = stmt.Close()
		if err != nil {
			t.Fatalf("stmt.Close unexpectedly failed: %v", err)
		}
	}()

	rows, err := stmt.Query(int32(1), int32(10))
	if err != nil {
		t.Fatalf("stmt.Query unexpectedly failed: %v", err)
	}

	// Close rows immediately without having read them
	err = rows.Close()
	if err != nil {
		t.Fatalf("rows.Close unexpectedly failed: %v", err)
	}

	// Run the query again to ensure the connection and statement are still ok
	rows, err = stmt.Query(int32(1), int32(10))
	if err != nil {
		t.Fatalf("stmt.Query unexpectedly failed: %v", err)
	}

	rowCount := int64(0)

	for rows.Next() {
		rowCount++

		var s string
		var n int64
		if err := rows.Scan(&s, &n); err != nil {
			t.Fatalf("rows.Scan unexpectedly failed: %v", err)
		}
		if s != "foo" {
			t.Errorf(`Expected "foo", received "%v"`, s)
		}
		if n != rowCount {
			t.Errorf("Expected %d, received %d", rowCount, n)
		}
	}
	err = rows.Err()
	if err != nil {
		t.Fatalf("rows.Err unexpectedly is: %v", err)
	}
	if rowCount != 10 {
		t.Fatalf("Expected to receive 10 rows, instead received %d", rowCount)
	}

	err = rows.Close()
	if err != nil {
		t.Fatalf("rows.Close unexpectedly failed: %v", err)
	}
}

func TestExec(t *testing.T) {
	db := openDB(t)
	defer closeDB(t, db)

	_, err := db.Exec("create temporary table t(a varchar not null)")
	if err != nil {
		t.Fatalf("db.Exec unexpectedly failed: %v", err)
	}

	result, err := db.Exec("insert into t values('hey')")
	if err != nil {
		t.Fatalf("db.Exec unexpectedly failed: %v", err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		t.Fatalf("result.RowsAffected unexpectedly failed: %v", err)
	}
	if n != 1 {
		t.Fatalf("Expected 1, received %d", n)
	}
}

func TestTransactionLifeCycle(t *testing.T) {
	db := openDB(t)
	defer closeDB(t, db)

	_, err := db.Exec("create temporary table t(a varchar not null)")
	if err != nil {
		t.Fatalf("db.Exec unexpectedly failed: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("db.Begin unexpectedly failed: %v", err)
	}

	_, err = tx.Exec("insert into t values('hi')")
	if err != nil {
		t.Fatalf("tx.Exec unexpectedly failed: %v", err)
	}

	err = tx.Rollback()
	if err != nil {
		t.Fatalf("tx.Rollback unexpectedly failed: %v", err)
	}

	var n int64
	err = db.QueryRow("select count(*) from t").Scan(&n)
	if err != nil {
		t.Fatalf("db.QueryRow.Scan unexpectedly failed: %v", err)
	}
	if n != 0 {
		t.Fatalf("Expected 0 rows due to rollback, instead found %d", n)
	}

	tx, err = db.Begin()
	if err != nil {
		t.Fatalf("db.Begin unexpectedly failed: %v", err)
	}

	_, err = tx.Exec("insert into t values('hi')")
	if err != nil {
		t.Fatalf("tx.Exec unexpectedly failed: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Fatalf("tx.Commit unexpectedly failed: %v", err)
	}

	err = db.QueryRow("select count(*) from t").Scan(&n)
	if err != nil {
		t.Fatalf("db.QueryRow.Scan unexpectedly failed: %v", err)
	}
	if n != 1 {
		t.Fatalf("Expected 1 rows due to rollback, instead found %d", n)
	}
}