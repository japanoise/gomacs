package main

import (
	"fmt"
	"github.com/japanoise/termbox-util"
	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
	"math"
	"strconv"
	"strings"
)

func editorRowCxToRx(row *EditorRow) int {
	rx := 0
	cx := Global.CurrentB.cx
	for i, rv := range row.Data {
		if i >= cx {
			break
		}
		if rv == '\t' {
			rx += (Global.Tabsize - 1) - (rx % Global.Tabsize)
			rx++
		} else {
			rx += termutil.Runewidth(rv)
		}
	}
	return rx
}

func editorRowRxToCx(row *EditorRow, rx int) int {
	cur_rx := 0
	var cx int
	for cx = 0; cx < row.Size; cx++ {
		if row.Data[cx] == '\t' {
			cur_rx += (Global.Tabsize - 1) - (cur_rx % Global.Tabsize)
		}
		cur_rx++
		if cur_rx > rx {
			return cx
		}
	}
	return cx
}

func editorRefreshScreen() {
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	x, y := termbox.Size()
	yrows := y - 2
	numwin := len(Global.Windows)
	winheight := yrows / numwin
	for i, win := range Global.Windows {
		gutter := 0
		if win.hasMode("line-number-mode") && win.NumRows > 0 {
			gutter = GetGutterWidth(win.NumRows)
		}
		starth := 0
		if i >= 1 {
			starth = 1 + winheight*i
			editorDrawStatusLine(x, winheight*i, Global.Windows[i-1])
			editorScroll(x-gutter, winheight-1)
		} else {
			editorScroll(x-gutter, winheight)
		}
		if win == Global.CurrentB {
			Global.CurrentBHeight = winheight
			termbox.SetCursor(Global.CurrentB.rx-Global.CurrentB.coloff+gutter, starth+Global.CurrentB.cy-Global.CurrentB.rowoff)
		}
		editorDrawRows(starth, winheight*(i+1)+1, win, gutter)
	}
	editorDrawStatusLine(x, y-2, Global.Windows[numwin-1])
	editorDrawPrompt(y)
	termbox.Flush()
}

func trimString(s string, coloff int) (string, int) {
	if coloff == 0 {
		return s, 0
	}
	sr := []rune(s)
	if coloff < len(sr) {
		ret := string(sr[coloff:])
		return ret, strings.Index(s, ret)
	} else {
		return "", 0
	}
}

func hlprint(s string, hl []EmacsColor, x, y int) {
	i := 0
	for in, ru := range s {
		termutil.PrintRune(x+i, y, ru, editorSyntaxToColor(hl[in]))
		i += termutil.Runewidth(ru)
	}
}

func editorDrawRows(starty, sy int, buf *EditorBuffer, gutsize int) {
	for y := starty; y < sy; y++ {
		filerow := (y - starty) + buf.rowoff
		if filerow >= buf.NumRows {
			if buf.coloff == 0 {
				termbox.SetCell(gutsize, y, '~', termbox.ColorBlue, termbox.ColorDefault)
			}
		} else {
			if gutsize > 0 {
				if buf.hasMode("gdi") {
					termutil.Printstring(string(buf.Rows[filerow].idx), 0, y)
				} else {
					termutil.Printstring(runewidth.FillLeft(LineNrToString(buf.Rows[filerow].idx), gutsize-2), 0, y)
				}
				termutil.PrintRune(gutsize-2, y, '│', termbox.ColorDefault)
				if buf.coloff > 0 {
					termutil.PrintRune(gutsize-1, y, '←', termbox.ColorDefault)
				}
			}
			if buf.coloff < buf.Rows[filerow].RenderSize {
				r, off := trimString(buf.Rows[filerow].Render, buf.coloff)
				hlprint(r, buf.Rows[filerow].Hl[off:], gutsize, y)
			}
		}
	}
}

func editorUpdateStatus(buf *EditorBuffer) string {
	fn := buf.getFilename()
	syn := "no ft"
	if buf.Syntax != nil {
		syn = buf.Syntax.filetype
	}
	if buf.Dirty {
		return fmt.Sprintf("%s [Modified] - (%s) %d:%d", fn, syn,
			buf.cy, buf.cx)
	} else {
		return fmt.Sprintf("%s - (%s) %d:%d", fn, syn,
			buf.cy, buf.cx)
	}
}

func GetScreenSize() (int, int) {
	x, _ := termbox.Size()
	return x, Global.CurrentBHeight
}

func editorDrawStatusLine(x, y int, buf *EditorBuffer) {
	line := editorUpdateStatus(buf)
	if buf == Global.CurrentB && buf.hasMode("terminal-title-mode") {
		fmt.Printf("\033]0;%s - gomacs\a", buf.getFilename())
	}
	var ru rune
	rx := 0
	for _, ru = range line {
		termbox.SetCell(rx, y, ru, termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
		rx += termutil.Runewidth(ru)
	}
	termbox.SetCell(rx, y, ' ', termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
	for ix := rx + 1; ix < x; ix++ {
		if buf == Global.CurrentB {
			termbox.SetCell(ix, y, '-', termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
		} else {
			termbox.SetCell(ix, y, ' ', termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
		}
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
