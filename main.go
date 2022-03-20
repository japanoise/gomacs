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
	"unicode/utf8"

	glisp "github.com/glycerine/zygomys/zygo"
	termutil "github.com/japanoise/termbox-util"
	"github.com/mitchellh/go-homedir"
	"github.com/nsf/termbox-go"
	"github.com/zyedidia/highlight"
)

type EditorRow struct {
	idx        int
	Size       int
	Data       string
	RenderSize int
	Render     string
	HlState    highlight.State
	HlMatches  highlight.LineMatch
	coloff     int
}

type EditorBuffer struct {
	Filename     string
	Rendername   string
	Dirty        bool
	cx           int
	cy           int
	rx           int
	rowoff       int
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
	WindowTree              *winTree
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
	MinorModes              map[string]bool
}

var Global EditorState
var Emacs *CommandList

func editorSetPrompt(prompt string) {
	Global.Prompt = prompt
}

func saveSomeBuffers(env *glisp.Zlisp) bool {
	nodirty := true
	for _, buf := range Global.Buffers {
		if buf.Dirty {
			ds, cancel := editorYesNoPrompt(fmt.Sprintf("%s has unsaved changes; save them?", buf.getRenderName()), true)
			if ds && cancel == nil {
				editorBufSave(buf, env)
			} else if cancel != nil {
				return false
			}
		}
		nodirty = nodirty && !buf.Dirty
	}
	return nodirty
}

func doSaveSomeBuffers(env *glisp.Zlisp) {
	nodirty := saveSomeBuffers(env)
	if nodirty {
		Global.Input = "All buffers saved."
	} else {
		Global.Input = "Some buffers remain unsaved."
	}
}

func saveBuffersKillEmacs(env *glisp.Zlisp) {
	nodirty := saveSomeBuffers(env)
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
	var buffer bytes.Buffer
	rx := 0
	for _, rv := range row.Data {
		if rv == '\t' {
			nextts := nextTabStop(rx)
			for i := 0; i < nextts; i++ {
				buffer.WriteByte(' ')
			}
			rx += nextts
		} else {
			buffer.WriteRune(rv)
			rx += termutil.Runewidth(rv)
		}
	}
	row.RenderSize = rx
	row.Render = buffer.String()
}

func editorReHighlightRow(row *EditorRow, buf *EditorBuffer) {
	if buf.Highlighter != nil {
		curstate := buf.State(row.idx)
		buf.Highlighter.ReHighlightStates(buf, row.idx)
		if curstate != buf.State(row.idx) {
			// If the EOL state changed, the buffer needs rehighlighting
			// as this was probably multiline comment or string.
			buf.Highlighter.HighlightMatches(buf, row.idx, buf.NumRows)
		} else {
			// Probably only this line changed.
			buf.Highlighter.ReHighlightLine(buf, row.idx)
		}
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
		len(line), line, 0, "", nil, nil, 0})
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
	Global.CurrentB.Rows[at] = &EditorRow{at, len(line), line, 0, "", nil, nil, 0}
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

func editorIndent() {
	tab := getTabString()
	if Global.SoftTab {
		buf := Global.CurrentB
		rx := editorRowCxToRx(buf.Rows[buf.cy])
		tab = tab[:nextTabStop(rx)]
	}
	editorInsertStr(tab)
}

func editorDeleteIndentation() {
	buf := Global.CurrentB
	if buf.cy == 0 || buf.NumRows <= 1 {
		return
	}

	cx := 0
	for _, ru := range buf.Rows[buf.cy].Data {
		if ru != ' ' && ru != '\t' {
			break
		}
		// Yes, we can just increment it; tabs and spaces are both one byte
		cx++
	}

	startc, startl, endc, endl := buf.Rows[buf.cy-1].Size, buf.cy-1, cx, buf.cy
	buf.cx, buf.cy = startc, startl
	buf.MarkX, buf.MarkY = endc, endl
	_, err := regionCmd(func(buf *EditorBuffer, startc, endc, startl, endl int) string {
		ret := bufKillRegion(buf, startc, endc, startl, endl)
		editorAddRegionUndo(false, startc, endc,
			startl, endl, ret)
		return ret
	})

	// "foo\nbar" -> "foo bar"
	if startc != 0 {
		editorInsertStr(" ")
		buf.cx--
		buf.Undo.paired = true
	}

	if err != nil {
		Global.Input = err.Error()
	}
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
	buf := Global.CurrentB
	for i := 0; i < times; i++ {
		if buf.cx == 0 && buf.cy == 0 {
			Global.Input = "Beginning of buffer"
			return
		}
		if buf.cy == buf.NumRows {
			return
		}
		row := buf.Rows[buf.cy]
		if buf.cx > 0 {
			rv, rs := utf8.DecodeLastRuneInString(row.Data[:buf.cx])
			if Global.SoftTab && rv == ' ' {
				for row.cxToRx(buf.cx-rs)%Global.Tabsize != 0 {
					rv, _ = utf8.DecodeLastRuneInString(row.Data[:buf.cx-rs])
					if rv != ' ' {
						break
					}
					rs++
				}
			}
			editorAddDeleteUndo(buf.cx-rs, buf.cx, buf.cy, buf.cy, row.Data[buf.cx-rs:buf.cx])
			editorRowDelChar(row, buf, buf.cx-rs, rs)
			buf.cx -= rs
		} else {
			editorAddDeleteUndo(buf.cx, buf.Rows[buf.cy-1].Size, buf.cy-1, buf.cy, row.Data)
			buf.cx = buf.Rows[buf.cy-1].Size
			editorRowAppendStr(buf.Rows[buf.cy-1], buf, row.Data)
			editorDelRow(buf.cy)
			buf.cy--
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

func EditorOpen(filename string, env *glisp.Zlisp) error {
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
	editorSelectSyntaxHighlight(Global.CurrentB, env)
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

func EditorSave(env *glisp.Zlisp) {
	editorBufSave(Global.CurrentB, env)
	ExecSaveHooksForMode(env, Global.CurrentB.MajorMode)
}

func editorBufSave(buf *EditorBuffer, env *glisp.Zlisp) {
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
	editorSelectSyntaxHighlight(buf, env)
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
		false, &winTree{false, false, true, buffer, nil, nil, nil}, 0,
		"", false, make(map[string]bool), []string{}, false, 0, false,
		loadDefaultHooks(), nil, false, 0, NewRegisterList(), 80,
		make(map[string]*CommandList), 0, 0, make(map[string]bool)}
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

func RunCommandForKey(key string, env *glisp.Zlisp) {
	//use f12 as panic button
	if key == "f12" {
		Global.quit = true
		return
	}
	var selfins *CommandFunc
	// Hack fixed (though we won't support any encoding save utf8)
	if !Global.CurrentB.hasMode("no-self-insert-mode") && utf8.RuneCountInString(key) == 1 {
		selfins = &CommandFunc{
			key,
			func(*glisp.Zlisp) {
				editorInsertStr(key)
			},
			false,
		}
	}

	Global.Input = ""
	com, comerr := GetCommand(key, Emacs, Global.MajorBindings[Global.CurrentB.MajorMode])
	if comerr != nil {
		if selfins != nil {
			selfins.Run(env)
			return
		}
		Global.Input = comerr.Error()
		return
	} else if com != nil {
		com.Run(env)
	}
}

func AddErrorMessage(msg string) {
	Global.messages = append(Global.messages, msg)
}

func SetUniversalArgument(env *glisp.Zlisp) {
	arg := ""
	for {
		key := editorGetKey()
		if (arg == "" && key == "-") || ('0' <= key[0] && key[0] <= '9') {
			arg += key
			Global.Input += key
			editorRefreshScreen()
		} else {
			if key == "C-u" {
				Global.Input += " " + key
				editorRefreshScreen()
				key = editorGetKey()
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
			RunCommandForKey(key, env)
			editorRefreshScreen()
			Global.SetUniversal = false
			return
		}
	}
}

func RepeatCommand(env *glisp.Zlisp) {
	cmd := Global.LastCommand
	Global.Universal = Global.LastCommandUniversal
	Global.SetUniversal = Global.LastCommandSetUniversal
	var s string
	if Global.SetUniversal {
		s = strconv.Itoa(Global.Universal) + " " + cmd.Name
	} else {
		s = cmd.Name
	}
	micromode("z", "Press z to repeat "+s, env, func(e *glisp.Zlisp) {
		cmd.Com(e)
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
	args := fs.Args()
	env := NewLispInterp(!dumptreequit)
	if dumptreequit {
		fmt.Println(WalkCommandTree(Emacs, ""))
		return
	}
	if Global.Input == "" {
		Global.Input = "Welcome to Emacs!"
	}
	if len(args) > 0 {
		var ferr error
		if args[0][0] == '+' && len(args[0]) > 1 && len(args) > 1 {
			num := args[0][1:]
			linum, err := strconv.Atoi(num)

			if err == nil {
				linum--
				args = args[1:]
			} else {
				AddErrorMessage(err.Error())
			}

			ferr := EditorOpen(args[0], env)
			if ferr != nil {
				Global.Input = ferr.Error()
				AddErrorMessage(ferr.Error())
				Global.CurrentB.Rows = make([]*EditorRow, 1)
				Global.CurrentB.Rows[0] = &EditorRow{
					Global.CurrentB.NumRows,
					0, "", 0, "", nil, nil, 0}
			}

			if err == nil {
				if linum >= Global.CurrentB.NumRows-1 {
					Global.CurrentB.MoveCursorToEndOfBuffer()
				} else if linum > 0 {
					Global.CurrentB.cy = linum
				}
				Global.CurrentB.cx = 0
			}
		} else {
			ferr := EditorOpen(args[0], env)
			if ferr != nil {
				Global.Input = ferr.Error()
				AddErrorMessage(ferr.Error())
				Global.CurrentB.Rows = make([]*EditorRow, 1)
				Global.CurrentB.Rows[0] = &EditorRow{Global.CurrentB.NumRows,
					0, "", 0, "", nil, nil, 0}
			}
		}
		if len(args) > 1 {
			for _, fn := range args[1:] {
				buffer := &EditorBuffer{}
				buffer.MajorMode = "Unknown"
				Global.Buffers = append(Global.Buffers, buffer)
				Global.CurrentB = buffer
				ferr = EditorOpen(fn, env)
				if ferr != nil {
					Global.Input = ferr.Error()
					AddErrorMessage(ferr.Error())
					Global.CurrentB.Rows = make([]*EditorRow, 1)
					Global.CurrentB.Rows[0] = &EditorRow{Global.CurrentB.NumRows,
						0, "", 0, "", nil, nil, 0}
				}
			}
			Global.CurrentB = Global.Buffers[0]
		}
	} else {
		Global.CurrentB.Rows = make([]*EditorRow, 1)
		Global.CurrentB.Rows[0] = &EditorRow{Global.CurrentB.NumRows,
			0, "", 0, "", nil, nil, 0}
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
