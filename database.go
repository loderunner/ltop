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
}

type log struct {
	level     string
	timestamp time.Time
	message   string
	data      map[string]any
}

const sqliteTimeLayout = "2006-01-02 15:04:05.999999999-07:00"

func newDatabase(sqlDB *sql.DB) (*DB, error) {
	db := DB{sqlDB: sqlDB}

	logger.Info("creating table and indexes")
	_, err := sqlDB.Exec(
		"CREATE TABLE logs(timestamp DATETIME NOT NULL, level TEXT, data TEXT);" +
			"CREATE INDEX logs__timestamp ON logs(timestamp);" +
			"CREATE INDEX logs__level ON logs(level)",
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't create schema: %w", err)
	}

	logger.Info("preparing insert statement")
	db.appendStmt, err = sqlDB.Prepare(
		"INSERT INTO logs(timestamp, level, data) VALUES (:timestamp, :level, json(:data))",
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't prepare append statement: %w", err)
	}

	return &db, nil
}

var invalidCharacters = regexp.MustCompile(`[^\w\d]+`)

func (db *DB) appendLog(logJSON []byte) error {
	var logData map[string]any

	logger.Info("unmarshaling JSON log")
	err := json.Unmarshal(logJSON, &logData)
	if err != nil {
		return fmt.Errorf("invalid json data: %w", err)
	}

	logger.Info("reading timestamp data")
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

	logger.Info("reading level data")
	levelData, ok := logData["level"].(string)
	if !ok {
		levelData, ok = logData["lvl"].(string)
	}
	var level string
	if ok {
		switch strings.ToLower(levelData) {
		case "error", "erro", "err":
			level = "error"
		case "warning", "warn":
			level = "warn"
		case "info":
			level = "info"
		case "debug", "debu":
			level = "debug"
		default:
			level = levelData
		}
	}

	logger.Info("collecting prop names")
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
			logger.Info("creating prop indexes", "prop_names", propNames)
			_, err = db.sqlDB.Exec(queryBuilder.String())
			if err != nil {
				return fmt.Errorf("couldn't create indexes: %w", err)
			}
		}
	}

	logger.Info("inserting log in database", "timestamp", timestamp, "level", level)
	_, err = db.appendStmt.Exec(
		sql.Named("timestamp", timestamp.UTC()),
		sql.Named("level", level),
		sql.Named("data", logJSON),
	)
	if err != nil {
		return fmt.Errorf("couldn't add log to database: %w", err)
	}

	return nil
}

func (db *DB) queryLogs(from, to time.Time) ([]log, error) {
	logger.Info("querying logs", "from", from, "to", to)
	rows, err := db.sqlDB.Query("SELECT timestamp, level, data"+
		" FROM logs"+
		" WHERE timestamp BETWEEN ? AND ?",
		from,
		to,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	logs := []log{}
	for rows.Next() {
		var ts time.Time
		var level string
		var logJSON []byte

		err := rows.Scan(&ts, &level, &logJSON)
		if err != nil {
			// skip invalid row
			continue
		}

		var logData map[string]any
		err = json.Unmarshal(logJSON, &logData)
		if err != nil {
			logs = append(logs, log{timestamp: ts, message: string(logJSON)})
			continue
		}

		if msg, ok := logData["message"]; ok {
			logs = append(logs, log{
				timestamp: ts,
				message:   fmt.Sprint(msg),
				data:      logData,
			})
			continue
		}
		if msg, ok := logData["msg"]; ok {
			logs = append(logs, log{
				timestamp: ts,
				level:     level,
				message:   fmt.Sprint(msg),
				data:      logData,
			})
			continue
		}

		logs = append(logs, log{
			timestamp: ts,
			message:   string(logJSON),
			data:      logData,
		})
	}

	return logs, nil
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
			return ts.AddDate(time.Now().Year(), 0, 0), nil
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
