package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
	"os"
	"strings"
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
	Size       int
	Data       string
	RenderSize int
	Render     string
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
}

type EditorState struct {
	quit     bool
	Status   string
	Input    string
	CurrentB *EditorBuffer
	Buffers  []*EditorBuffer
	Tabsize  int
	Prompt   string
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

func trimString(s string, coloff int) string {
	if coloff == 0 {
		return s
	}
	sr := []rune(s)
	if coloff < len(sr) {
		return string(sr[coloff:])
	} else {
		return ""
	}
}

func editorDrawRows(sy int) {
	for y := 0; y < sy; y++ {
		filerow := y + Global.CurrentB.rowoff
		if filerow >= Global.CurrentB.NumRows {
			termbox.SetCell(0, y, '~', termbox.ColorBlue, termbox.ColorDefault)
		} else {
			if Global.CurrentB.coloff < Global.CurrentB.Rows[filerow].RenderSize {
				printstring(trimString(Global.CurrentB.Rows[filerow].Render, Global.CurrentB.coloff), 0, y)
			}
		}
	}
}

func editorUpdateStatus() {
	fn := Global.CurrentB.Filename
	if fn == "" {
		fn = "*unnamed file*"
	}
	if Global.CurrentB.Dirty {
		Global.Status = fmt.Sprintf("%s [Modified] - %d:%d", fn,
			Global.CurrentB.cy, Global.CurrentB.cx)
	} else {
		Global.Status = fmt.Sprintf("%s - %d:%d", fn,
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
	editorDrawRows(y - 2)
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
}

func editorAppendRow(line string) {
	Global.CurrentB.Rows = append(Global.CurrentB.Rows, &EditorRow{len(line), line, 0, ""})
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
}

func editorInsertRow(at int, line string) {
	if at < 0 || at > Global.CurrentB.NumRows {
		return
	}
	Global.CurrentB.Rows = append(Global.CurrentB.Rows, nil)
	copy(Global.CurrentB.Rows[at+1:], Global.CurrentB.Rows[at:])
	Global.CurrentB.Rows[at] = &EditorRow{len(line), line, 0, ""}
	editorUpdateRow(Global.CurrentB.Rows[at])
	Global.CurrentB.NumRows++
	Global.CurrentB.Dirty = true
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
		editorInsertRow(Global.CurrentB.cy, s)
		Global.CurrentB.cx += len(s)
		return
	}
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
		_, rs := utf8.DecodeRuneInString(Global.CurrentB.Rows[Global.CurrentB.cy].Data[:Global.CurrentB.cx])
		editorRowDelChar(row, Global.CurrentB.cx-rs, rs)
		Global.CurrentB.cx -= rs
	} else {
		Global.CurrentB.cx = Global.CurrentB.Rows[Global.CurrentB.cy-1].Size
		editorRowAppendStr(Global.CurrentB.Rows[Global.CurrentB.cy-1], row.Data)
		editorDelRow(Global.CurrentB.cy)
		Global.CurrentB.cy--
	}
}

func editorInsertNewline() {
	if Global.CurrentB.cx == 0 {
		editorInsertRow(Global.CurrentB.cy, "")
	} else {
		row := Global.CurrentB.Rows[Global.CurrentB.cy]
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

func editorFindCallback(query string, key string) {
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
	Global = EditorState{false, "", "", buffer, []*EditorBuffer{buffer}, 4, ""}
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
	env := NewLispInterp()
	if Global.Input == "" {
		Global.Input = "Welcome to Emacs!"
	}
	if len(os.Args) > 1 {
		ferr := EditorOpen(os.Args[1])
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
