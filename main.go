package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

type CommandList struct {
	Parent   bool
	Command  string
	Children map[string]*CommandList
}

func (c *CommandList) PutCommand(key string, command string) {
	if c.Children == nil {
		c.Children = make(map[string]*CommandList)
	}
	keys := strings.Split(key, " ")
	if c.Children[keys[0]] == nil {
		c.Children[keys[0]] = &CommandList{false, "", nil}
	}
	if len(keys) > 1 {
		c.Children[keys[0]].Parent = true
		c.Children[keys[0]].PutCommand(strings.Join(keys[1:], " "), command)
	} else {
		c.Children[keys[0]].Command = command
	}
}

func (c *CommandList) GetCommand(key string) (string, error) {
	Global.Input += key + " "
	editorRefreshScreen()
	child := c.Children[key]
	if child == nil {
		return "", errors.New("Bad command: " + Global.Input)
	}
	if child.Parent {
		nextkey := editorGetKey()
		s, e := child.GetCommand(nextkey)
		return s, e
	} else {
		return child.Command, nil
	}
}

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
}

type EditorState struct {
	quit     bool
	Status   string
	Input    string
	CurrentB *EditorBuffer
	Buffers  []*EditorBuffer
	Tabsize  int
	Prompt   string
	NoSyntax bool
}

type EditorUndo struct {
	ins    bool
	startl int
	endl   int
	startc int
	endc   int
	str    string
	prev   *EditorUndo
}

var Global EditorState
var Emacs *CommandList

func printstring(s string, x, y int) {
	i := 0
	for _, ru := range s {
		termbox.SetCell(x+i, y, ru, termbox.ColorDefault, termbox.ColorDefault)
		i += runewidth.RuneWidth(ru)
	}
}

func editorSwitchBuffer() {
	choices := []string{}
	def := 0
	for i, buf := range Global.Buffers {
		if buf == Global.CurrentB {
			def = i
		}
		d := ""
		if buf.Dirty {
			d = "[M] "
		}
		if buf.Filename == "" {
			choices = append(choices, d+"*unnamed buffer*")
		} else {
			choices = append(choices, d+buf.Filename)
		}
	}
	in := editorChoiceIndex("Switch buffer", choices, def)
	Global.CurrentB = Global.Buffers[in]
}

func editorChoiceIndex(title string, choices []string, def int) int {
	selection := def
	// Will need these in a smarter version of this function
	//offset := 0
	//sx, sy := termbox.Size()
	for {
		termbox.HideCursor()
		termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
		printstring(title, 0, 0)
		for i, s := range choices {
			printstring(s, 3, i+1)
		}
		printstring(">", 1, selection+1)
		termbox.Flush()
		key := editorGetKey()
		switch key {
		case "C-c":
			fallthrough
		case "C-g":
			return def
		case "UP":
			if selection > 0 {
				selection--
			}
		case "DOWN":
			if selection < len(choices)-1 {
				selection++
			}
		case "RET":
			return selection
		}
	}
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
		if unicode.IsControl(ru) || !utf8.ValidRune(ru) {
			sym := '?'
			if ru <= rune(26) {
				sym = '@' + ru
			}
			termbox.SetCell(x+i, y, sym, termbox.AttrReverse, termbox.ColorDefault)
		} else {
			col := editorSyntaxToColor(hl[in])
			termbox.SetCell(x+i, y, ru, col, termbox.ColorDefault)
		}
		i += runewidth.RuneWidth(ru)
	}
}

func editorDrawRows(starty, sy int, buf *EditorBuffer) {
	for y := starty; y < sy; y++ {
		filerow := y + buf.rowoff
		if filerow >= buf.NumRows {
			if buf.coloff == 0 {
				termbox.SetCell(0, y, '~', termbox.ColorBlue, termbox.ColorDefault)
			}
		} else {
			if buf.coloff < buf.Rows[filerow].RenderSize {
				r, off := trimString(buf.Rows[filerow].Render, buf.coloff)
				hlprint(r, buf.Rows[filerow].Hl[off:], 0, y)
			}
		}
	}
}

func editorUpdateStatus() {
	fn := Global.CurrentB.Filename
	if fn == "" {
		fn = "*unnamed file*"
	}
	syn := "no ft"
	if Global.CurrentB.Syntax != nil {
		syn = Global.CurrentB.Syntax.filetype
	}
	if Global.CurrentB.Dirty {
		Global.Status = fmt.Sprintf("%s [Modified] - (%s) %d:%d", fn, syn,
			Global.CurrentB.cy, Global.CurrentB.cx)
	} else {
		Global.Status = fmt.Sprintf("%s - (%s) %d:%d", fn, syn,
			Global.CurrentB.cy, Global.CurrentB.cx)
	}
}

func GetScreenSize() (int, int) {
	x, y := termbox.Size()
	return x, y - 2
}

func editorDrawStatusLine(x, y int) {
	editorUpdateStatus()
	var ru rune
	rx := 0
	for _, ru = range Global.Status {
		termbox.SetCell(rx, y-2, ru, termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
		rx += runewidth.RuneWidth(ru)
	}
	for ix := rx; ix < x; ix++ {
		termbox.SetCell(ix, y-2, ' ', termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
	}
	rx = 0
	for _, ru = range Global.Prompt + "-> " + Global.Input {
		termbox.SetCell(rx, y-1, ru, termbox.ColorDefault, termbox.ColorDefault)
		rx += runewidth.RuneWidth(ru)
	}
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
	x, y := termbox.Size()
	editorScroll(x, y-2)
	termbox.Clear(termbox.ColorDefault, termbox.ColorDefault)
	termbox.SetCursor(Global.CurrentB.rx-Global.CurrentB.coloff, Global.CurrentB.cy-Global.CurrentB.rowoff)
	editorDrawRows(0, y-2, Global.CurrentB)
	editorDrawStatusLine(x, y)
	termbox.Flush()
}

func editorSetPrompt(prompt string) {
	Global.Prompt = prompt
}

func editorGetKey() string {
	for {
		ev := termbox.PollEvent()
		if ev.Type == termbox.EventResize {
			editorRefreshScreen()
		} else if ev.Type == termbox.EventKey {
			return ParseTermboxEvent(ev)
		}
	}
}

func editorPrompt(prompt string, callback func(string, string)) string {
	buffer := ""
	buflen := 0
	cursor := 0
	editorSetPrompt(prompt)
	defer editorSetPrompt("")
	for {
		_, y := termbox.Size()
		Global.Input = buffer
		editorRefreshScreen()
		termbox.SetCursor(utf8.RuneCountInString(prompt)+3+cursor, y-1)
		termbox.Flush()
		key := editorGetKey()
		switch key {
		case "C-c":
			fallthrough
		case "C-g":
			if callback != nil {
				callback(buffer, key)
			}
			return ""
		case "RET":
			if callback != nil {
				callback(buffer, key)
			}
			return buffer
		case "DEL":
			if buflen > 0 {
				r, rs := utf8.DecodeLastRuneInString(buffer)
				buffer = buffer[0 : buflen-rs]
				buflen -= rs
				cursor -= runewidth.RuneWidth(r)
			}
		default:
			if utf8.RuneCountInString(key) == 1 {
				r, _ := utf8.DecodeLastRuneInString(buffer)
				buffer += key
				buflen += len(key)
				cursor += runewidth.RuneWidth(r)
			}
		}
		if callback != nil {
			callback(buffer, key)
		}
	}
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
	Global.quit = true
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
			rx += runewidth.RuneWidth(rv)
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

func editorAddUndo(ins bool, startc, endc, startl, endl int, str string) {
	old := Global.CurrentB.Undo
	app := false
	if old != nil {
		app = old.startl == startl && old.endl == endl && old.ins == ins
		if app {
			if ins {
				app = old.endc == startc
			} else {
				app = old.startc == endc
			}
		}
	}
	if app {
		//append to group things together, ala gnu
		if ins {
			old.str += str
			old.endc = endc
		} else {
			old.str = str + old.str
			old.startc = startc
		}
	} else {
		ret := new(EditorUndo)
		ret.endl = endl
		ret.startl = startl
		ret.endc = endc
		ret.startc = startc
		ret.str = str
		ret.ins = ins

		if old == nil {
			ret.prev = nil
		} else {
			ret.prev = old
		}
		Global.CurrentB.Undo = ret
	}
}

func editorDoUndo(tree *EditorUndo, redo bool) bool {
	if tree == nil {
		return false
	}
	if tree.ins {
		// Insertion
		if tree.startl == tree.endl {
			// Basic string insertion
			editorRowDelChar(Global.CurrentB.Rows[tree.startl],
				tree.startc, len(tree.str))
			Global.CurrentB.cx = tree.startc
			Global.CurrentB.cy = tree.startl
			return true
		} else if tree.startl == -1 {
			// inserting a string on the last line
			editorDelRow(Global.CurrentB.NumRows - 1)
			Global.CurrentB.cx = tree.startc
			Global.CurrentB.cy = tree.endl
			return true
		} else {
			// inserting a line
			Global.CurrentB.cx = tree.startc
			Global.CurrentB.cy = tree.startl
			editorRowAppendStr(Global.CurrentB.Rows[tree.startl], tree.str)
			editorDelRow(tree.endl)
			return true
		}
	} else {
		// Deletion
		if tree.startl == tree.endl {
			// Character or word deletion
			editorRowInsertStr(Global.CurrentB.Rows[tree.startl],
				tree.startc, tree.str)
			Global.CurrentB.cx = tree.endc
			Global.CurrentB.cy = tree.startl
			return true
		} else {
			// deleting a line
			editorInsertRow(tree.startl, Global.CurrentB.Rows[tree.startl].Data[:tree.endc])
			row := Global.CurrentB.Rows[tree.endl]
			row.Data = tree.str
			row.Size = len(row.Data)
			Global.CurrentB.Rows[tree.startl].Size = len(Global.CurrentB.Rows[tree.startl].Data)
			editorUpdateRow(row)
			editorUpdateRow(Global.CurrentB.Rows[tree.startl])
			return true
		}
	}
}

func editorUndoAction() {
	succ := editorDoUndo(Global.CurrentB.Undo, false)
	if succ {
		Global.CurrentB.Undo = Global.CurrentB.Undo.prev
	} else {
		Global.Input = "No further undo information."
	}
	if Global.CurrentB.Undo == nil {
		Global.CurrentB.Dirty = false
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

func editorInsertNewline() {
	row := Global.CurrentB.Rows[Global.CurrentB.cy]
	if Global.CurrentB.cx == 0 {
		editorAddUndo(true, Global.CurrentB.cx, Global.CurrentB.cx,
			Global.CurrentB.cy, Global.CurrentB.cy+1, row.Data)
		editorInsertRow(Global.CurrentB.cy, "")
	} else {
		editorAddUndo(true, Global.CurrentB.cx, Global.CurrentB.cx,
			Global.CurrentB.cy, Global.CurrentB.cy+1, row.Data[Global.CurrentB.cx:])
		editorInsertRow(Global.CurrentB.cy+1, row.Data[Global.CurrentB.cx:])
		row = Global.CurrentB.Rows[Global.CurrentB.cy]
		row.Size = Global.CurrentB.cx
		row.Data = row.Data[0:Global.CurrentB.cx]
		editorUpdateRow(row)
	}
	Global.CurrentB.cy++
	Global.CurrentB.cx = 0
}

func editorFindFile() {
	fn := editorPrompt("Find File", nil)
	if fn == "" {
		return
	}
	buffer := &EditorBuffer{}
	Global.CurrentB = buffer
	Global.Buffers = append(Global.Buffers, buffer)
	EditorOpen(fn)
}

func EditorOpen(filename string) error {
	Global.CurrentB.Filename = filename
	editorSelectSyntaxHighlight(Global.CurrentB)
	f, err := os.Open(filename)
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
	fn := Global.CurrentB.Filename
	if fn == "" {
		fn = editorPrompt("Save as", nil)
		if fn == "" {
			Global.Input = "Save aborted"
			return
		} else {
			Global.CurrentB.Filename = fn
			editorSelectSyntaxHighlight(Global.CurrentB)
		}
	}
	f, err := os.Create(fn)
	if err != nil {
		Global.Input = err.Error()
		return
	}
	defer f.Close()
	l, b := 0, 0
	for _, row := range Global.CurrentB.Rows {
		f.WriteString(row.Data)
		f.WriteString("\n")
		b += row.Size + 1
		l++
	}
	Global.Input = fmt.Sprintf("Wrote %d lines (%d bytes) to %s", l, b, fn)
	Global.CurrentB.Dirty = false
}

// HACK: Go does not have static variables, so these have to go in global state.
var last_match int = -1
var direction int = 1
var saved_hl_line int
var saved_hl []EmacsColor = nil

func editorFindCallback(query string, key string) {
	if saved_hl != nil {
		Global.CurrentB.Rows[saved_hl_line].Hl = saved_hl
		saved_hl = nil
	}
	if key == "C-s" {
		direction = 1
	} else if key == "C-r" {
		direction = -1
		//If it's an unprintable character, and we're not just ammending the string...
	} else if utf8.RuneCountInString(key) > 1 && key != "DEL" {
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

func InitEditor() {
	buffer := &EditorBuffer{}
	Global = EditorState{false, "", "", buffer, []*EditorBuffer{buffer}, 4, "", false}
	Emacs = new(CommandList)
	Emacs.Parent = true
}

func InitTerm() {
	err := termbox.Init()
	if err != nil {
		panic(err)
	}
	termbox.SetInputMode(termbox.InputAlt)
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
			}
			comerr = env.LoadString(com)
			if comerr != nil {
				Global.Input = comerr.Error()
				continue
			}
			_, comerr = env.Run()
			if comerr != nil {
				Global.Input = comerr.Error()
				dumpCrashLog(comerr.Error())
				env = NewLispInterp()
				continue
			}
		}
	}
}
