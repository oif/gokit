package sqlbuilder_test

import (
	"database/sql"
	"errors"
	"github.com/go-sql-driver/mysql"
	"github.com/oif/gokit/sqlbuilder"
	"net"
	"os"
	"testing"
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

func TestQuery(t *testing.T) {
	query := sqlbuilder.Select(TestMySQLAccount{}).From("user").Where(`host != "%"`)
	queryString, err := query.Build()
	if err != nil {
		t.Fatalf("Failed to build sql string: %s", err)
	}
	t.Logf("SQL string: %s", *queryString)

	// Query
	row, err := engine.Query(*queryString)
	if err != nil {
		t.Fatalf("Failed to query: %s", err)
	}
	result, err := query.ScanRows(row)
	if err != nil {
		t.Fatalf("Failed to scan result: %s", err)
	}
	var accounts []TestMySQLAccount
	for _, res := range result {
		if parsed, ok := res.(TestMySQLAccount); ok {
			accounts = append(accounts, parsed)
		}
	}
	t.Logf("Got %d accounts", len(accounts))
	for _, account := range accounts {
		t.Logf("User: %s, Host: %s\n", account.User, net.IP(account.Host).String())
	}
}
