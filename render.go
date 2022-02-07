package main

import (
	"fmt"
	"math"
	"strconv"
	"sync"

	"github.com/japanoise/termbox-util"
	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

var redrawLock *sync.Mutex

func init() {
	redrawLock = &sync.Mutex{}
}

func (row *EditorRow) cxToRx(cx int) int {
	rx := 0
	for i, rv := range row.Data {
		if i >= cx {
			break
		}
		if rv == '\t' {
			rx += Global.Tabsize
		} else {
			rx += termutil.Runewidth(rv)
		}
	}
	return rx
}

func editorRowCxToRx(row *EditorRow) int {
	return row.cxToRx(Global.CurrentB.cx)
}

func editorRowRxToCx(row *EditorRow, rx int) int {
	cur_rx := 0
	var cx int
	for cx = 0; cx < row.Size; cx++ {
		if row.Data[cx] == '\t' {
			cur_rx += Global.Tabsize
		} else {
			cur_rx++
		}
		if cur_rx > rx {
			return cx
		}
	}
	return cx
}

func editorRefreshScreen() {
	redrawLock.Lock()
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	sx, sy := termbox.Size()
	Global.WindowTree.draw(0, 0, sx, sy-2)
	editorDrawPrompt(sy)
	termbox.Flush()
	redrawLock.Unlock()
}

func trimString(s string, coloff int) (string, int) {
	if coloff == 0 {
		return s, 0
	}
	if coloff < len(s) {
		rw := 0
		for i, ru := range s {
			if rw >= coloff {
				return s[i:], i
			}
			rw += termutil.Runewidth(ru)
		}
	}
	return "", 0
}

func editorDrawRows(startx, starty, sx, sy int, buf *EditorBuffer, gutsize int) {
	for y := starty; y < sy; y++ {
		filerow := (y - starty) + buf.rowoff
		if filerow >= buf.NumRows {
			if buf.hasMode("tilde-mode") {
				termbox.SetCell(startx+gutsize, y, '~', termbox.ColorBlue, termbox.ColorDefault)
			}
		} else {
			row := buf.Rows[filerow]
			if gutsize > 0 {
				if buf.hasMode("gdi") {
					termutil.Printstring(string(buf.Rows[filerow].idx), startx, y)
				} else {
					termutil.Printstring(runewidth.FillLeft(LineNrToString(buf.Rows[filerow].idx+1), gutsize-2), startx, y)
				}
				termutil.PrintRune(startx+gutsize-2, y, '│', termbox.ColorDefault)
				if row.coloff > 0 {
					termutil.PrintRune(startx+gutsize-1, y, '←', termbox.ColorDefault)
				}
			}
			if row.coloff < row.RenderSize {
				ts, off := trimString(row.Render, row.coloff)
				row.Print(startx+gutsize, y, row.coloff, off, sx-gutsize, ts, buf)
			}
			if row.coloff > 0 && gutsize == 0 {
				termutil.PrintRune(startx, y, '←', termbox.ColorDefault)
			}
		}
	}
}

func editorDrawRowsFocused(startx, starty, sx, sy int, buf *EditorBuffer, gutsize int) {
	termbox.SetCursor(startx, starty)
	for y := starty; y < sy; y++ {
		filerow := (y - starty) + buf.rowoff
		if filerow >= buf.NumRows {
			if buf.hasMode("tilde-mode") {
				termbox.SetCell(startx+gutsize, y, '~', termbox.ColorBlue, termbox.ColorDefault)
			}
		} else {
			row := buf.Rows[filerow]
			if gutsize > 0 {
				if buf.hasMode("gdi") {
					termutil.Printstring(string(buf.Rows[filerow].idx), startx, y)
				} else {
					termutil.Printstring(runewidth.FillLeft(LineNrToString(buf.Rows[filerow].idx+1), gutsize-2), startx, y)
				}
				termutil.PrintRune(startx+gutsize-2, y, '│', termbox.ColorDefault)
				if row.coloff > 0 {
					termutil.PrintRune(startx+gutsize-1, y, '←', termbox.ColorDefault)
				}
			}
			if filerow == buf.cy {
				termbox.SetCursor(0, y)
			}
			if row.coloff < row.RenderSize {
				ts, off := trimString(row.Render, row.coloff)
				if filerow == buf.cy {
					row.PrintWCursor(startx+gutsize, y, row.coloff, off, sx-gutsize, ts, buf)
				} else {
					row.Print(startx+gutsize, y, row.coloff, off, sx-gutsize, ts, buf)
				}
			}
			if row.coloff > 0 && gutsize == 0 {
				termutil.PrintRune(startx, y, '←', termbox.ColorDefault)
			}
		}
	}
}

func editorUpdateStatus(buf *EditorBuffer) string {
	fn := buf.getRenderName()
	dc := '-'
	if buf.Dirty {
		dc = '*'
	}
	return fmt.Sprintf("-%c %s - (%s) %d:%d", dc, fn, buf.MajorMode,
		buf.cy+1, buf.Rows[buf.cy].cxToRx(buf.cx))
}

func GetScreenSize() (int, int) {
	x, _ := termbox.Size()
	return x, Global.CurrentBHeight
}

func editorDrawStatusLine(x, y, wx int, t *winTree) {
	buf := t.buf
	// Get & draw the standard Emacs line
	line := editorUpdateStatus(buf)
	rx := x
	maxrx := x + wx
	for _, ru := range line {
		termbox.SetCell(rx, y, ru,
			termbox.ColorDefault|termbox.AttrReverse,
			termbox.ColorDefault)
		rx += termutil.Runewidth(ru)
		if rx >= maxrx {
			// Truncate if it's too long
			termbox.SetCell(rx, y, ' ',
				termbox.ColorDefault|termbox.AttrReverse,
				termbox.ColorDefault)
			return
		}
	}
	termbox.SetCell(rx, y, ' ', termbox.ColorDefault|termbox.AttrReverse,
		termbox.ColorDefault)

	// Draw some flexi space
	var ix int
	for ix = rx + 1; ix < maxrx-7; ix++ {
		if t.focused {
			termbox.SetCell(ix, y, '-',
				termbox.ColorDefault|termbox.AttrReverse,
				termbox.ColorDefault)
		} else {
			termbox.SetCell(ix, y, ' ',
				termbox.ColorDefault|termbox.AttrReverse,
				termbox.ColorDefault)
		}
	}

	// Draw the end label (%age through the buffer)
	el := calcEndLabel(buf)
	for _, ru := range el {
		termbox.SetCell(ix, y, ru,
			termbox.ColorDefault|termbox.AttrReverse,
			termbox.ColorDefault)
		ix++
		if ix >= maxrx {
			// Truncate if it's too long
			return
		}
	}
	for ix < maxrx {
		if t.focused {
			termbox.SetCell(ix, y, '-',
				termbox.ColorDefault|termbox.AttrReverse,
				termbox.ColorDefault)
		} else {
			termbox.SetCell(ix, y, ' ',
				termbox.ColorDefault|termbox.AttrReverse,
				termbox.ColorDefault)
		}
		ix++
	}
}

func calcEndLabel(buf *EditorBuffer) string {
	if buf.NumRows == 0 {
		return " Emp "
	} else if Global.CurrentBHeight >= buf.NumRows {
		return " All "
	} else if buf.rowoff+Global.CurrentBHeight >= buf.NumRows {
		return " Bot "
	} else if buf.rowoff == 0 {
		return " Top "
	} else {
		perc := float64(buf.rowoff) / float64(buf.NumRows)
		return fmt.Sprintf(" %2d%% ", int(perc*100))
	}
}

func editorDrawPrompt(y int) {
	termutil.Printstring(Global.Prompt+"-> "+Global.Input, 0, y-1)
}

func NumStrWidth(num int) int {
	return int(math.Log10(float64(num))) + 1
}

func GetGutterWidth(NumRows int) int {
	return NumStrWidth(NumRows) + 2
}

func LineNrToString(num int) string {
	return strconv.Itoa(num)
}
