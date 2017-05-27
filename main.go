package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/japanoise/termbox-util"
	"github.com/mattn/go-runewidth"
	"github.com/mitchellh/go-homedir"
	"github.com/nsf/termbox-go"
	"os"
	"strings"
	"unicode/utf8"
)

type EditorRow struct {
	idx             int
	Size            int
	Data            string
	RenderSize      int
	Render          string
	Hl              []EmacsColor
	hl_open_comment bool
}

type EditorBuffer struct {
	Filename string
	Dirty    bool
	cx       int
	cy       int
	rx       int
	rowoff   int
	coloff   int
	NumRows  int
	Rows     []*EditorRow
	Syntax   *EditorSyntax
	Undo     *EditorUndo
	Redo     *EditorUndo
	MarkX    int
	MarkY    int
	Modes    ModeList
}

type EditorState struct {
	quit           bool
	Input          string
	CurrentB       *EditorBuffer
	Buffers        []*EditorBuffer
	Tabsize        int
	Prompt         string
	NoSyntax       bool
	Windows        []*EditorBuffer
	CurrentBHeight int
	Clipboard      string
	SoftTab        bool
	DefaultModes   map[string]bool
}

var Global EditorState
var Emacs *CommandList

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
	if buf.hasMode("terminal-title-mode") {
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

func editorScroll(sx, sy int) {
	Global.CurrentB.rx = 0
	if Global.CurrentB.cy < Global.CurrentB.NumRows {
		Global.CurrentB.rx = editorRowCxToRx(Global.CurrentB.Rows[Global.CurrentB.cy])
	}

	if Global.CurrentB.cy < Global.CurrentB.rowoff {
		Global.CurrentB.rowoff = Global.CurrentB.cy
	}
	if Global.CurrentB.cy >= Global.CurrentB.rowoff+sy {
		Global.CurrentB.rowoff = Global.CurrentB.cy - sy + 1
	}
	if Global.CurrentB.rx < Global.CurrentB.coloff {
		Global.CurrentB.coloff = Global.CurrentB.rx
	}
	if Global.CurrentB.rx >= Global.CurrentB.coloff+sx {
		Global.CurrentB.coloff = Global.CurrentB.rx - sx + 1
	}
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

func editorCentreView() {
	rowoff := Global.CurrentB.cy - (Global.CurrentBHeight / 2)
	if rowoff >= 0 {
		Global.CurrentB.rowoff = rowoff
	}
}

func editorSetPrompt(prompt string) {
	Global.Prompt = prompt
}

func MoveCursor(x, y int) {
	// Initial position of the cursor
	icx, icy := Global.CurrentB.cx, Global.CurrentB.cy
	// Regular cursor movement - most cases
	realline := icy < Global.CurrentB.NumRows && Global.CurrentB.NumRows != 0
	nx, ny := icx+x, icy+y
	if realline && icx <= Global.CurrentB.Rows[icy].Size {
		if x >= 1 {
			_, rs := utf8.DecodeRuneInString(Global.CurrentB.Rows[icy].Data[icx:])
			nx = icx + rs
		} else if x <= -1 {
			_, rs :=
				utf8.DecodeLastRuneInString(Global.CurrentB.Rows[icy].Data[:icx])
			nx = icx - rs
		}
	}
	if nx >= 0 && ny < Global.CurrentB.NumRows && realline && nx <= Global.CurrentB.Rows[icy].Size {
		Global.CurrentB.cx = nx
	}
	if ny >= 0 && ny <= Global.CurrentB.NumRows {
		Global.CurrentB.cy = ny
	}

	// Edge cases
	realline = Global.CurrentB.cy < Global.CurrentB.NumRows && Global.CurrentB.NumRows != 0
	if x < 0 && Global.CurrentB.cy > 0 && icx == 0 {
		// Left at the beginning of a line
		Global.CurrentB.cy--
		MoveCursorToEol()
	} else if realline && y == 0 && icx == Global.CurrentB.Rows[Global.CurrentB.cy].Size && x > 0 {
		// Right at the end of a line
		Global.CurrentB.cy++
		MoveCursorToBol()
	} else if realline && Global.CurrentB.cx > Global.CurrentB.Rows[Global.CurrentB.cy].Size {
		// Snapping to the end of the line when coming from a longer line
		MoveCursorToEol()
	} else if !realline && y == 1 {
		// Moving cursor down to the EOF
		MoveCursorToBol()
	}
}

func MoveCursorToEol() {
	if Global.CurrentB.cy < Global.CurrentB.NumRows {
		Global.CurrentB.cx = Global.CurrentB.Rows[Global.CurrentB.cy].Size
	}
}

func MoveCursorToBol() {
	Global.CurrentB.cx = 0
}

func MovePage(back bool, sy int) {
	for i := 0; i < sy; i++ {
		if back {
			MoveCursor(0, -1)
		} else {
			MoveCursor(0, 1)
		}
	}
}

func MoveCursorBackPage() {
	_, sy := GetScreenSize()
	Global.CurrentB.cy = Global.CurrentB.rowoff
	MovePage(true, sy)
}

func MoveCursorForthPage() {
	_, sy := GetScreenSize()
	Global.CurrentB.cy = Global.CurrentB.rowoff + sy - 1
	if Global.CurrentB.cy > Global.CurrentB.NumRows {
		Global.CurrentB.cy = Global.CurrentB.NumRows
	}
	MovePage(false, sy)
}

func EditorQuit() {
	nodirty := true
	for _, buf := range Global.Buffers {
		if buf.Dirty {
			ds, cancel := editorYesNoPrompt(fmt.Sprintf("%s has unsaved changes; save them?", buf.getFilename()), true)
			if ds && cancel == nil {
				editorBufSave(buf)
			} else if cancel != nil {
				return
			}
		}
		nodirty = nodirty && !buf.Dirty
	}
	if !nodirty {
		dq, cancel := editorYesNoPrompt("Unsaved buffers exist; really quit?", false)
		if dq && cancel == nil {
			Global.quit = true
		} else {
			Global.Input = "Cancelled."
		}
	} else {
		Global.quit = true
	}
}

func lineEdDrawLine(prompt, ret string, cpos int) {
	px := 0
	var ru rune
	_, sy := GetScreenSize()
	for px, ru = range prompt {
		termbox.SetCell(px, sy-1, ru, termbox.ColorDefault, termbox.ColorDefault)
	}
	px++
	termbox.SetCell(px, sy-1, '→', termbox.ColorDefault, termbox.ColorDefault)
	px++
	termbox.SetCursor(len(prompt)+3+cpos, sy-1)
}

func EditorGetline(prompt string) string {
	ret := ""
	cpos := 0
	lineEdDrawLine(prompt, ret, cpos)
	for {
		ev := termbox.PollEvent()
		if ev.Type == termbox.EventResize {
			editorRefreshScreen()
		} else if ev.Type == termbox.EventKey {
			return ret
		}
		lineEdDrawLine(prompt, ret, cpos)
	}
}

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

func editorUpdateRow(row *EditorRow) {
	tabs := 0
	for _, rv := range row.Data {
		if rv == '\t' {
			tabs++
		}
	}
	var buffer bytes.Buffer
	row.RenderSize = row.Size + tabs*(Global.Tabsize-1) + 1
	for _, rv := range row.Data {
		if rv == '\t' {
			for i := 0; i < Global.Tabsize; i++ {
				buffer.WriteByte(' ')
			}
		} else {
			buffer.WriteRune(rv)
		}
	}
	row.Render = buffer.String()
	editorUpdateSyntax(row)
}

func updateLineIndexes() {
	for i, row := range Global.CurrentB.Rows {
		row.idx = i
	}
}

func editorAppendRow(line string) {
	Global.CurrentB.Rows = append(Global.CurrentB.Rows, &EditorRow{Global.CurrentB.NumRows,
		len(line), line, 0, "", []EmacsColor{}, false})
	editorUpdateRow(Global.CurrentB.Rows[Global.CurrentB.NumRows])
	Global.CurrentB.NumRows++
	Global.CurrentB.Dirty = true
}

func editorDelRow(at int) {
	if at < 0 || at > Global.CurrentB.NumRows {
		return
	}
	copy(Global.CurrentB.Rows[at:], Global.CurrentB.Rows[at+1:])
	Global.CurrentB.Rows[len(Global.CurrentB.Rows)-1] = nil
	Global.CurrentB.Rows = Global.CurrentB.Rows[:len(Global.CurrentB.Rows)-1]
	Global.CurrentB.NumRows--
	Global.CurrentB.Dirty = true
	updateLineIndexes()
}

func editorInsertRow(at int, line string) {
	if at < 0 || at > Global.CurrentB.NumRows {
		return
	}
	Global.CurrentB.Rows = append(Global.CurrentB.Rows, nil)
	copy(Global.CurrentB.Rows[at+1:], Global.CurrentB.Rows[at:])
	Global.CurrentB.Rows[at] = &EditorRow{at, len(line), line, 0, "", []EmacsColor{}, false}
	editorUpdateRow(Global.CurrentB.Rows[at])
	Global.CurrentB.NumRows++
	Global.CurrentB.Dirty = true
	updateLineIndexes()
}

func editorRowAppendStr(row *EditorRow, s string) {
	row.Data += s
	row.Size += len(s)
	editorUpdateRow(row)
	Global.CurrentB.Dirty = true
}

func editorRowInsertStr(row *EditorRow, at int, s string) {
	var buffer bytes.Buffer
	if row.Size > 0 && at < row.Size {
		buffer.WriteString(row.Data[0:at])
	} else if at == row.Size {
		buffer.WriteString(row.Data)
	}
	buffer.WriteString(s)
	if row.Size > 0 && at < row.Size {
		buffer.WriteString(row.Data[at:])
	}
	row.Data = buffer.String()
	row.Size = len(row.Data)
	editorUpdateRow(row)
	Global.CurrentB.Dirty = true
}

func editorRowDelChar(row *EditorRow, at int, rw int) {
	if at < 0 || row.Size < 0 || at >= row.Size {
		return
	}
	var buffer bytes.Buffer
	buffer.WriteString(row.Data[0:at])
	buffer.WriteString(row.Data[at+rw:])
	row.Data = buffer.String()
	row.Size = len(row.Data)
	editorUpdateRow(row)
	Global.CurrentB.Dirty = true
}

func editorInsertStr(s string) {
	Global.Input = "Insert " + s
	if Global.CurrentB.cy == Global.CurrentB.NumRows {
		editorAddUndo(true, Global.CurrentB.cx, Global.CurrentB.cx+len(s),
			-1, Global.CurrentB.cy, s)
		editorInsertRow(Global.CurrentB.cy, s)
		Global.CurrentB.cx += len(s)
		return
	}
	editorAddUndo(true, Global.CurrentB.cx, Global.CurrentB.cx+len(s),
		Global.CurrentB.cy, Global.CurrentB.cy, s)
	editorRowInsertStr(Global.CurrentB.Rows[Global.CurrentB.cy], Global.CurrentB.cx, s)
	Global.CurrentB.cx += len(s)
}

func editorDelChar() {
	if Global.CurrentB.cx == 0 && Global.CurrentB.cy == 0 {
		return
	}
	if Global.CurrentB.cy == Global.CurrentB.NumRows {
		return
	}
	row := Global.CurrentB.Rows[Global.CurrentB.cy]
	if Global.CurrentB.cx > 0 {
		_, rs := utf8.DecodeLastRuneInString(Global.CurrentB.Rows[Global.CurrentB.cy].Data[:Global.CurrentB.cx])
		editorAddUndo(false, Global.CurrentB.cx-rs, Global.CurrentB.cx, Global.CurrentB.cy,
			Global.CurrentB.cy, row.Data[Global.CurrentB.cx-rs:Global.CurrentB.cx])
		editorRowDelChar(row, Global.CurrentB.cx-rs, rs)
		Global.CurrentB.cx -= rs
	} else {
		editorAddUndo(false, Global.CurrentB.cx, Global.CurrentB.Rows[Global.CurrentB.cy-1].Size,
			Global.CurrentB.cy-1, Global.CurrentB.cy, row.Data)
		Global.CurrentB.cx = Global.CurrentB.Rows[Global.CurrentB.cy-1].Size
		editorRowAppendStr(Global.CurrentB.Rows[Global.CurrentB.cy-1], row.Data)
		editorDelRow(Global.CurrentB.cy)
		Global.CurrentB.cy--
	}
}

func getIndentation(s string) string {
	ret := ""
	for _, ru := range s {
		if ru == ' ' || ru == '\t' {
			ret += string(ru)
		} else {
			return ret
		}
	}
	return ret
}

func editorInsertNewline() {
	if Global.CurrentB.cy == Global.CurrentB.NumRows {
		return
	}
	row := Global.CurrentB.Rows[Global.CurrentB.cy]
	if Global.CurrentB.cx == 0 {
		editorAddUndo(true, Global.CurrentB.cx, Global.CurrentB.cx,
			Global.CurrentB.cy, Global.CurrentB.cy+1, row.Data)
		editorInsertRow(Global.CurrentB.cy, "")
		Global.CurrentB.cx = 0
	} else {
		editorAddUndo(true, Global.CurrentB.cx, Global.CurrentB.cx,
			Global.CurrentB.cy, Global.CurrentB.cy+1, row.Data[Global.CurrentB.cx:])
		pre := ""
		if Global.CurrentB.hasMode("indent-mode") {
			pre = getIndentation(row.Data[:Global.CurrentB.cx])
		}
		editorInsertRow(Global.CurrentB.cy+1, pre+row.Data[Global.CurrentB.cx:])
		row = Global.CurrentB.Rows[Global.CurrentB.cy]
		row.Size = Global.CurrentB.cx
		row.Data = row.Data[0:Global.CurrentB.cx]
		editorUpdateRow(row)
		Global.CurrentB.cx = len(pre)
	}
	Global.CurrentB.cy++
}

func EditorOpen(filename string) error {
	path, perr := homedir.Expand(filename)
	if perr != nil {
		return perr
	}
	Global.CurrentB.Filename = path
	editorSelectSyntaxHighlight(Global.CurrentB)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		editorAppendRow(scanner.Text())
	}
	Global.CurrentB.Dirty = false
	return nil
}

func EditorSave() {
	editorBufSave(Global.CurrentB)
}

func editorBufSave(buf *EditorBuffer) {
	fn := buf.Filename
	if fn == "" {
		fn = editorPrompt("Save as", nil)
		if fn == "" {
			Global.Input = "Save aborted"
			return
		} else {
			buf.Filename = fn
			editorSelectSyntaxHighlight(buf)
		}
	}
	f, err := os.Create(fn)
	if err != nil {
		Global.Input = err.Error()
		return
	}
	defer f.Close()
	l, b := 0, 0
	for _, row := range buf.Rows {
		f.WriteString(row.Data)
		f.WriteString("\n")
		b += row.Size + 1
		l++
	}
	Global.Input = fmt.Sprintf("Wrote %d lines (%d bytes) to %s", l, b, fn)
	buf.Dirty = false
}

// HACK: Go does not have static variables, so these have to go in global state.
var last_match int = -1
var direction int = 1
var saved_hl_line int
var saved_hl []EmacsColor = nil

func editorFindCallback(query string, key string) {
	Global.Input = query
	if saved_hl != nil {
		Global.CurrentB.Rows[saved_hl_line].Hl = saved_hl
		saved_hl = nil
	}
	if key == "C-s" {
		direction = 1
	} else if key == "C-r" {
		direction = -1
		//If we cancelled or finished...
	} else if key == "C-c" || key == "C-g" || key == "RET" {
		if key == "C-c" || key == "C-g" {
			Global.Input = "Cancelled search."
		}
		//...outta here!
		last_match = -1
		direction = 1
		return
	} else {
		last_match = -1
		direction = 1
	}

	if last_match == -1 {
		direction = 1
	}
	current := last_match
	for range Global.CurrentB.Rows {
		current += direction
		if current == -1 {
			current = Global.CurrentB.NumRows - 1
		} else if current == Global.CurrentB.NumRows {
			current = 0
		}
		row := Global.CurrentB.Rows[current]
		match := strings.Index(row.Render, query)
		if match > -1 {
			last_match = current
			Global.CurrentB.cy = current
			Global.CurrentB.cx = editorRowRxToCx(row, match)
			Global.CurrentB.rowoff = Global.CurrentB.NumRows
			saved_hl_line = current
			saved_hl = make([]EmacsColor, len(Global.CurrentB.Rows[current].Hl))
			copy(saved_hl, Global.CurrentB.Rows[current].Hl)
			for i := range query {
				Global.CurrentB.Rows[current].Hl[match+i] = HlSearch
			}
			break
		}
	}
}

func editorFind() {
	saved_cx := Global.CurrentB.cx
	saved_cy := Global.CurrentB.cy
	saved_co := Global.CurrentB.coloff
	saved_ro := Global.CurrentB.rowoff

	query := editorPrompt("Search", editorFindCallback)

	if query == "" {
		//Search cancelled, go back to where we were
		Global.CurrentB.cx = saved_cx
		Global.CurrentB.cy = saved_cy
		Global.CurrentB.coloff = saved_co
		Global.CurrentB.rowoff = saved_ro
	}
}

func getTabString() string {
	if Global.SoftTab {
		return strings.Repeat(" ", Global.Tabsize)
	} else {
		return "\t"
	}
}

func InitEditor() {
	buffer := &EditorBuffer{}
	Global = EditorState{false, "", buffer, []*EditorBuffer{buffer}, 4, "",
		false, []*EditorBuffer{buffer}, 0, "", false, make(map[string]bool)}
	Global.DefaultModes["terminal-title-mode"] = true
	Emacs = new(CommandList)
	Emacs.Parent = true
	funcnames = make(map[string]*CommandFunc)
}

func dumpCrashLog(e string) {
	f, err := os.Create("crash.log")
	if err != nil {
		Global.Input = err.Error()
		return
	}
	f.WriteString(e)
	f.Close()
}

func main() {
	InitEditor()
	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.BoolVar(&Global.NoSyntax, "s", false, "disable syntax highlighting")
	fs.Parse(os.Args[1:])
	args := fs.Args()
	env := NewLispInterp()
	if Global.Input == "" {
		Global.Input = "Welcome to Emacs!"
	}
	if len(args) > 0 {
		ferr := EditorOpen(args[0])
		if ferr != nil {
			Global.Input = ferr.Error()
		}
		if len(args) > 1 {
			for _, fn := range args[1:] {
				buffer := &EditorBuffer{}
				Global.Buffers = append(Global.Buffers, buffer)
				Global.CurrentB = buffer
				ferr = EditorOpen(fn)
				if ferr != nil {
					Global.Input = ferr.Error()
				}
			}
			Global.CurrentB = Global.Buffers[0]
		}
	}

	InitTerm()
	defer termbox.Close()

	for {
		editorRefreshScreen()
		if Global.quit {
			return
		} else {
			key := editorGetKey()
			//use f12 as panic button
			if key == "f12" {
				Global.quit = true
				continue
			}
			// Hack fixed (though we won't support any encoding save utf8)
			if utf8.RuneCountInString(key) == 1 {
				editorInsertStr(key)
				continue
			}
			Global.Input = ""
			com, comerr := Emacs.GetCommand(key)
			if comerr != nil {
				Global.Input = comerr.Error()
				continue
			} else if com != nil {
				com.Com(env)
			}
		}
	}
}
