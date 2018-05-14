package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mitchellh/go-homedir"
	"github.com/nsf/termbox-go"
	"github.com/zyedidia/highlight"
)

const TIMEOUT int64 = 10000000

type EditorRow struct {
	idx        int
	Size       int
	Data       string
	RenderSize int
	Render     string
	HlState    highlight.State
	HlMatches  highlight.LineMatch
}

type EditorBuffer struct {
	Filename     string
	Rendername   string
	Dirty        bool
	cx           int
	cy           int
	rx           int
	rowoff       int
	coloff       int
	NumRows      int
	Rows         []*EditorRow
	Undo         *EditorUndo
	Redo         *EditorUndo
	SaveUndo     *EditorUndo // The undo at which we can undirty the buffer
	MarkX        int
	MarkY        int
	Modes        ModeList
	Highlighter  *highlight.Highlighter
	MajorMode    string
	prefcx       int
	rehlfrom     int
	needshl      bool
	regionActive bool
	region       *Region
}

type EditorState struct {
	quit                    bool
	Input                   string
	CurrentB                *EditorBuffer
	Buffers                 []*EditorBuffer
	Tabsize                 int
	Prompt                  string
	NoSyntax                bool
	Windows                 []*EditorBuffer
	CurrentBHeight          int
	Clipboard               string
	SoftTab                 bool
	DefaultModes            map[string]bool
	messages                []string
	debug                   bool
	Universal               int
	SetUniversal            bool
	MajorHooks              HookList
	LastCommand             *CommandFunc
	LastCommandSetUniversal bool
	LastCommandUniversal    int
	Registers               *RegisterList
	Fillcolumn              int
	MajorBindings           map[string]*CommandList
	MouseX                  int
	MouseY                  int
}

var Global EditorState
var Emacs *CommandList

func editorSetPrompt(prompt string) {
	Global.Prompt = prompt
}

func saveSomeBuffers() bool {
	nodirty := true
	for _, buf := range Global.Buffers {
		if buf.Dirty {
			ds, cancel := editorYesNoPrompt(fmt.Sprintf("%s has unsaved changes; save them?", buf.getRenderName()), true)
			if ds && cancel == nil {
				editorBufSave(buf)
			} else if cancel != nil {
				return false
			}
		}
		nodirty = nodirty && !buf.Dirty
	}
	return nodirty
}

func doSaveSomeBuffers() {
	nodirty := saveSomeBuffers()
	if nodirty {
		Global.Input = "All buffers saved."
	} else {
		Global.Input = "Some buffers remain unsaved."
	}
}

func saveBuffersKillEmacs() {
	nodirty := saveSomeBuffers()
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

func rowUpdateRender(row *EditorRow) {
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

func editorReHighlightRow(row *EditorRow, buf *EditorBuffer) {
	if buf.Highlighter != nil {
		if buf.rehlfrom >= 0 && buf.rehlfrom != row.idx {
			if row.idx < buf.rehlfrom {
				buf.rehlfrom = row.idx
			}
			buf.needshl = true
		} else {
			buf.rehlfrom = row.idx
		}
	}
}

func (buf *EditorBuffer) updateHighlighting() {
	if buf.Highlighter != nil && buf.rehlfrom >= 0 && buf.NumRows != 0 {
		row := buf.Rows[buf.rehlfrom]
		curstate := buf.State(row.idx)
		buf.Highlighter.ReHighlightStates(buf, row.idx)
		if curstate != buf.State(row.idx) || buf.needshl {
			// If the EOL state changed, the buffer needs rehighlighting
			// as this was probably multiline comment or string.
			buf.Highlighter.HighlightMatches(buf, row.idx, buf.NumRows)
		} else {
			// Probably only this line changed.
			buf.Highlighter.ReHighlightLine(buf, row.idx)
		}
		buf.rehlfrom = -1
		buf.needshl = false
	}
}

func editorUpdateRow(row *EditorRow, buf *EditorBuffer) {
	rowUpdateRender(row)
	editorReHighlightRow(row, buf)
}

func updateLineIndexes() {
	for i, row := range Global.CurrentB.Rows {
		row.idx = i
	}
}

func editorAppendRow(line string) {
	Global.CurrentB.Rows = append(Global.CurrentB.Rows, &EditorRow{Global.CurrentB.NumRows,
		len(line), line, 0, "", nil, nil})
	editorUpdateRow(Global.CurrentB.Rows[Global.CurrentB.NumRows], Global.CurrentB)
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
	Global.CurrentB.Rows[at] = &EditorRow{at, len(line), line, 0, "", nil, nil}
	editorUpdateRow(Global.CurrentB.Rows[at], Global.CurrentB)
	Global.CurrentB.NumRows++
	Global.CurrentB.Dirty = true
	updateLineIndexes()
}

func editorRowAppendStr(row *EditorRow, buf *EditorBuffer, s string) {
	editorMutateRow(row, buf, row.Data+s)
}

func editorRowInsertStr(row *EditorRow, buf *EditorBuffer, at int, s string) {
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
	editorMutateRow(row, buf, buffer.String())
}

func editorMutateRow(row *EditorRow, buf *EditorBuffer, s string) {
	row.Data = s
	row.Size = len(row.Data)
	editorUpdateRow(row, buf)
	Global.CurrentB.Dirty = true
}

func editorRowDelChar(row *EditorRow, buf *EditorBuffer, at int, rw int) {
	if at < 0 || row.Size < 0 || at >= row.Size {
		return
	}
	var buffer bytes.Buffer
	buffer.WriteString(row.Data[0:at])
	buffer.WriteString(row.Data[at+rw:])
	row.Data = buffer.String()
	row.Size = len(row.Data)
	editorUpdateRow(row, buf)
	Global.CurrentB.Dirty = true
}

func editorInsertStr(s string) {
	if Global.SetUniversal && Global.Universal >= 0 {
		os := s
		s = ""
		for i := 0; i < Global.Universal; i++ {
			s += os
		}
	}
	Global.Input = ""
	if Global.CurrentB.cy == Global.CurrentB.NumRows {
		editorAddInsertUndo(Global.CurrentB.cx, Global.CurrentB.cy, s)
		editorInsertRow(Global.CurrentB.cy, s)
		Global.CurrentB.cx += len(s)
		return
	}
	editorAddInsertUndo(Global.CurrentB.cx, Global.CurrentB.cy, s)
	editorRowInsertStr(Global.CurrentB.Rows[Global.CurrentB.cy], Global.CurrentB, Global.CurrentB.cx, s)
	Global.CurrentB.cx += len(s)
}

func editorDelChar() {
	times := 1
	if Global.SetUniversal {
		if Global.Universal >= 0 {
			times = Global.Universal
		} else {
			Global.Universal *= -1
			editorDelForwardChar()
			return
		}
	}
	for i := 0; i < times; i++ {
		if Global.CurrentB.cx == 0 && Global.CurrentB.cy == 0 {
			Global.Input = "Beginning of buffer"
			return
		}
		if Global.CurrentB.cy == Global.CurrentB.NumRows {
			return
		}
		row := Global.CurrentB.Rows[Global.CurrentB.cy]
		if Global.CurrentB.cx > 0 {
			_, rs := utf8.DecodeLastRuneInString(Global.CurrentB.Rows[Global.CurrentB.cy].Data[:Global.CurrentB.cx])
			editorAddDeleteUndo(Global.CurrentB.cx-rs, Global.CurrentB.cx, Global.CurrentB.cy,
				Global.CurrentB.cy, row.Data[Global.CurrentB.cx-rs:Global.CurrentB.cx])
			editorRowDelChar(row, Global.CurrentB, Global.CurrentB.cx-rs, rs)
			Global.CurrentB.cx -= rs
		} else {
			editorAddDeleteUndo(Global.CurrentB.cx, Global.CurrentB.Rows[Global.CurrentB.cy-1].Size,
				Global.CurrentB.cy-1, Global.CurrentB.cy, row.Data)
			Global.CurrentB.cx = Global.CurrentB.Rows[Global.CurrentB.cy-1].Size
			editorRowAppendStr(Global.CurrentB.Rows[Global.CurrentB.cy-1], Global.CurrentB, row.Data)
			editorDelRow(Global.CurrentB.cy)
			Global.CurrentB.cy--
		}
	}
}

func editorDelForwardChar() {
	cx, cy := Global.CurrentB.cx, Global.CurrentB.cy
	if cy == Global.CurrentB.NumRows-1 && cx == Global.CurrentB.Rows[cy].Size {
		Global.Input = "End of buffer"
		return
	}
	times := 1
	if Global.SetUniversal {
		if Global.Universal < 0 {
			Global.Universal *= -1
			editorDelChar()
			return
		} else {
			times = Global.Universal
		}
	}
	for i := 0; i < times; i++ {
		Global.CurrentB.MoveCursorRight()
	}
	for i := 0; i < times; i++ {
		editorDelChar()
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

func editorInsertNewline(indent bool) {
	if Global.CurrentB.cy == Global.CurrentB.NumRows {
		defer func() { Global.CurrentB.cy++; Global.CurrentB.cx = 0 }()
		if Global.CurrentB.NumRows == 0 {
			editorAppendRow("")
			return
		} else {
			Global.CurrentB.cy--
			Global.CurrentB.cx = Global.CurrentB.Rows[Global.CurrentB.cy].Size
		}
	}
	row := Global.CurrentB.Rows[Global.CurrentB.cy]
	if Global.CurrentB.cx == 0 {
		editorAddInsertUndo(Global.CurrentB.cx, Global.CurrentB.cy, "\n")
		editorInsertRow(Global.CurrentB.cy, "")
		Global.CurrentB.cx = 0
	} else {
		pre := ""
		if indent {
			pre = getIndentation(row.Data[:Global.CurrentB.cx])
		}
		data := pre + row.Data[Global.CurrentB.cx:]
		editorAddInsertUndo(Global.CurrentB.cx, Global.CurrentB.cy, "\n"+pre)
		editorInsertRow(Global.CurrentB.cy+1, data)
		row = Global.CurrentB.Rows[Global.CurrentB.cy]
		row.Size = Global.CurrentB.cx
		row.Data = row.Data[0:Global.CurrentB.cx]
		editorUpdateRow(row, Global.CurrentB)
		Global.CurrentB.cx = len(pre)
	}
	Global.CurrentB.cy++
}

func AbsPath(filename string) (string, error) {
	hdpath, perr := homedir.Expand(filename)
	if perr != nil {
		return filename, perr
	}
	if len(hdpath) > 0 && hdpath[0] == '/' {
		return hdpath, nil
	}
	cwd, cerr := os.Getwd()
	if cerr != nil {
		return filename, cerr
	}
	return path.Join(cwd, filename), nil
}

func EditorOpen(filename string) error {
	fpath, perr := AbsPath(filename)
	if perr != nil {
		return perr
	}
	Global.CurrentB.Filename = fpath
	Global.CurrentB.UpdateRenderName()
	f, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		editorAppendRow(scanner.Text())
	}
	Global.CurrentB.Dirty = false
	editorSelectSyntaxHighlight(Global.CurrentB)
	return nil
}

func (buf *EditorBuffer) UpdateRenderName() {
	buf.Rendername = filepath.Base(buf.Filename)
}

func tabCompleteFilename(fn string) []string {
	if fn == "" {
		return nil
	}
	fpath, perr := AbsPath(fn)
	if perr != nil {
		return nil
	}
	fdir := filepath.Dir(fpath)
	files, err := ioutil.ReadDir(fdir)
	if err != nil || len(files) <= 0 {
		return nil
	}
	ret := make([]string, 0)
	for _, file := range files {
		fileFullPath := fdir + string(filepath.Separator) + file.Name()
		if file.IsDir() {
			fileFullPath += string(filepath.Separator)
		}
		if strings.HasPrefix(fileFullPath, fpath) {
			ret = append(ret, fileFullPath)
		}
	}
	return ret
}

func EditorSave() {
	editorBufSave(Global.CurrentB)
	ExecSaveHooksForMode(Global.CurrentB.MajorMode)
}

func editorBufSave(buf *EditorBuffer) {
	fn := buf.Filename
	if fn == "" {
		fn = editorPrompt("Save as", nil)
		if fn == "" {
			Global.Input = "Save aborted"
			return
		} else {
			fpath, perr := AbsPath(fn)
			if perr != nil {
				Global.Input = perr.Error()
				AddErrorMessage(Global.Input)
				return
			}
			buf.Filename = fpath
			fn = buf.Filename
			buf.Rendername = filepath.Base(fpath)
		}
	}
	editorSelectSyntaxHighlight(buf)
	f, err := os.Create(fn)
	if err != nil {
		Global.Input = err.Error()
		AddErrorMessage(err.Error())
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
	AddErrorMessage(Global.Input)
	buf.Dirty = false
	buf.SaveUndo = buf.Undo
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
	buffer.MajorMode = "Unknown"
	Global = EditorState{false, "", buffer, []*EditorBuffer{buffer}, 4, "",
		false, []*EditorBuffer{buffer}, 0, "", false, make(map[string]bool),
		[]string{}, false, 0, false, loadDefaultHooks(), nil, false, 0,
		NewRegisterList(), 80, make(map[string]*CommandList), 0, 0}
	Global.DefaultModes["terminal-title-mode"] = true
	Emacs = new(CommandList)
	Emacs.Parent = true
	funcnames = make(map[string]*CommandFunc)
}

func dumpCrashLog(e string) {
	AddErrorMessage(e)
	if Global.debug {
		f, err := os.Create("crash.log")
		if err != nil {
			Global.Input = err.Error()
			return
		}
		f.WriteString(e)
		f.Close()
	}
}

func RunCommandForKey(key string) {
	//use f12 as panic button
	if key == "f12" {
		Global.quit = true
		return
	}
	// Hack fixed (though we won't support any encoding save utf8)
	if !Global.CurrentB.hasMode("no-self-insert-mode") && utf8.RuneCountInString(key) == 1 {
		com := &CommandFunc{
			key,
			func() {
				editorInsertStr(key)
			},
			false,
		}
		com.Run()
		return
	}
	if Global.MajorBindings[Global.CurrentB.MajorMode] != nil {
		Global.Input = ""
		com, comerr := Global.MajorBindings[Global.CurrentB.MajorMode].GetCommand(key)
		if com != nil && comerr == nil {
			com.Run()
			return
		}
	}
	Global.Input = ""
	com, comerr := Emacs.GetCommand(key)
	if comerr != nil {
		Global.Input = comerr.Error()
		return
	} else if com != nil {
		com.Run()
	}
}

func AddErrorMessage(msg string) {
	Global.messages = append(Global.messages, msg)
}

func SetUniversalArgument() {
	arg := ""
	for {
		key, drhl := editorGetKey()
		if (arg == "" && key == "-") || ('0' <= key[0] && key[0] <= '9') {
			arg += key
			Global.Input += key
			editorRefreshScreen()
		} else {
			if key == "C-u" {
				Global.Input += " " + key
				editorRefreshScreen()
				key, drhl = editorGetKey()
			}
			argi := 0
			if arg != "" {
				var err error = nil
				argi, err = strconv.Atoi(arg)
				if err != nil {
					Global.Input = err.Error()
					return
				}
			}
			Global.Universal = argi
			Global.SetUniversal = true
			RunCommandForKey(key)
			if drhl {
				editorRefreshScreen()
				Global.CurrentB.updateHighlighting()
			}
			Global.SetUniversal = false
			return
		}
	}
}

func RepeatCommand() {
	cmd := Global.LastCommand
	Global.Universal = Global.LastCommandUniversal
	Global.SetUniversal = Global.LastCommandSetUniversal
	var s string
	if Global.SetUniversal {
		s = strconv.Itoa(Global.Universal) + " " + cmd.Name
	} else {
		s = cmd.Name
	}
	micromode("z", "Press z to repeat "+s, func() {
		cmd.Com()
		if !cmd.NoRepeat && macrorec {
			macro = append(macro, &EditorAction{Global.SetUniversal, Global.Universal, cmd})
		}
	})
}

func getRepeatTimes() int {
	if Global.SetUniversal && 0 < Global.Universal {
		return Global.Universal
	} else {
		return 1
	}
}

func setFillColumn() {
	if Global.SetUniversal {
		if Global.Universal > 0 {
			Global.Fillcolumn = Global.Universal
			Global.Input = fmt.Sprintf("Fill column set to %d", Global.Fillcolumn)
		} else {
			Global.Input = fmt.Sprintf("Invalid value for fill column: %d", Global.Universal)
			AddErrorMessage(Global.Input)
		}
	} else {
		fc, err := strconv.Atoi(editorPrompt("Set the fill column to", nil))
		if err != nil {
			Global.Input = "Invalid value for fill column: " + err.Error()
			AddErrorMessage(Global.Input)
		} else if fc <= 0 {
			Global.Input = fmt.Sprintf("Invalid value for fill column: %d", Global.Universal)
			AddErrorMessage(Global.Input)
		} else {
			Global.Fillcolumn = fc
			Global.Input = fmt.Sprintf("Fill column set to %d", fc)
		}
	}
}

func keyboardQuit() {
	Global.Input = "Quit"
	Global.CurrentB.regionActive = false
}

func main() {
	var dumptreequit bool
	cpuprofile := ""
	InitEditor()
	fs := flag.NewFlagSet("", flag.ExitOnError)
	fs.BoolVar(&Global.NoSyntax, "s", false, "disable syntax highlighting")
	fs.BoolVar(&Global.debug, "d", false, "enable dumps of crash logs")
	fs.BoolVar(&dumptreequit, "D", false, "dump the keybindings to stdout and quit")
	fs.StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to file")
	fs.Parse(os.Args[1:])
	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	LoadSyntaxDefs()
	LoadDefaultCommands()
	LoadKeys(Emacs)
	args := fs.Args()
	if dumptreequit {
		fmt.Println(WalkCommandTree(Emacs, ""))
		return
	}
	if Global.Input == "" {
		Global.Input = "Welcome to Emacs!"
	}
	if len(args) > 0 {
		ferr := EditorOpen(args[0])
		if ferr != nil {
			Global.Input = ferr.Error()
			AddErrorMessage(ferr.Error())
		}
		if len(args) > 1 {
			for _, fn := range args[1:] {
				buffer := &EditorBuffer{}
				buffer.MajorMode = "Unknown"
				Global.Buffers = append(Global.Buffers, buffer)
				Global.CurrentB = buffer
				ferr = EditorOpen(fn)
				if ferr != nil {
					Global.Input = ferr.Error()
					AddErrorMessage(ferr.Error())
				}
			}
			Global.CurrentB = Global.Buffers[0]
		}
	}

	InitTerm()
	defer termbox.Close()
	editorRefreshScreen()
	lastkey := "<none>"
	lt := time.Now()
	for {
		if Global.quit {
			return
		} else {
			key, drhl := editorGetKey()
			t := time.Now()
			RunCommandForKey(key)
			// A bit hacky, but this fixes some of our speed issues when pasting.
			// Don't do the optimisation if this key and the last were the same!
			if t.UnixNano()-lt.UnixNano() > TIMEOUT || lastkey == key {
				editorRefreshScreen()
			}
			if drhl {
				Global.CurrentB.updateHighlighting()
				editorRefreshScreen()
			}
			lt = time.Now()
		}
	}
}
