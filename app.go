package main

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/exp/slog"
)

type application struct {
	*tview.Application
	pages   *tview.Pages
	table   *tview.Table
	content tableContent
	db      *DB
}

type tableContent struct {
	*tview.TableContentReadOnly
	logs          []log
	columns       []string
	selectedLogId int64
}

func newApplication(db *DB) *tview.Application {
	app := application{db: db, content: tableContent{selectedLogId: -1}}

	app.pages = tview.NewPages()

	app.table = tview.NewTable()
	app.table.SetSelectable(true, false)
	app.table.SetContent(&app.content)
	app.table.SetSelectionChangedFunc(
		func(row, col int) { app.content.selectionChanged(row, col) },
	)
	app.table.SetFixed(1, 2)

	app.pages.AddPage("main", app.table, true, true)

	app.Application = tview.NewApplication()
	app.SetRoot(app.pages, true)
	app.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		key := e.Key()
		slog.Debug("received key event", "key", key)
		switch key {
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
		app.updateMain()
	}
}

func (app *application) updateMain() {
	logs, err := app.db.queryLogs(time.Time{}, time.Now())
	if err != nil {
		slog.Error(err.Error())
		return
	}

	app.QueueUpdateDraw(func() {
		app.content.logs = logs
		app.content.columns = []string{}

		if len(app.content.logs) > 0 {
			if app.content.selectedLogId == -1 {
				slog.Debug("first select", "row", 1)
				app.table.Select(1, 0)
				app.table.SetOffset(0, 0)
			} else {
				for i, log := range logs {
					if log.id == app.content.selectedLogId {
						slog.Debug("select", "row", i+1)
						app.table.Select(i+1, 0)
						break
					}
				}
			}
		}
	})
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

func (tc *tableContent) GetCell(row, col int) *tview.TableCell {
	if row == 0 {
		return tc.getHeaderCell(row, col)
	} else {
		return tc.getContentCell(row-1, col)
	}
}

func (tc *tableContent) getHeaderCell(row, col int) *tview.TableCell {
	cell := tview.NewTableCell("").
		SetTextColor(tcell.ColorBlack).
		SetBackgroundColor(tcell.ColorPurple).
		SetSelectable(false)
	switch col {
	case 0:
		cell.SetText("level")
	case 1:
		cell.SetText("timestamp")
	case 2:
		cell.SetText("message")
		cell.SetMaxWidth(80)
		cell.SetExpansion(1)
	default:
		cell.SetText(tc.columns[col-3])
	}
	return cell
}

func (tc *tableContent) getContentCell(row, col int) *tview.TableCell {
	cell := tview.NewTableCell("")
	log := tc.logs[row]
	switch col {
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
		v := log.data[tc.columns[col]]
		if v != nil {
			cell.SetText(fmt.Sprint(v))
		}
	}
	return cell
}

func (tc *tableContent) selectionChanged(row, col int) {
	tc.selectedLogId = tc.logs[row-1].id
}

func (tc *tableContent) GetRowCount() int {
	return len(tc.logs) + 1
}

func (tc *tableContent) GetColumnCount() int {
	return len(tc.columns) + 3
}
