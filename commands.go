package main

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"

	glisp "github.com/glycerine/zygomys/zygo"
)

type CommandList struct {
	Parent   bool
	Command  *CommandFunc
	Children map[string]*CommandList
}

var funcnames map[string]*CommandFunc

type CommandFunc struct {
	Name     string
	Com      func(env *glisp.Zlisp)
	NoRepeat bool
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
	if key == " " {
		c.Children[key] = &CommandList{false, nil, nil}
		c.Children[key].Parent = false
		c.Children[key].Command = command
		return
	}
	keys := strings.Split(key, " ")
	if c.Children[keys[0]] == nil {
		c.Children[keys[0]] = &CommandList{false, nil, nil}
	}
	if len(keys) > 1 {
		c.Children[keys[0]].Parent = true
		c.Children[keys[0]].PutCommand(strings.Join(keys[1:], " "), command)
	} else {
		c.Children[keys[0]].Parent = false
		c.Children[keys[0]].Command = command
	}
}

func getMousek(key string) string {
	var mousek string
	var x, y int
	_, err := fmt.Sscanf(key, "<%s %d %d>", &mousek, &x, &y)
	if err == nil {
		Global.MouseX = x
		Global.MouseY = y
		return mousek
	} else {
		return key
	}
}

func (c *CommandList) GetCommand(key string) (*CommandFunc, error) {
	Global.Input += key + " "
	key = getMousek(key)
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

func doDescribeBindings() {
	if Global.MajorBindings[Global.CurrentB.MajorMode] != nil {
		showMessages("Global bindings:", WalkCommandTree(Emacs, ""), "",
			"Bindings for major mode "+Global.CurrentB.MajorMode+":",
			WalkCommandTree(Global.MajorBindings[Global.CurrentB.MajorMode], ""))
	} else {
		showMessages(WalkCommandTree(Emacs, ""))
	}
}

func DescribeKeyBriefly() {
	editorSetPrompt("Describe key sequence")
	defer editorSetPrompt("")
	Global.Input = ""
	editorRefreshScreen()
	key := editorGetKey()
	if Global.MajorBindings[Global.CurrentB.MajorMode] != nil {
		com, comerr := Global.MajorBindings[Global.CurrentB.MajorMode].GetCommand(key)
		if com != nil && comerr == nil {
			if com.Name == "lisp code" {
				Global.Input += "runs anonymous lisp code"
			} else {
				Global.Input += "runs the command " + com.Name
			}
			return
		}
	}
	com, comerr := Emacs.GetCommand(key)
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
}

func RunCommand(env *glisp.Zlisp) {
	cmdname := StrToCmdName(tabCompletedEditorPrompt("Run command", func(prefix string) []string {
		halfcmd := StrToCmdName(prefix)
		ret := []string{}
		for cmd := range funcnames {
			if strings.HasPrefix(cmd, halfcmd) {
				ret = append(ret, cmd)
			}
		}
		return ret
	}))
	if cmdname == "" {
		Global.Input = "Cancelled."
		return
	}
	err := RunNamedCommand(env, cmdname)
	if err != nil {
		Global.Input = cmdname + ": " + err.Error()
	}
}

func RunNamedCommand(env *glisp.Zlisp, cmdname string) error {
	cmd := funcnames[cmdname]
	if cmd == nil && strings.HasSuffix(cmdname, "mode") {
		cmd = &CommandFunc{
			cmdname,
			func(*glisp.Zlisp) {
				doToggleMode(cmdname)
			},
			false,
		}
	}
	if cmd != nil && cmd.Com != nil {
		return cmd.Run(env)
	} else {
		return errors.New("no such command")
	}
}

func (cmd *CommandFunc) Run(env *glisp.Zlisp) error {
	if cmd.Com != nil {
		cmd.Com(env)
		if !cmd.NoRepeat {
			Global.LastCommand = cmd
			Global.LastCommandSetUniversal = Global.SetUniversal
			Global.LastCommandUniversal = Global.Universal
			if macrorec {
				macro = append(macro, &EditorAction{Global.SetUniversal, Global.Universal, cmd})
			}
		}
		return nil
	} else {
		return errors.New("Malformed command")
	}
}

func StrToCmdName(s string) string {
	return strings.Replace(strings.ToLower(s), " ", "-", -1)
}

func AproposCommand() {
	search := editorPrompt("Search for a command", nil)
	if search == "" {
		Global.Input = "Cancelled."
		return
	}
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
	DefineCommand(&CommandFunc{"describe-key-briefly",
		func(env *glisp.Zlisp) { DescribeKeyBriefly() }, false})
	DefineCommand(&CommandFunc{"run-command", RunCommand, true})
	DefineCommand(&CommandFunc{"redo", editorRedoAction, false})
	DefineCommand(&CommandFunc{"suspend-emacs",
		func(env *glisp.Zlisp) { suspend() }, false})
	DefineCommand(&CommandFunc{"move-end-of-line",
		func(env *glisp.Zlisp) { MoveCursorToEol() }, false})
	DefineCommand(&CommandFunc{"move-beginning-of-line",
		func(env *glisp.Zlisp) { MoveCursorToBol() }, false})
	DefineCommand(&CommandFunc{"scroll-up-command",
		func(env *glisp.Zlisp) { MoveCursorBackPage() }, false})
	DefineCommand(&CommandFunc{"scroll-down-command",
		func(env *glisp.Zlisp) { MoveCursorForthPage() }, false})
	DefineCommand(&CommandFunc{"save-buffer",
		func(env *glisp.Zlisp) { EditorSave(env) }, false})
	DefineCommand(&CommandFunc{"delete-char",
		func(env *glisp.Zlisp) { editorDelForwardChar() }, false})
	DefineCommand(&CommandFunc{"delete-backward-char",
		func(env *glisp.Zlisp) { editorDelChar() }, false})
	DefineCommand(&CommandFunc{"find-file",
		func(env *glisp.Zlisp) { editorFindFile(env) }, false})
	DefineCommand(&CommandFunc{"insert-newline-and-indent",
		func(env *glisp.Zlisp) { editorInsertNewline(true) }, false})
	DefineCommand(&CommandFunc{"insert-newline-maybe-indent",
		func(env *glisp.Zlisp) {
			editorInsertNewline(Global.CurrentB.hasMode("indent-mode"))
		}, false})
	DefineCommand(&CommandFunc{"insert-newline",
		func(env *glisp.Zlisp) { editorInsertNewline(false) }, false})
	DefineCommand(&CommandFunc{"isearch",
		func(env *glisp.Zlisp) { editorFind() }, false})
	DefineCommand(&CommandFunc{"buffers-list",
		func(env *glisp.Zlisp) { editorSwitchBuffer() }, false})
	DefineCommand(&CommandFunc{"end-of-buffer",
		func(env *glisp.Zlisp) {
			Global.CurrentB.MoveCursorToEndOfBuffer()
		}, false})
	DefineCommand(&CommandFunc{"beginning-of-buffer",
		func(env *glisp.Zlisp) {
			Global.CurrentB.cy = 0
			Global.CurrentB.cx = 0
		}, false})
	DefineCommand(&CommandFunc{"undo",
		func(env *glisp.Zlisp) { editorUndoAction() }, false})
	DefineCommand(&CommandFunc{"indent",
		func(env *glisp.Zlisp) {
			editorIndent()
		}, false})
	DefineCommand(&CommandFunc{"other-window",
		func(env *glisp.Zlisp) { switchWindow() }, false})
	DefineCommand(&CommandFunc{"delete-window",
		func(env *glisp.Zlisp) { closeThisWindow() }, false})
	DefineCommand(&CommandFunc{"delete-other-windows",
		func(env *glisp.Zlisp) { closeOtherWindows() }, false})
	DefineCommand(&CommandFunc{"split-window",
		func(env *glisp.Zlisp) { vSplit() }, false})
	DefineCommand(&CommandFunc{"split-window-right",
		func(env *glisp.Zlisp) { hSplit() }, false})
	DefineCommand(&CommandFunc{"find-file-other-window",
		func(env *glisp.Zlisp) {
			callFunOtherWindow(func() { editorFindFile(env) })
		}, false})
	DefineCommand(&CommandFunc{"switch-buffer-other-window",
		func(env *glisp.Zlisp) {
			callFunOtherWindow(editorSwitchBuffer)
		}, false})
	DefineCommand(&CommandFunc{"set-mark",
		func(env *glisp.Zlisp) {
			setMark(Global.CurrentB)
		}, false})
	DefineCommand(&CommandFunc{"kill-region",
		func(env *glisp.Zlisp) {
			doKillRegion()
		}, false})
	DefineCommand(&CommandFunc{"yank-region",
		func(env *glisp.Zlisp) {
			doYankRegion()
		}, false})
	DefineCommand(&CommandFunc{"copy-region",
		func(env *glisp.Zlisp) {
			doCopyRegion()
		}, false})
	DefineCommand(&CommandFunc{"forward-word",
		func(env *glisp.Zlisp) {
			moveForwardWord()
		}, false})
	DefineCommand(&CommandFunc{"backward-word",
		func(env *glisp.Zlisp) {
			moveBackWord()
		}, false})
	DefineCommand(&CommandFunc{"backward-kill-word",
		func(env *glisp.Zlisp) {
			delBackWord()
		}, false})
	DefineCommand(&CommandFunc{"kill-word",
		func(env *glisp.Zlisp) {
			delForwardWord()
		}, false})
	DefineCommand(&CommandFunc{"recenter-top-bottom",
		func(env *glisp.Zlisp) {
			editorCentreView()
		}, false})
	DefineCommand(&CommandFunc{"kill-buffer",
		func(env *glisp.Zlisp) {
			killBuffer()
		}, false})
	DefineCommand(&CommandFunc{"kill-line",
		func(env *glisp.Zlisp) {
			killToEol()
		}, false})
	DefineCommand(&CommandFunc{"downcase-region",
		func(*glisp.Zlisp) {
			doLCRegion()
		}, false})
	DefineCommand(&CommandFunc{"upcase-region",
		func(*glisp.Zlisp) {
			doUCRegion()
		}, false})
	DefineCommand(&CommandFunc{"upcase-word",
		func(*glisp.Zlisp) {
			upcaseWord()
		}, false})
	DefineCommand(&CommandFunc{"downcase-word",
		func(*glisp.Zlisp) {
			downcaseWord()
		}, false})
	DefineCommand(&CommandFunc{"capitalize-word",
		func(*glisp.Zlisp) {
			capitalizeWord()
		}, false})
	DefineCommand(&CommandFunc{"toggle-mode",
		func(*glisp.Zlisp) {
			mode := editorPrompt("Which mode?", nil)
			Global.CurrentB.toggleMode(StrToCmdName(mode))
		}, false})
	DefineCommand(&CommandFunc{"show-modes",
		func(*glisp.Zlisp) {
			showModes()
		}, false})
	DefineCommand(&CommandFunc{"forward-char",
		func(*glisp.Zlisp) {
			Global.CurrentB.MoveCursorRight()
		}, false})
	DefineCommand(&CommandFunc{"backward-char",
		func(*glisp.Zlisp) {
			Global.CurrentB.MoveCursorLeft()
		}, false})
	DefineCommand(&CommandFunc{"next-line",
		func(*glisp.Zlisp) {
			Global.CurrentB.MoveCursorDown()
		}, false})
	DefineCommand(&CommandFunc{"previous-line",
		func(*glisp.Zlisp) {
			Global.CurrentB.MoveCursorUp()
		}, false})
	DefineCommand(&CommandFunc{"describe-bindings",
		func(*glisp.Zlisp) {
			doDescribeBindings()
		}, false})
	DefineCommand(&CommandFunc{"quick-help", func(*glisp.Zlisp) {
		showMessages(`Welcome to Gomacs - Go-powered emacs!

If you've not edited your rc file (see README.md or the man page),
here are some emergency commands that should help you out. C-n means
hold Ctrl and press n, M-n means hold Meta (Alt on modern keyboards)
and press n.

- C-x C-c - Save all buffers and quit emacs
- C-x C-s - Save currently selected buffer
- C-x C-f - Open a file (prompt)
- C-@ (control-space) - Set mark to current cursor position
- C-w - Kill (cut) the region (the space between the mark and cursor)
- M-w - Copy the region
- C-y - Yank (paste) the last thing you killed or copied.

Current key bindings:
`, WalkCommandTree(Emacs, ""))
	}, false})
	DefineCommand(&CommandFunc{"dired-mode",
		func(env *glisp.Zlisp) {
			DiredMode(env)
		}, false})
	DefineCommand(&CommandFunc{"goto-line",
		func(*glisp.Zlisp) {
			gotoLine()
		}, false})
	DefineCommand(&CommandFunc{"goto-char",
		func(*glisp.Zlisp) {
			gotoChar()
		}, false})
	DefineCommand(&CommandFunc{"start-macro",
		func(*glisp.Zlisp) {
			recMacro()
		}, true})
	DefineCommand(&CommandFunc{"end-macro",
		func(*glisp.Zlisp) {
			stopRecMacro()
		}, true})
	DefineCommand(&CommandFunc{"end-macro-and-run",
		func(e *glisp.Zlisp) {
			doRunMacro(e)
		}, true})
	DefineCommand(&CommandFunc{"kill-buffer-and-window",
		func(*glisp.Zlisp) {
			KillBufferAndWindow()
		}, false})
	DefineCommand(&CommandFunc{"view-messages",
		func(*glisp.Zlisp) {
			showMessages(Global.messages...)
		}, false})
	DefineCommand(&CommandFunc{"query-replace",
		func(*glisp.Zlisp) {
			doQueryReplace()
		}, false})
	DefineCommand(&CommandFunc{"replace-string",
		func(*glisp.Zlisp) {
			doReplaceString()
		}, false})
	DefineCommand(&CommandFunc{"what-cursor-position",
		func(*glisp.Zlisp) {
			whatCursorPosition()
		}, false})
	DefineCommand(&CommandFunc{"save-some-buffers",
		func(env *glisp.Zlisp) {
			doSaveSomeBuffers(env)
		}, false})
	DefineCommand(&CommandFunc{"apropos-command",
		func(*glisp.Zlisp) {
			AproposCommand()
		}, false})
	DefineCommand(&CommandFunc{"quoted-insert",
		func(*glisp.Zlisp) {
			InsertRaw()
		}, false})
	DefineCommand(&CommandFunc{"exchange-point-and-mark",
		func(*glisp.Zlisp) {
			doSwapMarkAndCursor(Global.CurrentB)
		}, false})
	DefineCommand(&CommandFunc{"universal-argument",
		func(env *glisp.Zlisp) {
			SetUniversalArgument(env)
		}, true})
	DefineCommand(&CommandFunc{"forward-paragraph",
		func(*glisp.Zlisp) {
			forwardParagraph()
		}, false})
	DefineCommand(&CommandFunc{"backward-paragraph",
		func(*glisp.Zlisp) {
			backwardParagraph()
		}, false})
	DefineCommand(&CommandFunc{"zap-to-char",
		func(*glisp.Zlisp) {
			zapToChar()
		}, false})
	DefineCommand(&CommandFunc{"dired-other-window",
		func(env *glisp.Zlisp) {
			callFunOtherWindow(func() {
				DiredMode(env)
			})
		}, false})
	DefineCommand(&CommandFunc{"scroll-other-window",
		func(env *glisp.Zlisp) {
			callFunOtherWindowAndGoBack(func() {
				MoveCursorForthPage()
				editorScroll(GetScreenSize())
			})
		}, false})
	DefineCommand(&CommandFunc{"scroll-other-window-back",
		func(env *glisp.Zlisp) {
			callFunOtherWindowAndGoBack(func() {
				MoveCursorBackPage()
				editorScroll(GetScreenSize())
			})
		}, false})
	DefineCommand(&CommandFunc{"write-file",
		func(env *glisp.Zlisp) {
			editorWriteFile(env)
		}, false})
	DefineCommand(&CommandFunc{"visit-file",
		func(env *glisp.Zlisp) {
			editorVisitFile(env)
		}, false})
	if Global.debug {
		DefineCommand(&CommandFunc{"debug-undo", func(*glisp.Zlisp) { showMessages(fmt.Sprint(Global.CurrentB.Undo, "\n", Global.CurrentB.Undo.prev)) }, false})
		DefineCommand(&CommandFunc{"debug-universal", func(*glisp.Zlisp) { showMessages(fmt.Sprint(Global.Universal), fmt.Sprint(Global.SetUniversal)) }, false})
		DefineCommand(&CommandFunc{"debug-buffer", func(*glisp.Zlisp) {
			linedata := make([]string, Global.CurrentB.NumRows+2)
			linedata[0] = fmt.Sprintf("cx: %d, cy: %d", Global.CurrentB.cx, Global.CurrentB.cy)
			for i, row := range Global.CurrentB.Rows {
				linedata[i+1] = fmt.Sprintf("Size: %d, data: \"%s\"", row.Size, row.Data)
			}
			showMessages(linedata...)
		}, false})
	}
	DefineCommand(&CommandFunc{"repeat",
		func(env *glisp.Zlisp) {
			RepeatCommand(env)
		}, true})
	DefineCommand(&CommandFunc{"display-buffer",
		func(env *glisp.Zlisp) {
			callFunOtherWindowAndGoBack(editorSwitchBuffer)
		}, false})
	DefineCommand(&CommandFunc{"untabify",
		func(env *glisp.Zlisp) {
			doUntabifyRegion()
		}, false})
	DefineCommand(&CommandFunc{"tabify",
		func(env *glisp.Zlisp) {
			doTabifyRegion()
		}, false})
	DefineCommand(&CommandFunc{"jump-to-register",
		func(env *glisp.Zlisp) {
			DoJumpRegister(env)
		}, false})
	DefineCommand(&CommandFunc{"copy-to-register",
		func(env *glisp.Zlisp) {
			DoSaveTextToRegister()
		}, false})
	DefineCommand(&CommandFunc{"kmacro-to-register",
		func(env *glisp.Zlisp) {
			DoSaveMacroToRegister()
		}, false})
	DefineCommand(&CommandFunc{"insert-register",
		func(env *glisp.Zlisp) {
			DoInsertTextFromRegister()
		}, false})
	DefineCommand(&CommandFunc{"point-to-register",
		func(env *glisp.Zlisp) {
			DoSavePositionToRegister()
		}, false})
	DefineCommand(&CommandFunc{"view-register",
		func(env *glisp.Zlisp) {
			DoDescribeRegister()
		}, false})
	DefineCommand(&CommandFunc{"fill-region",
		func(env *glisp.Zlisp) {
			doFillRegion()
		}, false})
	DefineCommand(&CommandFunc{"fill-paragraph",
		func(env *glisp.Zlisp) {
			doFillParagraph()
		}, false})
	DefineCommand(&CommandFunc{"set-fill-column",
		func(env *glisp.Zlisp) {
			setFillColumn()
		}, false})
	DefineCommand(&CommandFunc{"not-modified",
		func(env *glisp.Zlisp) {
			Global.CurrentB.Dirty = false
			Global.Input = "Modification flag cleared."
		}, false})
	DefineCommand(&CommandFunc{"query-replace-regexp",
		func(env *glisp.Zlisp) {
			doQueryReplaceRegexp()
		}, false})
	DefineCommand(&CommandFunc{"replace-regexp",
		func(env *glisp.Zlisp) {
			doReplaceRegexp()
		}, false})
	DefineCommand(&CommandFunc{"shell-command",
		func(env *glisp.Zlisp) {
			doShellCmd()
		}, false})
	DefineCommand(&CommandFunc{"shell-command-on-region",
		func(env *glisp.Zlisp) {
			doShellCmdRegion()
		}, false})
	DefineCommand(&CommandFunc{"string-rectangle",
		func(*glisp.Zlisp) {
			doStringRectangle()
		}, false})
	DefineCommand(&CommandFunc{"copy-rectangle-as-kill",
		func(*glisp.Zlisp) {
			doCopyRectangle()
		}, false})
	DefineCommand(&CommandFunc{"copy-rectangle-to-register",
		func(*glisp.Zlisp) {
			rectToRegister()
		}, false})
	DefineCommand(&CommandFunc{"kill-rectangle",
		func(*glisp.Zlisp) {
			doKillRectangle()
		}, false})
	DefineCommand(&CommandFunc{"yank-rectangle",
		func(*glisp.Zlisp) {
			doYankRectangle()
		}, false})
	DefineCommand(&CommandFunc{"keyboard-quit",
		func(*glisp.Zlisp) {
			keyboardQuit()
		}, false})
	DefineCommand(&CommandFunc{"mouse-set-point",
		func(*glisp.Zlisp) {
			JumpToMousePoint()
		}, false})
	DefineCommand(&CommandFunc{"mouse-drag-region",
		func(*glisp.Zlisp) {
			MouseDragRegion()
		}, false})
	DefineCommand(&CommandFunc{"mwheel-scroll-up",
		func(*glisp.Zlisp) {
			MouseScrollUp()
		}, false})
	DefineCommand(&CommandFunc{"mwheel-scroll-down",
		func(*glisp.Zlisp) {
			MouseScrollDown()
		}, false})
	DefineCommand(&CommandFunc{"mouse-release",
		func(*glisp.Zlisp) {
			MouseRelease()
		}, false})
	DefineCommand(&CommandFunc{"mouse-yank-primary",
		func(*glisp.Zlisp) {
			MouseYankXsel()
		}, false})
	DefineCommand(&CommandFunc{"insert-space-maybe-fill",
		func(*glisp.Zlisp) {
			insertSpaceMaybeFill()
		}, false})
	DefineCommand(&CommandFunc{"fill-paragraph-or-region",
		func(env *glisp.Zlisp) {
			doFillParagraphOrRegion()
		}, false})
	DefineCommand(&CommandFunc{"auto-complete", autoComplete, false})
	DefineCommand(&CommandFunc{"transpose-chars",
		func(env *glisp.Zlisp) {
			doTransposeChars()
		}, false})
	DefineCommand(&CommandFunc{"transpose-words",
		func(env *glisp.Zlisp) {
			doTransposeWords()
		}, false})
	DefineCommand(&CommandFunc{"rotate-windows",
		func(env *glisp.Zlisp) {
			switchWindowOrientation()
		}, false})
	DefineCommand(&CommandFunc{"swap-windows",
		func(env *glisp.Zlisp) {
			swapWindows()
		}, false})
}
