package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/zhemao/glisp/interpreter"
	"sort"
	"strings"
)

type CommandList struct {
	Parent   bool
	Command  *CommandFunc
	Children map[string]*CommandList
}

var funcnames map[string]*CommandFunc

type CommandFunc struct {
	Name string
	Com  func(env *glisp.Glisp)
}

func getSortedBindings(root *CommandList, pre string) []string {
	buf := bytes.Buffer{}
	cmdlist := []string{}
	for k, v := range root.Children {
		if v.Parent {
			cmdlist = append(cmdlist, getSortedBindings(v, pre+" "+k)...)
		} else {
			buf.WriteString(pre + " " + k + " - " + v.Command.Name)
			cmdlist = append(cmdlist, buf.String())
			buf = bytes.Buffer{}
		}
	}
	sort.Strings(cmdlist)
	return cmdlist
}

func WalkCommandTree(root *CommandList, pre string) string {
	return strings.Join(getSortedBindings(root, pre), "\n")
}

func DefineCommand(command *CommandFunc) {
	funcnames[command.Name] = command
}

func (c *CommandList) PutCommand(key string, command *CommandFunc) {
	if command.Name != "lisp code" && funcnames[command.Name] == nil {
		DefineCommand(command)
	}
	if c.Children == nil {
		c.Children = make(map[string]*CommandList)
	}
	keys := strings.Split(key, " ")
	if c.Children[keys[0]] == nil {
		c.Children[keys[0]] = &CommandList{false, nil, nil}
	}
	if len(keys) > 1 {
		c.Children[keys[0]].Parent = true
		c.Children[keys[0]].PutCommand(strings.Join(keys[1:], " "), command)
	} else {
		c.Children[keys[0]].Command = command
	}
}

func (c *CommandList) GetCommand(key string) (*CommandFunc, error) {
	Global.Input += key + " "
	editorRefreshScreen()
	child := c.Children[key]
	if child == nil {
		return nil, errors.New("Bad command: " + Global.Input)
	}
	if child.Parent {
		nextkey := editorGetKey()
		s, e := child.GetCommand(nextkey)
		return s, e
	} else {
		return child.Command, nil
	}
}

func (c *CommandList) UnbindAll() {
	c.Children = make(map[string]*CommandList)
}

func DescribeKeyBriefly() {
	editorSetPrompt("Describe key sequence")
	Global.Input = ""
	editorRefreshScreen()
	com, comerr := Emacs.GetCommand(editorGetKey())
	if comerr != nil {
		Global.Input += "is not bound to a command"
	} else if com != nil {
		if com.Name == "lisp code" {
			Global.Input += "runs anonymous lisp code"
		} else {
			Global.Input += "runs the command " + com.Name
		}
	} else {
		Global.Input += "is a null command"
	}
	editorSetPrompt("")
}

func RunCommand(env *glisp.Glisp) {
	cmdname := StrToCmdName(editorPrompt("Run command", nil))
	err := RunNamedCommand(env, cmdname)
	if err != nil {
		Global.Input = cmdname + ": " + err.Error()
	}
}

func RunNamedCommand(env *glisp.Glisp, cmdname string) error {
	cmd := funcnames[cmdname]
	if cmd == nil && strings.HasSuffix(cmdname, "mode") {
		cmd = &CommandFunc{
			cmdname,
			func(*glisp.Glisp) {
				doToggleMode(cmdname)
			},
		}
	}
	if cmd != nil && cmd.Com != nil {
		cmd.Com(env)
		return nil
	} else {
		return errors.New("no such command")
	}
}

func StrToCmdName(s string) string {
	return strings.Replace(strings.ToLower(s), " ", "-", -1)
}

func AproposCommand() {
	search := editorPrompt("Search for a command", nil)
	results := []string{}
	for cmd := range funcnames {
		if strings.Contains(cmd, search) {
			results = append(results, cmd)
		}
	}
	sort.Strings(results)
	if len(results) == 0 {
		Global.Input = "Nothing found for " + search
	} else {
		showMessages(results...)
		Global.Input = ""
	}
}

func LoadDefaultCommands() {
	DefineCommand(&CommandFunc{"describe-key-briefly", func(env *glisp.Glisp) { DescribeKeyBriefly() }})
	DefineCommand(&CommandFunc{"run-command", RunCommand})
	DefineCommand(&CommandFunc{"redo", editorRedoAction})
	DefineCommand(&CommandFunc{"suspend-emacs", func(env *glisp.Glisp) { suspend() }})
	DefineCommand(&CommandFunc{"move-end-of-line", func(env *glisp.Glisp) { MoveCursorToEol() }})
	DefineCommand(&CommandFunc{"move-beginning-of-line", func(env *glisp.Glisp) { MoveCursorToBol() }})
	DefineCommand(&CommandFunc{"scroll-up-command", func(env *glisp.Glisp) { MoveCursorBackPage() }})
	DefineCommand(&CommandFunc{"scroll-down-command", func(env *glisp.Glisp) { MoveCursorForthPage() }})
	DefineCommand(&CommandFunc{"save-buffer", func(env *glisp.Glisp) { EditorSave() }})
	DefineCommand(&CommandFunc{"delete-char", func(env *glisp.Glisp) { editorDelForwardChar() }})
	DefineCommand(&CommandFunc{"delete-backward-char", func(env *glisp.Glisp) { editorDelChar() }})
	DefineCommand(&CommandFunc{"find-file", func(env *glisp.Glisp) { editorFindFile() }})
	DefineCommand(&CommandFunc{"insert-newline-and-indent", func(env *glisp.Glisp) { editorInsertNewline(true) }})
	DefineCommand(&CommandFunc{"insert-newline-maybe-indent", func(env *glisp.Glisp) { editorInsertNewline(Global.CurrentB.hasMode("indent-mode")) }})
	DefineCommand(&CommandFunc{"insert-newline", func(env *glisp.Glisp) { editorInsertNewline(false) }})
	DefineCommand(&CommandFunc{"isearch", func(env *glisp.Glisp) { editorFind() }})
	DefineCommand(&CommandFunc{"buffers-list", func(env *glisp.Glisp) { editorSwitchBuffer() }})
	DefineCommand(&CommandFunc{"end-of-buffer", func(env *glisp.Glisp) { Global.CurrentB.cy = Global.CurrentB.NumRows; Global.CurrentB.cx = 0 }})
	DefineCommand(&CommandFunc{"beginning-of-buffer", func(env *glisp.Glisp) { Global.CurrentB.cy = 0; Global.CurrentB.cx = 0 }})
	DefineCommand(&CommandFunc{"undo", func(env *glisp.Glisp) { editorUndoAction() }})
	DefineCommand(&CommandFunc{"indent", func(env *glisp.Glisp) { editorInsertStr(getTabString()) }})
	DefineCommand(&CommandFunc{"other-window", func(env *glisp.Glisp) { switchWindow() }})
	DefineCommand(&CommandFunc{"delete-window", func(env *glisp.Glisp) { closeThisWindow() }})
	DefineCommand(&CommandFunc{"delete-other-windows", func(env *glisp.Glisp) { closeOtherWindows() }})
	DefineCommand(&CommandFunc{"split-window", func(env *glisp.Glisp) { splitWindows() }})
	DefineCommand(&CommandFunc{"find-file-other-window", func(env *glisp.Glisp) { callFunOtherWindow(editorFindFile) }})
	DefineCommand(&CommandFunc{"switch-buffer-other-window", func(env *glisp.Glisp) { callFunOtherWindow(editorSwitchBuffer) }})
	DefineCommand(&CommandFunc{"set-mark", func(env *glisp.Glisp) { setMark(Global.CurrentB) }})
	DefineCommand(&CommandFunc{"kill-region", func(env *glisp.Glisp) { doKillRegion() }})
	DefineCommand(&CommandFunc{"yank-region", func(env *glisp.Glisp) { doYankRegion() }})
	DefineCommand(&CommandFunc{"copy-region", func(env *glisp.Glisp) { doCopyRegion() }})
	DefineCommand(&CommandFunc{"forward-word", func(env *glisp.Glisp) { moveForwardWord() }})
	DefineCommand(&CommandFunc{"backward-word", func(env *glisp.Glisp) { moveBackWord() }})
	DefineCommand(&CommandFunc{"backward-kill-word", func(env *glisp.Glisp) { delBackWord() }})
	DefineCommand(&CommandFunc{"kill-word", func(env *glisp.Glisp) { delForwardWord() }})
	DefineCommand(&CommandFunc{"recenter-top-bottom", func(env *glisp.Glisp) { editorCentreView() }})
	DefineCommand(&CommandFunc{"kill-buffer", func(env *glisp.Glisp) { killBuffer() }})
	DefineCommand(&CommandFunc{"kill-line", func(env *glisp.Glisp) { killToEol() }})
	DefineCommand(&CommandFunc{"downcase-region", func(*glisp.Glisp) { doLCRegion() }})
	DefineCommand(&CommandFunc{"upcase-region", func(*glisp.Glisp) { doUCRegion() }})
	DefineCommand(&CommandFunc{"upcase-word", func(*glisp.Glisp) { upcaseWord() }})
	DefineCommand(&CommandFunc{"downcase-word", func(*glisp.Glisp) { downcaseWord() }})
	DefineCommand(&CommandFunc{"capitalize-word", func(*glisp.Glisp) { capitalizeWord() }})
	DefineCommand(&CommandFunc{"toggle-mode", func(*glisp.Glisp) {
		mode := editorPrompt("Which mode?", nil)
		Global.CurrentB.toggleMode(StrToCmdName(mode))
	}})
	DefineCommand(&CommandFunc{"show-modes", func(*glisp.Glisp) { showModes() }})
	DefineCommand(&CommandFunc{"forward-char", func(*glisp.Glisp) { Global.CurrentB.MoveCursorRight() }})
	DefineCommand(&CommandFunc{"backward-char", func(*glisp.Glisp) { Global.CurrentB.MoveCursorLeft() }})
	DefineCommand(&CommandFunc{"next-line", func(*glisp.Glisp) { Global.CurrentB.MoveCursorDown() }})
	DefineCommand(&CommandFunc{"previous-line", func(*glisp.Glisp) { Global.CurrentB.MoveCursorUp() }})
	DefineCommand(&CommandFunc{"describe-bindings", func(*glisp.Glisp) { showMessages(WalkCommandTree(Emacs, "")) }})
	DefineCommand(&CommandFunc{"quick-help", func(*glisp.Glisp) {
		showMessages(`Welcome to Gomacs - Go-powered emacs!

If you've not edited your rc file (~/.gomacs.lisp), here are some emergency
commands that should help you out. C-n means hold Ctrl and press n, M-n means
hold Meta (Alt on modern keyboards) and press n.

- C-x C-c - Save all buffers and quit emacs
- C-x C-s - Save currently selected buffer
- C-x C-f - Open a file (prompt)
- C-@ (control-space) - Set mark to current cursor position
- C-w - Kill (cut) the region (the space between the mark and cursor)
- M-w - Copy the region
- C-y - Yank (paste) the last thing you killed or copied.

Current key bindings:
`, WalkCommandTree(Emacs, ""))
	}})
	DefineCommand(&CommandFunc{"dired-mode", func(*glisp.Glisp) { DiredMode() }})
	DefineCommand(&CommandFunc{"goto-line", func(*glisp.Glisp) { gotoLine() }})
	DefineCommand(&CommandFunc{"goto-char", func(*glisp.Glisp) { gotoChar() }})
	DefineCommand(&CommandFunc{"start-macro", func(*glisp.Glisp) { recMacro() }})
	DefineCommand(&CommandFunc{"end-macro", func(*glisp.Glisp) { stopRecMacro() }})
	DefineCommand(&CommandFunc{"end-macro-and-run", func(e *glisp.Glisp) { doRunMacro(e) }})
	DefineCommand(&CommandFunc{"kill-buffer-and-window", func(*glisp.Glisp) { KillBufferAndWindow() }})
	DefineCommand(&CommandFunc{"view-messages", func(*glisp.Glisp) { showMessages(Global.messages...) }})
	DefineCommand(&CommandFunc{"query-replace", func(*glisp.Glisp) { doQueryReplace() }})
	DefineCommand(&CommandFunc{"replace-string", func(*glisp.Glisp) { doReplaceString() }})
	DefineCommand(&CommandFunc{"what-cursor-position", func(*glisp.Glisp) { whatCursorPosition() }})
	DefineCommand(&CommandFunc{"save-some-buffers", func(*glisp.Glisp) { doSaveSomeBuffers() }})
	DefineCommand(&CommandFunc{"apropos-command", func(*glisp.Glisp) { AproposCommand() }})
	DefineCommand(&CommandFunc{"quoted-insert", func(*glisp.Glisp) { InsertRaw() }})
	DefineCommand(&CommandFunc{"exchange-point-and-mark", func(*glisp.Glisp) { doSwapMarkAndCursor(Global.CurrentB) }})
	DefineCommand(&CommandFunc{"universal-argument", func(env *glisp.Glisp) { SetUniversalArgument(env) }})
	DefineCommand(&CommandFunc{"forward-paragraph", func(*glisp.Glisp) { forwardParagraph() }})
	DefineCommand(&CommandFunc{"backward-paragraph", func(*glisp.Glisp) { backwardParagraph() }})
	if Global.debug {
		DefineCommand(&CommandFunc{"debug-undo", func(*glisp.Glisp) { showMessages(fmt.Sprint(Global.CurrentB.Undo)) }})
		DefineCommand(&CommandFunc{"debug-universal", func(*glisp.Glisp) { showMessages(fmt.Sprint(Global.Universal), fmt.Sprint(Global.SetUniversal)) }})
	}
}
