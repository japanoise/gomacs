package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/nsf/termbox-go"
	"github.com/zhemao/glisp/interpreter"
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
	Filename   string
	Rendername string
	Dirty      bool
	cx         int
	cy         int
	rx         int
	rowoff     int
	coloff     int
	NumRows    int
	Rows       []*EditorRow
	Syntax     *EditorSyntax
	Undo       *EditorUndo
	Redo       *EditorUndo
	MarkX      int
	MarkY      int
	Modes      ModeList
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

func editorSetPrompt(prompt string) {
	Global.Prompt = prompt
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
	Global.CurrentB.Rendername = path
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

func RunCommandForKey(key string, env *glisp.Glisp) {
	//use f12 as panic button
	if key == "f12" {
		Global.quit = true
		return
	}
	// Hack fixed (though we won't support any encoding save utf8)
	if !Global.CurrentB.hasMode("no-self-insert-mode") && utf8.RuneCountInString(key) == 1 {
		editorInsertStr(key)
		return
	}
	Global.Input = ""
	com, comerr := Emacs.GetCommand(key)
	if comerr != nil {
		Global.Input = comerr.Error()
		return
	} else if com != nil {
		if macrorec {
			macro = append(macro, com)
		}
		com.Com(env)
	}
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
			RunCommandForKey(key, env)
		}
	}
}
