package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/nsf/termbox-go"
	"os"
	"strings"
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
	Tabsize  int
	Prompt   string
}

var Global EditorState
var Emacs *CommandList

func printstring(s string, y int) {
	for i, ru := range s {
		termbox.SetCell(i, y, ru, termbox.ColorDefault, termbox.ColorDefault)
	}
}

func editorDrawRows(sy int) {
	for y := 0; y < sy; y++ {
		filerow := y + Global.CurrentB.rowoff
		if filerow >= Global.CurrentB.NumRows {
			termbox.SetCell(0, y, '~', termbox.ColorBlue, termbox.ColorDefault)
		} else {
			if Global.CurrentB.coloff < Global.CurrentB.Rows[filerow].RenderSize {
				printstring(Global.CurrentB.Rows[filerow].Render[Global.CurrentB.coloff:], y)
			}
		}
	}
}

func editorUpdateStatus() {
	if Global.CurrentB.Dirty {
		Global.Status = fmt.Sprintf("%s [Modified] - %d:%d", Global.CurrentB.Filename,
			Global.CurrentB.cy, Global.CurrentB.cx)
	} else {
		Global.Status = fmt.Sprintf("%s - %d:%d", Global.CurrentB.Filename,
			Global.CurrentB.cy, Global.CurrentB.cx)
	}
}

func GetScreenSize() (int, int) {
	x, y := termbox.Size()
	return x, y - 2
}

func editorDrawStatusLine(x, y int) {
	editorUpdateStatus()
	var i int
	var ru rune
	for i, ru = range Global.Status {
		termbox.SetCell(i, y-2, ru, termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
	}
	for ix := i + 1; ix < x; ix++ {
		termbox.SetCell(ix, y-2, ' ', termbox.ColorDefault|termbox.AttrReverse, termbox.ColorDefault)
	}
	for i, ru = range Global.Prompt + "-> " + Global.Input {
		termbox.SetCell(i, y-1, ru, termbox.ColorDefault, termbox.ColorDefault)
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
	editorSetPrompt(prompt)
	defer editorSetPrompt("")
	for {
		_, y := termbox.Size()
		Global.Input = buffer
		editorRefreshScreen()
		termbox.SetCursor(len(prompt)+3+buflen, y-1)
		termbox.Flush()
		key := editorGetKey()
		switch key {
		case "C-c":
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
				buffer = buffer[0 : buflen-1]
				buflen--
			}
		default:
			if len(key) == 1 {
				buffer += key
				buflen++
			}
		}
		if callback != nil {
			callback(buffer, key)
		}
	}
}

func MoveCursor(x, y int) {
	// Regular cursor movement - most cases
	realline := Global.CurrentB.cy < Global.CurrentB.NumRows && Global.CurrentB.NumRows != 0
	nx, ny := Global.CurrentB.cx+x, Global.CurrentB.cy+y
	if nx >= 0 && ny < Global.CurrentB.NumRows && realline && nx <= Global.CurrentB.Rows[Global.CurrentB.cy].Size {
		Global.CurrentB.cx = nx
	}
	if ny >= 0 && ny <= Global.CurrentB.NumRows {
		Global.CurrentB.cy = ny
	}

	// Edge cases
	realline = Global.CurrentB.cy < Global.CurrentB.NumRows && Global.CurrentB.NumRows != 0
	if nx < 0 && Global.CurrentB.cy > 0 {
		// Left at the beginning of a line
		Global.CurrentB.cy--
		MoveCursorToEol()
	} else if realline && y == 0 && nx > Global.CurrentB.Rows[Global.CurrentB.cy].Size {
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
	if Global.CurrentB.cy <= Global.CurrentB.NumRows {
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
		}
		rx++
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
	for j, rv := range row.Data {
		if rv == '\t' {
			for i := 0; i < Global.Tabsize; i++ {
				buffer.WriteByte(' ')
			}
		} else {
			buffer.WriteByte(row.Data[j])
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

func editorRowDelChar(row *EditorRow, at int) {
	if at < 0 || row.Size < 0 || at >= row.Size {
		return
	}
	var buffer bytes.Buffer
	buffer.WriteString(row.Data[0:at])
	buffer.WriteString(row.Data[at+1:])
	row.Data = buffer.String()
	row.Size = len(row.Data)
	editorUpdateRow(row)
	Global.CurrentB.Dirty = true
}

func editorInsertStr(s string) {
	Global.Input = "Insert " + s
	if Global.CurrentB.cy == Global.CurrentB.NumRows {
		editorInsertRow(Global.CurrentB.cy, s)
		Global.CurrentB.cx++
		return
	}
	editorRowInsertStr(Global.CurrentB.Rows[Global.CurrentB.cy], Global.CurrentB.cx, s)
	Global.CurrentB.cx++
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
		editorRowDelChar(row, Global.CurrentB.cx-1)
		Global.CurrentB.cx--
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

func editorFindCallback(query string, key string) {
	//If it's an unprintable character, and we're not just ammending the string...
	if len(key) > 1 && key != "DEL" {
		//...outta here!
		return
	}
	for i, row := range Global.CurrentB.Rows {
		match := strings.Index(row.Render, query)
		if match > -1 {
			Global.CurrentB.cy = i
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
	Global = EditorState{false, "", "", buffer, 4, ""}
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
			//HACK: Currently I'm a little unsure of how to do this.
			//We'll just guess that it's a printable character by its len
			if len(key) == 1 {
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
