package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type DB struct {
	sqlDB      *sql.DB
	appendStmt *sql.Stmt
	statsStmt  *sql.Stmt
}

type stats struct {
	count        int64
	minTimestamp time.Time
	maxTimestamp time.Time
}

const sqliteTimeLayout = "2006-01-02 15:04:05.999999999-07:00"

func newDatabase(sqlDB *sql.DB) (DB, error) {
	db := DB{sqlDB: sqlDB}

	_, err := sqlDB.Exec(
		"CREATE TABLE logs(timestamp DATETIME NOT NULL, data TEXT);" +
			"CREATE INDEX logs__timestamp ON logs(timestamp)",
	)
	if err != nil {
		return DB{}, fmt.Errorf("couldn't create schema: %w", err)
	}

	db.appendStmt, err = sqlDB.Prepare(
		"INSERT INTO logs(timestamp, data) VALUES (:timestamp, json(:data))",
	)
	if err != nil {
		return DB{}, fmt.Errorf("couldn't create append statement: %w", err)
	}

	db.statsStmt, err = sqlDB.Prepare(
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

var invalidCharacters = regexp.MustCompile(`[^\w\d]+`)

func (db DB) appendLog(logJSON []byte) error {
	var logData map[string]any
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
			queryBuilder.WriteString(
				`CREATE INDEX IF NOT EXISTS "logs__` +
					invalidCharacters.ReplaceAllString(name, "_") +
					`" ON logs(json_extract(data, '$.` +
					name +
					"'));\n",
			)
		}
		if queryBuilder.Len() > 0 {
			_, err = db.sqlDB.Exec(queryBuilder.String())
			if err != nil {
				return fmt.Errorf("couldn't create indexes: %w", err)
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
	row := db.statsStmt.QueryRow()
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

	minTimestamp, err := time.Parse(sqliteTimeLayout, minTimestampString)
	if err != nil {
		return stats{}, fmt.Errorf("couldn't parse min timestamp: %w", err)
	}

	maxTimestamp, err := time.Parse(sqliteTimeLayout, maxTimestampString)
	if err != nil {
		return stats{}, fmt.Errorf("couldn't parse max timestamp: %w", err)
	}

	return stats{
		count:        count,
		minTimestamp: minTimestamp,
		maxTimestamp: maxTimestamp,
	}, nil
}

func parseTime(t any) (time.Time, error) {
	switch t := t.(type) {
	case int:
		return time.UnixMilli(int64(t)), nil
	case int8:
		return time.UnixMilli(int64(t)), nil
	case int16:
		return time.UnixMilli(int64(t)), nil
	case int32:
		return time.UnixMilli(int64(t)), nil
	case int64:
		return time.UnixMilli(t), nil
	case uint:
		return time.UnixMilli(int64(t)), nil
	case uint8:
		return time.UnixMilli(int64(t)), nil
	case uint16:
		return time.UnixMilli(int64(t)), nil
	case uint32:
		return time.UnixMilli(int64(t)), nil
	case float32:
		return time.UnixMilli(int64(t)), nil
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
		ts, err = time.Parse(time.Stamp, t)
		if err == nil {
			return ts, nil
		}
		return time.Time{}, fmt.Errorf("failed to parse timestamp: \"%s\"", t)
	default:
		return time.Time{}, fmt.Errorf("invalid type for timestamp: %T", t)
	}
}

func collectPropNames(m map[string]any) []string {
	propNames := make([]string, 0, len(m))
	for name, child := range m {
		if childMap, ok := child.(map[string]any); ok && len(childMap) > 0 {
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
