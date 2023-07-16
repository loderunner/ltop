package main

import (
	"fmt"
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
		app.QueueUpdateDraw(func() {
			if row, _ := app.table.GetSelection(); row == 0 {
				app.table.ScrollToBeginning()
			}
		})
	}
}

func (tc *tableContent) GetCell(row, column int) *tview.TableCell {
	cell := tview.NewTableCell("")
	if column == 0 {
		cell.SetText(tc.logs[row].timestamp.Format("15:04:05.000"))
	} else if column == 1 {
		cell.SetText(tc.logs[row].message)
	} else {
		if field, ok := tc.logs[row].data[tc.columns[column]]; ok {
			cell.SetText(fmt.Sprint(field))
		}
	}
	return cell
}

func (tc *tableContent) GetRowCount() int {
	return len(tc.logs)
}

func (tc *tableContent) GetColumnCount() int {
	return len(tc.columns) + 2
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
