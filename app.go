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
	table   *tview.Table
	content tableContent
}

type tableContent struct {
	tview.TableContent
	logs    []log
	columns []string
}

func newApplication(db DB) *tview.Application {
	var app application

	app.table = tview.NewTable()
	app.table.SetSelectable(true, false)
	app.table.SetContent(&app.content)
	app.table.SetFixed(1, 0)

	app.Application = tview.NewApplication()
	app.SetRoot(app.table, true)
	app.SetInputCapture(func(e *tcell.EventKey) *tcell.EventKey {
		if e.Key() == tcell.KeyRune && e.Rune() == 'q' {
			app.Stop()
			return nil
		}
		return e
	})

	go app.pollLogs(db)

	return app.Application
}

func (app *application) pollLogs(db DB) {
	var err error
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		app.content.logs, err = db.queryLogs(time.Time{}, time.Now())
		if err != nil {
			break
		}
		app.content.columns = app.content.getColumns()
		sort.Stable(sort.StringSlice(app.content.columns))
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
	return len(tc.getColumns()) + 3
}

func (tc *tableContent) SetCell(row, column int, cell *tview.TableCell) {
	panic("table content is immutable")
}

func (tc *tableContent) RemoveRow(row int) {
	panic("table content is immutable")
}

func (tc *tableContent) RemoveColumn(column int) {
	panic("table content is immutable")
}

func (tc *tableContent) InsertRow(row int) {
	panic("table content is immutable")
}

func (tc *tableContent) InsertColumn(column int) {
	panic("table content is immutable")
}

func (tc *tableContent) Clear() {
	panic("table content is immutable")
}
