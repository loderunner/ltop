package main

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

const millis int64 = 1688596148685
const rfc3339UTC = "2023-07-05T22:29:08.685Z"
const rfc3339TZ = "2023-07-06T00:29:08.685+02:00"
const syslog = "Jul 10 22:29:08"

var timestamp = time.UnixMilli(millis).UTC()
var sqliteTimestamp = timestamp.Format(sqliteTimeLayout)

func TestCollectPropNames(t *testing.T) {
	type testCase struct {
		name   string
		in     map[string]any
		expect []string
	}

	testCases := []testCase{
		{name: "empty", in: map[string]any{}, expect: []string{}},
		{
			name:   "single field",
			in:     map[string]any{"hello": "world"},
			expect: []string{"hello"},
		},
		{
			name:   "2 fields",
			in:     map[string]any{"hello": "world", "toto": true},
			expect: []string{"hello", "toto"},
		},
		{
			name: "3 fields",
			in: map[string]any{
				"hello":      "world",
				"toto":       true,
				"properties": map[string]any{},
			},
			expect: []string{"hello", "toto", "properties"},
		},
		{
			name: "nested",
			in: map[string]any{
				"hello": "world",
				"toto":  true,
				"properties": map[string]any{
					"hello": "world",
					"dead":  0xbeef,
				},
			},
			expect: []string{
				"hello",
				"toto",
				"properties.hello",
				"properties.dead",
			},
		},
	}

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			got := collectPropNames(c.in)
			assert.ElementsMatch(t, c.expect, got)
		})
	}
}

func TestParseTime(t *testing.T) {
	type testCase struct {
		in     any
		expect time.Time
		err    bool
	}
	testCases := []testCase{
		{in: millis, expect: timestamp},
		{in: uint64(millis), err: true},
		{in: rfc3339UTC, expect: timestamp},
		{in: rfc3339TZ, expect: timestamp},
		{in: strconv.Itoa(int(millis)), expect: timestamp},
		{in: syslog, expect: time.Date(0, time.July, 10, 22, 29, 8, 0, time.UTC)},
	}
	for _, c := range testCases {
		t.Run(fmt.Sprintf("%T(%v)", c.in, c.in), func(t *testing.T) {
			got, err := parseTime(c.in)
			if c.err {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, c.expect, got.UTC())
		})
	}
}

func TestNewDatabase(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error opening mock database: %s", err)
	}
	defer sqlDB.Close()

	db := testCreateDatabase(t, sqlDB, mock)

	if db.appendStmt == nil {
		t.Fatalf("missing prepared statement for 'append'")
	}
	if db.statsStmt == nil {
		t.Fatalf("missing prepared statement for 'stats'")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %s", err)
	}
}

type testCase struct {
	name      string
	input     string
	timestamp time.Time
	indexes   []string
	err       bool
}

func TestAppendLog(t *testing.T) {
	testCases := []testCase{
		{name: "empty string", err: true},
		{name: "invalid JSON", input: "{", err: true},
		{name: "empty JSON", input: "{}"},
		{
			name:      "with timestamp",
			input:     fmt.Sprintf(`{"timestamp":%d}`, millis),
			timestamp: timestamp,
			indexes:   []string{"timestamp"},
		},
		{
			name:      "with time",
			input:     fmt.Sprintf(`{"time":%d}`, millis),
			timestamp: timestamp,
			indexes:   []string{"time"},
		},
		{
			name:      "with date",
			input:     fmt.Sprintf(`{"date":%d}`, millis),
			timestamp: timestamp,
			indexes:   []string{"date"},
		},
		{
			name: "with fields",
			input: fmt.Sprintf(
				`{"timestamp":%d, "hello": "world", "one": 1, "empty": null, "nested": {"toto": 0}}`,
				millis,
			),
			timestamp: timestamp,
			indexes:   []string{"timestamp", "hello", "one", "empty", "nested_toto"},
		},
	}

	for _, c := range testCases {
		sqlDB, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("error opening mock database: %s", err)
		}
		defer sqlDB.Close()

		db := testCreateDatabase(t, sqlDB, mock)
		t.Run(
			c.name,
			func(t *testing.T) { testAppendLog(t, c, db, mock) },
		)
	}
}

func TestStats(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("error opening mock database: %s", err)
	}
	defer sqlDB.Close()

	db := testCreateDatabase(t, sqlDB, mock)

	mock.ExpectQuery("SELECT").
		WillReturnRows(
			sqlmock.NewRowsWithColumnDefinition(
				sqlmock.NewColumn("count"),
				sqlmock.NewColumn("minTimestamp"),
				sqlmock.NewColumn("maxTimestamp"),
			).
				AddRow(1, sqliteTimestamp, sqliteTimestamp),
		)

	expect := stats{count: 1, minTimestamp: timestamp, maxTimestamp: timestamp}
	got, err := db.stats()
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	assert.Equal(t, expect.count, got.count)
	assert.Equal(t, expect.minTimestamp, got.minTimestamp.UTC())
	assert.Equal(t, expect.maxTimestamp, got.maxTimestamp.UTC())
}

func testCreateDatabase(t *testing.T, sqlDB *sql.DB, mock sqlmock.Sqlmock) DB {
	mock.
		ExpectExec("CREATE TABLE logs.*CREATE INDEX logs__timestamp ON logs").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectPrepare("INSERT INTO logs")
	mock.ExpectPrepare("SELECT .* FROM logs")

	db, err := newDatabase(sqlDB)
	if err != nil {
		t.Fatalf("error creating database: %s", err)
	}
	return db
}

type anyTime struct{}

func (a anyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

func testAppendLog(t *testing.T, c testCase, db DB, mock sqlmock.Sqlmock) {
	if len(c.indexes) > 0 {
		createIndexString := ""
		for range c.indexes {
			createIndexString += `CREATE INDEX IF NOT EXISTS "logs__.*?".*`
		}
		mock.ExpectExec(createIndexString).WillReturnResult(sqlmock.NewResult(0, 0))
	}
	expectedExec := mock.ExpectExec("INSERT INTO logs")
	if c.timestamp.IsZero() {
		expectedExec = expectedExec.WithArgs(anyTime{}, []byte(c.input))
	} else {
		expectedExec = expectedExec.WithArgs(c.timestamp, []byte(c.input))
	}
	expectedExec = expectedExec.WillReturnResult(sqlmock.NewResult(1, 1))

	err := db.appendLog([]byte(c.input))
	if c.err {
		if err == nil {
			t.Fatalf("expected error")
		}
		return
	} else {
		if err != nil {
			t.Fatalf("unexpected error appending log: %s", err)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %s", err)
	}
}
