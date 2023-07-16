package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"
)

const RFC3339Milli = "2006-01-02T15:04:05.999Z07:00"

var timeFormats = map[string]string{
	"Layout":      time.Layout,
	"ANSIC":       time.ANSIC,
	"UnixDate":    time.UnixDate,
	"RubyDate":    time.RubyDate,
	"RFC822":      time.RFC822,
	"RFC822Z":     time.RFC822Z,
	"RFC850":      time.RFC850,
	"RFC1123":     time.RFC1123,
	"RFC1123Z":    time.RFC1123Z,
	"RFC3339":     time.RFC3339,
	"RFC3339Nano": time.RFC3339Nano,
	"Kitchen":     time.Kitchen,
	"Stamp":       time.Stamp,
	"StampMilli":  time.StampMilli,
	"StampMicro":  time.StampMicro,
	"StampNano":   time.StampNano,
	"DateTime":    time.DateTime,
	"DateOnly":    time.DateOnly,
	"TimeOnly":    time.TimeOnly,
}

func main() {
	timestamp := flag.String("time", "timestamp", "JSON key for log timestamp")
	timeFmt := flag.String("time-fmt", RFC3339Milli, "go time layout string or constant")
	msg := flag.String("msg", "msg", "JSON key for log message")
	interval := flag.Int64("interval", 1000, "time interval in milliseconds between logs")

	flag.Parse()

	for range time.NewTicker(time.Duration(*interval) * time.Millisecond).C {
		logData := map[string]any{}
		logData[*timestamp] = time.Now().Format(*timeFmt)
		logData[*msg] = fmt.Sprintf("It's %s", time.Now().Format("15:04:05.999"))
		logJSON, err := json.Marshal(logData)
		if err != nil {
			panic(err.Error())
		}
		fmt.Println(string(logJSON))
	}
}
