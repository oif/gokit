package sqlbuilder_test

import (
	"database/sql"
	"errors"
	"fmt"
	"net"
	"os"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/oif/gokit/sqlbuilder"
)

var engine *sql.DB

func TestMain(m *testing.M) {
	setup()
	defer teardown()
	os.Exit(m.Run())
}

func setup() {
	var err error
	testConfig := &mysql.Config{
		User:                 "root",
		Passwd:               "123456",
		Net:                  "tcp",
		Addr:                 "127.0.0.1:13306",
		DBName:               "mysql",
		AllowNativePasswords: true,
	}
	engine, err = sql.Open("mysql", testConfig.FormatDSN())
	if err != nil {
		panic(err)
	}
	if err = engine.Ping(); err != nil {
		panic(err)
	}
}

func teardown() {
	if engine != nil {
		engine.Close()
	}
}

type TestMySQLAccount struct {
	User string `sb:"user"`
	Host Host   `sb:"host"`
}

type Host net.IP

func (h *Host) Scan(src interface{}) error {
	var castedSource string
	switch src.(type) {
	case string:
		castedSource = src.(string)
	case []byte:
		castedSource = string(src.([]byte))
	default:
		return errors.New("unknown labels source type")
	}
	if castedSource == "localhost" {
		castedSource = "127.0.0.1"
	}
	*h = Host(net.ParseIP(castedSource))
	return nil
}

func sqlBuilderSelectQuery() ([]TestMySQLAccount, error) {
	query := sqlbuilder.Select(TestMySQLAccount{}).From("user").Where(`host != "%"`)
	queryString, err := query.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build sql string: %s", err)
	}
	return sqlBuilderSelectQueryWithSQL(query, *queryString)
}

func sqlBuilderSelectQueryWithSQL(query *sqlbuilder.SQL, queryString string) ([]TestMySQLAccount, error) {
	// Query
	rows, err := engine.Query(queryString)
	if err != nil {
		return nil, fmt.Errorf("failed to query: %s", err)
	}
	result, err := query.ScanRows(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to scan result: %s", err)
	}
	var accounts []TestMySQLAccount
	for _, res := range result {
		if parsed, ok := res.(TestMySQLAccount); ok {
			accounts = append(accounts, parsed)
		}
	}
	return accounts, nil
}

func nativeSelectQuery() ([]TestMySQLAccount, error) {
	// Read direct
	rows, err := engine.Query(`SELECT user, host FROM user WHERE host != "%"`)
	if err != nil {
		return nil, fmt.Errorf("failed to query direct: %s", err)
	}
	var accounts []TestMySQLAccount
	for rows.Next() {
		var account TestMySQLAccount
		err = rows.Scan(&account.User, &account.Host)
		if err != nil {
			return nil, fmt.Errorf("failed to scan: %s", err)
		}
		accounts = append(accounts, account)
	}
	return accounts, nil
}

func TestSelectQuery(t *testing.T) {
	nativeSelectAccounts, err := nativeSelectQuery()
	if err != nil {
		t.Fatal(err)
	}
	SQLBuilderSelectAccounts, err := sqlBuilderSelectQuery()
	if err != nil {
		t.Fatal(err)
	}
	if len(nativeSelectAccounts) != len(SQLBuilderSelectAccounts) {
		t.Fatalf("Native got %d, SQL builder got %d", len(nativeSelectAccounts), len(SQLBuilderSelectAccounts))
	}
	var hint int
	for _, na := range nativeSelectAccounts {
		for _, sa := range SQLBuilderSelectAccounts {
			if na.User == sa.User && net.IP(na.Host).Equal(net.IP(na.Host)) {
				hint++
				break
			}
		}
	}
	if hint != len(nativeSelectAccounts) {
		t.Fatalf("Result mismatch")
	}
}

func BenchmarkNativeSelectQuery(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		nativeSelectQuery()
	}
}

func BenchmarkSQLBuilderSelectQuery(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sqlBuilderSelectQuery()
	}
}

func BenchmarkSQLBuilderSelectQueryReused(b *testing.B) {
	query := sqlbuilder.Select(TestMySQLAccount{}).From("user").Where(`host != "%"`)
	queryString, err := query.Build()
	if err != nil {
		b.Fatalf("failed to build sql string: %s", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sqlBuilderSelectQueryWithSQL(query, *queryString)
	}
}
