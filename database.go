package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	sqlDB        *sql.DB
	columns      map[string]struct{}
	appendStmt   *sql.Stmt
	logCountStmt *sql.Stmt
}

type stats struct {
	count        int64
	minTimestamp time.Time
	maxTimestamp time.Time
}

const sqlite3TimeLayout = "2006-01-02 03:04:05.999999999-07:00"

func newDatabase() (DB, error) {
	sqlDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return DB{}, fmt.Errorf("couldn't open database: %w", err)
	}

	_, err = sqlDB.Exec(
		"CREATE TABLE IF NOT EXISTS logs(timestamp DATETIME NOT NULL, data TEXT);" +
			"CREATE INDEX IF NOT EXISTS logs__timestamp ON logs(timestamp)",
	)
	if err != nil {
		return DB{}, fmt.Errorf("couldn't create schema: %w", err)
	}

	db := DB{sqlDB: sqlDB, columns: make(map[string]struct{})}

	db.appendStmt, err = sqlDB.Prepare(
		"INSERT INTO logs(timestamp, data) VALUES (:timestamp, json(:data))",
	)
	if err != nil {
		return DB{}, fmt.Errorf("couldn't create append statement: %w", err)
	}

	db.logCountStmt, err = sqlDB.Prepare(
		"SELECT " +
			"COUNT(timestamp) AS count, " +
			"MIN(timestamp) AS minTimestamp, " +
			"MAX(timestamp) AS maxTimestamp " +
			"FROM logs",
	)
	if err != nil {
		return DB{}, fmt.Errorf("couldn't create count statement: %w", err)
	}

	return db, nil
}

func (db DB) appendLog(logJSON []byte) error {
	var logData map[string]interface{}
	err := json.Unmarshal(logJSON, &logData)
	if err != nil {
		return fmt.Errorf("invalid json data: %w", err)
	}

	timestampData, ok := logData["timestamp"]
	if !ok {
		timestampData, ok = logData["time"]
	}
	if !ok {
		timestampData, ok = logData["date"]
	}

	var timestamp time.Time
	if ok {
		timestamp, _ = parseTime(timestampData)
	}

	if timestamp.IsZero() {
		timestamp = time.Now()
	}

	propNames := collectPropNames(logData)
	if len(propNames) > 0 {
		queryBuilder := strings.Builder{}
		for _, name := range propNames {
			if _, ok := db.columns[name]; !ok {
				queryBuilder.WriteString(
					"CREATE INDEX IF NOT EXISTS \"logs__" +
						strings.ReplaceAll(name, ".", "_") +
						"\" ON logs(json_extract(data, '$." +
						name +
						"'));\n",
				)
			}
		}
		if queryBuilder.Len() > 0 {
			_, err = db.sqlDB.Exec(queryBuilder.String())
			if err != nil {
				return fmt.Errorf("couldn't create index on new field: %w", err)
			}

			for _, name := range propNames {
				db.columns[name] = struct{}{}
			}
		}
	}

	_, err = db.appendStmt.Exec(
		sql.Named("timestamp", timestamp.UTC()),
		sql.Named("data", logJSON),
	)
	if err != nil {
		return fmt.Errorf("couldn't add log to database: %w", err)
	}

	return nil
}

func (db DB) stats() (stats, error) {
	row := db.logCountStmt.QueryRow()
	err := row.Err()
	if err != nil {
		return stats{}, fmt.Errorf("couldn't query stats: %w", err)
	}

	var count int64
	var minTimestampString, maxTimestampString string
	err = row.Scan(&count, &minTimestampString, &maxTimestampString)
	if err != nil {
		return stats{}, fmt.Errorf("couldn't read stats: %w", err)
	}

	minTimestamp, err := time.Parse(sqlite3TimeLayout, minTimestampString)
	if err != nil {
		return stats{}, fmt.Errorf("couldn't parse min timestamp: %w", err)
	}

	maxTimestamp, err := time.Parse(sqlite3TimeLayout, maxTimestampString)
	if err != nil {
		return stats{}, fmt.Errorf("couldn't parse max timestamp: %w", err)
	}

	return stats{
		count:        count,
		minTimestamp: minTimestamp,
		maxTimestamp: maxTimestamp,
	}, nil
}

func parseTime(t interface{}) (time.Time, error) {
	switch t := t.(type) {
	case float64:
		return time.UnixMilli(int64(t)), nil
	case string:
		ts, err := time.Parse(time.RFC3339Nano, t)
		if err == nil {
			return ts, nil
		}
		tsInt, err := strconv.Atoi(t)
		if err == nil {
			return time.UnixMilli(int64(tsInt)), nil
		}
		ts, err = time.Parse(time.StampNano, t)
		if err == nil {
			return ts, nil
		}
		return time.Time{}, fmt.Errorf("failed to parse timestamp: \"%s\"", t)
	default:
		return time.Time{}, fmt.Errorf("invalid type for timestamp: %T", t)
	}
}

func collectPropNames(m map[string]interface{}) []string {
	propNames := make([]string, 0, len(m))
	for name, child := range m {
		if childMap, ok := child.(map[string]interface{}); ok {
			childrenNames := collectPropNames(childMap)
			for _, childName := range childrenNames {
				propNames = append(propNames, name+"."+childName)
			}
		} else {
			propNames = append(propNames, name)
		}
	}
	return propNames
}
