package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type application struct {
	*tview.Application
	pages    *tview.Pages
	logsView *tview.TextView
	table    *tview.Table
	content  tableContent
	db       *DB
}

type tableContent struct {
	*tview.TableContentReadOnly
	logs    []log
	columns []string
}

func newApplication(db *DB) *tview.Application {
	app := application{db: db}

	app.pages = tview.NewPages()

	app.table = tview.NewTable()
	app.table.SetSelectable(true, false)
	app.table.SetContent(&app.content)
	app.table.SetFixed(1, 2)

	app.pages.AddPage("main", app.table, true, true)

	app.logsView = tview.NewTextView()

	app.pages.AddPage("logs", app.logsView, true, false)

	app.Application = tview.NewApplication()
	app.SetRoot(app.pages, true)
	app.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		key := e.Key()
		logger.Info("received key event", "key", key)
		switch key {
		case tcell.KeyF1:
			app.pages.SwitchToPage("main")
		case tcell.KeyF2:
			app.pages.SwitchToPage("logs")
		case tcell.KeyRune:
			if e.Rune() == 'q' {
				app.Stop()
				return nil
			}
		}
		return e
	})

	go app.pollingLoop()

	return app.Application
}

func (app *application) pollingLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		page, _ := app.pages.GetFrontPage()
		switch page {
		case "main":
			logs, err := app.db.queryLogs(time.Time{}, time.Now())
			if err != nil {
				logger.Error(err.Error())
				return
			}

			wasEmpty := len(app.content.logs) == 0
			app.content.logs = logs
			app.content.columns = []string{}
			sort.Stable(sort.StringSlice(app.content.columns))
			if wasEmpty && len(app.content.logs) > 0 {
				app.table.Select(1, 0)
				app.table.ScrollToBeginning()
			}
		case "logs":
			app.logsView.SetText(logBuf.String())
		}
		app.QueueUpdateDraw(func() {})
	}
}

func (tc *tableContent) getColumns() []string {
	columnSet := map[string]struct{}{}
	for _, log := range tc.logs {
		for k, v := range log.data {
			if k == "timestamp" || k == "time" || k == "date" {
				continue
			}
			if k == "level" || k == "lvl" {
				continue
			}
			if k == "msg" || k == "message" {
				continue
			}
			if v == nil || v == "" {
				continue
			}
			columnSet[k] = struct{}{}
		}
	}
	columns := make([]string, 0, len(columnSet))
	for k := range columnSet {
		columns = append(columns, k)
	}
	return columns
}

func (tc *tableContent) GetCell(row, column int) *tview.TableCell {
	if row == 0 {
		return tc.getHeaderCell(row, column)
	} else {
		return tc.getContentCell(row-1, column)
	}
}

func (tc *tableContent) getHeaderCell(row int, column int) *tview.TableCell {
	var text string
	switch column {
	case 0:
		text = "level"
	case 1:
		text = "timestamp"
	case 2:
		text = "message"
	default:
		text = tc.columns[column-3]
	}
	return tview.NewTableCell(text).
		SetTextColor(tcell.ColorBlack).
		SetBackgroundColor(tcell.ColorPurple).
		SetSelectable(false)
}

func (tc *tableContent) getContentCell(row int, column int) *tview.TableCell {
	cell := tview.NewTableCell("")
	log := tc.logs[row]
	switch column {
	case 0:
		cell.SetText(log.level)
		switch log.level {
		case "error":
			cell.SetTextColor(tcell.ColorBlack)
			cell.SetBackgroundColor(tcell.ColorRed)
		case "warn":
			cell.SetTextColor(tcell.ColorBlack)
			cell.SetBackgroundColor(tcell.ColorYellow)
		case "info":
			cell.SetTextColor(tcell.ColorBlack)
			cell.SetBackgroundColor(tcell.ColorBlue)
		case "debug":
			cell.SetTextColor(tcell.ColorBlack)
			cell.SetBackgroundColor(tcell.ColorGreen)
		}
	case 1:
		cell.SetText(log.timestamp.Format(time.StampMilli))
	case 2:
		cell.SetText(log.message)
		cell.SetMaxWidth(80)
		cell.SetExpansion(1)
	default:
		v := log.data[tc.columns[column]]
		if v != nil {
			cell.SetText(fmt.Sprint(v))
		}
	}
	return cell
}

func (tc *tableContent) GetRowCount() int {
	return len(tc.logs)
}

func (tc *tableContent) GetColumnCount() int {
	return len(tc.columns) + 3
}
