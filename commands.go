package main

import (
	"bytes"
	"errors"
	"fmt"
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
	Name     string
	Com      func()
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
		nextkey, _ := editorGetKey()
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
	key, _ := editorGetKey()
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

func RunCommand() {
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
	err := RunNamedCommand(cmdname)
	if err != nil {
		Global.Input = cmdname + ": " + err.Error()
	}
}

func RunNamedCommand(cmdname string) error {
	cmd := funcnames[cmdname]
	if cmd == nil && strings.HasSuffix(cmdname, "mode") {
		cmd = &CommandFunc{
			cmdname,
			func() {
				doToggleMode(cmdname)
			},
			false,
		}
	}
	if cmd != nil && cmd.Com != nil {
		return cmd.Run()
	} else {
		return errors.New("no such command")
	}
}

func (cmd *CommandFunc) Run() error {
	if cmd.Com != nil {
		cmd.Com()
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
	DefineCommand(&CommandFunc{"describe-key-briefly", func() { DescribeKeyBriefly() }, false})
	DefineCommand(&CommandFunc{"run-command", RunCommand, true})
	DefineCommand(&CommandFunc{"redo", editorRedoAction, false})
	DefineCommand(&CommandFunc{"suspend-emacs", func() { suspend() }, false})
	DefineCommand(&CommandFunc{"move-end-of-line", func() { MoveCursorToEol() }, false})
	DefineCommand(&CommandFunc{"move-beginning-of-line", func() { MoveCursorToBol() }, false})
	DefineCommand(&CommandFunc{"scroll-up-command", func() { MoveCursorBackPage() }, false})
	DefineCommand(&CommandFunc{"scroll-down-command", func() { MoveCursorForthPage() }, false})
	DefineCommand(&CommandFunc{"save-buffer", func() { EditorSave() }, false})
	DefineCommand(&CommandFunc{"delete-char", func() { editorDelForwardChar() }, false})
	DefineCommand(&CommandFunc{"delete-backward-char", func() { editorDelChar() }, false})
	DefineCommand(&CommandFunc{"find-file", func() { editorFindFile() }, false})
	DefineCommand(&CommandFunc{"insert-newline-and-indent", func() { editorInsertNewline(true) }, false})
	DefineCommand(&CommandFunc{"insert-newline-maybe-indent", func() { editorInsertNewline(Global.CurrentB.hasMode("indent-mode")) }, false})
	DefineCommand(&CommandFunc{"insert-newline", func() { editorInsertNewline(false) }, false})
	DefineCommand(&CommandFunc{"isearch", func() { editorFind() }, false})
	DefineCommand(&CommandFunc{"buffers-list", func() { editorSwitchBuffer() }, false})
	DefineCommand(&CommandFunc{"end-of-buffer", func() { Global.CurrentB.MoveCursorToEndOfBuffer() }, false})
	DefineCommand(&CommandFunc{"beginning-of-buffer", func() { Global.CurrentB.cy = 0; Global.CurrentB.cx = 0 }, false})
	DefineCommand(&CommandFunc{"undo", func() { editorUndoAction() }, false})
	DefineCommand(&CommandFunc{"indent", func() { editorInsertStr(getTabString()) }, false})
	DefineCommand(&CommandFunc{"other-window", func() { switchWindow() }, false})
	DefineCommand(&CommandFunc{"delete-window", func() { closeThisWindow() }, false})
	DefineCommand(&CommandFunc{"delete-other-windows", func() { closeOtherWindows() }, false})
	DefineCommand(&CommandFunc{"split-window", func() { splitWindows() }, false})
	DefineCommand(&CommandFunc{"find-file-other-window", func() { callFunOtherWindow(func() { editorFindFile() }) }, false})
	DefineCommand(&CommandFunc{"switch-buffer-other-window", func() { callFunOtherWindow(editorSwitchBuffer) }, false})
	DefineCommand(&CommandFunc{"set-mark", func() { setMark(Global.CurrentB) }, false})
	DefineCommand(&CommandFunc{"kill-region", func() { doKillRegion() }, false})
	DefineCommand(&CommandFunc{"yank-region", func() { doYankRegion() }, false})
	DefineCommand(&CommandFunc{"copy-region", func() { doCopyRegion() }, false})
	DefineCommand(&CommandFunc{"forward-word", func() { moveForwardWord() }, false})
	DefineCommand(&CommandFunc{"backward-word", func() { moveBackWord() }, false})
	DefineCommand(&CommandFunc{"backward-kill-word", func() { delBackWord() }, false})
	DefineCommand(&CommandFunc{"kill-word", func() { delForwardWord() }, false})
	DefineCommand(&CommandFunc{"recenter-top-bottom", func() { editorCentreView() }, false})
	DefineCommand(&CommandFunc{"kill-buffer", func() { killBuffer() }, false})
	DefineCommand(&CommandFunc{"kill-line", func() { killToEol() }, false})
	DefineCommand(&CommandFunc{"downcase-region", func() { doLCRegion() }, false})
	DefineCommand(&CommandFunc{"upcase-region", func() { doUCRegion() }, false})
	DefineCommand(&CommandFunc{"upcase-word", func() { upcaseWord() }, false})
	DefineCommand(&CommandFunc{"downcase-word", func() { downcaseWord() }, false})
	DefineCommand(&CommandFunc{"capitalize-word", func() { capitalizeWord() }, false})
	DefineCommand(&CommandFunc{"toggle-mode", func() {
		mode := editorPrompt("Which mode?", nil)
		Global.CurrentB.toggleMode(StrToCmdName(mode))
	}, false})
	DefineCommand(&CommandFunc{"show-modes", func() { showModes() }, false})
	DefineCommand(&CommandFunc{"forward-char", func() { Global.CurrentB.MoveCursorRight() }, false})
	DefineCommand(&CommandFunc{"backward-char", func() { Global.CurrentB.MoveCursorLeft() }, false})
	DefineCommand(&CommandFunc{"next-line", func() { Global.CurrentB.MoveCursorDown() }, false})
	DefineCommand(&CommandFunc{"previous-line", func() { Global.CurrentB.MoveCursorUp() }, false})
	DefineCommand(&CommandFunc{"describe-bindings", func() { doDescribeBindings() }, false})
	DefineCommand(&CommandFunc{"quick-help", func() {
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
	}, false})
	DefineCommand(&CommandFunc{"dired-mode", func() { DiredMode() }, false})
	DefineCommand(&CommandFunc{"goto-line", func() { gotoLine() }, false})
	DefineCommand(&CommandFunc{"goto-char", func() { gotoChar() }, false})
	DefineCommand(&CommandFunc{"start-macro", func() { recMacro() }, true})
	DefineCommand(&CommandFunc{"end-macro", func() { stopRecMacro() }, true})
	DefineCommand(&CommandFunc{"end-macro-and-run", func() { doRunMacro() }, true})
	DefineCommand(&CommandFunc{"kill-buffer-and-window", func() { KillBufferAndWindow() }, false})
	DefineCommand(&CommandFunc{"view-messages", func() { showMessages(Global.messages...) }, false})
	DefineCommand(&CommandFunc{"query-replace", func() { doQueryReplace() }, false})
	DefineCommand(&CommandFunc{"replace-string", func() { doReplaceString() }, false})
	DefineCommand(&CommandFunc{"what-cursor-position", func() { whatCursorPosition() }, false})
	DefineCommand(&CommandFunc{"save-some-buffers", func() { doSaveSomeBuffers() }, false})
	DefineCommand(&CommandFunc{"apropos-command", func() { AproposCommand() }, false})
	DefineCommand(&CommandFunc{"quoted-insert", func() { InsertRaw() }, false})
	DefineCommand(&CommandFunc{"exchange-point-and-mark", func() { doSwapMarkAndCursor(Global.CurrentB) }, false})
	DefineCommand(&CommandFunc{"universal-argument", func() { SetUniversalArgument() }, true})
	DefineCommand(&CommandFunc{"forward-paragraph", func() { forwardParagraph() }, false})
	DefineCommand(&CommandFunc{"backward-paragraph", func() { backwardParagraph() }, false})
	DefineCommand(&CommandFunc{"zap-to-char", func() { zapToChar() }, false})
	DefineCommand(&CommandFunc{"dired-other-window", func() { callFunOtherWindow(func() { DiredMode() }) }, false})
	DefineCommand(&CommandFunc{"scroll-other-window", func() {
		callFunOtherWindowAndGoBack(func() {
			MoveCursorForthPage()
			editorScroll(GetScreenSize())
		})
	}, false})
	DefineCommand(&CommandFunc{"scroll-other-window-back", func() {
		callFunOtherWindowAndGoBack(func() {
			MoveCursorBackPage()
			editorScroll(GetScreenSize())
		})
	}, false})
	DefineCommand(&CommandFunc{"write-file", func() { editorWriteFile() }, false})
	DefineCommand(&CommandFunc{"visit-file", func() { editorVisitFile() }, false})
	if Global.debug {
		DefineCommand(&CommandFunc{"debug-undo", func() { showMessages(fmt.Sprint(Global.CurrentB.Undo, "\n", Global.CurrentB.Undo.prev)) }, false})
		DefineCommand(&CommandFunc{"debug-universal", func() { showMessages(fmt.Sprint(Global.Universal), fmt.Sprint(Global.SetUniversal)) }, false})
	}
	DefineCommand(&CommandFunc{"repeat", func() { RepeatCommand() }, true})
	DefineCommand(&CommandFunc{"display-buffer", func() { callFunOtherWindowAndGoBack(editorSwitchBuffer) }, false})
	DefineCommand(&CommandFunc{"untabify", func() { doUntabifyRegion() }, false})
	DefineCommand(&CommandFunc{"tabify", func() { doTabifyRegion() }, false})
	DefineCommand(&CommandFunc{"jump-to-register", func() { DoJumpRegister() }, false})
	DefineCommand(&CommandFunc{"copy-to-register", func() { DoSaveTextToRegister() }, false})
	DefineCommand(&CommandFunc{"kmacro-to-register", func() { DoSaveMacroToRegister() }, false})
	DefineCommand(&CommandFunc{"insert-register", func() { DoInsertTextFromRegister() }, false})
	DefineCommand(&CommandFunc{"point-to-register", func() { DoSavePositionToRegister() }, false})
	DefineCommand(&CommandFunc{"view-register", func() { DoDescribeRegister() }, false})
	DefineCommand(&CommandFunc{"fill-region", func() { doFillRegion() }, false})
	DefineCommand(&CommandFunc{"fill-paragraph", func() { doFillParagraph() }, false})
	DefineCommand(&CommandFunc{"set-fill-column", func() { setFillColumn() }, false})
	DefineCommand(&CommandFunc{"not-modified", func() { Global.CurrentB.Dirty = false; Global.Input = "Modification flag cleared." }, false})
	DefineCommand(&CommandFunc{"query-replace-regexp", func() { doQueryReplaceRegexp() }, false})
	DefineCommand(&CommandFunc{"replace-regexp", func() { doReplaceRegexp() }, false})
	DefineCommand(&CommandFunc{"shell-command", func() { doShellCmd() }, false})
	DefineCommand(&CommandFunc{"shell-command-on-region", func() { doShellCmdRegion() }, false})
	DefineCommand(&CommandFunc{"string-rectangle", func() { doStringRectangle() }, false})
	DefineCommand(&CommandFunc{"copy-rectangle-as-kill", func() { doCopyRectangle() }, false})
	DefineCommand(&CommandFunc{"copy-rectangle-to-register", func() { rectToRegister() }, false})
	DefineCommand(&CommandFunc{"kill-rectangle", func() { doKillRectangle() }, false})
	DefineCommand(&CommandFunc{"yank-rectangle", func() { doYankRectangle() }, false})
	DefineCommand(&CommandFunc{"keyboard-quit", func() { keyboardQuit() }, false})
	DefineCommand(&CommandFunc{"mouse-set-point", func() { JumpToMousePoint() }, false})
	DefineCommand(&CommandFunc{"mouse-drag-region", func() { MouseDragRegion() }, false})
	DefineCommand(&CommandFunc{"mwheel-scroll-up", func() { MouseScrollUp() }, false})
	DefineCommand(&CommandFunc{"mwheel-scroll-down", func() { MouseScrollDown() }, false})
	DefineCommand(&CommandFunc{"mouse-release", func() { MouseRelease() }, false})
	DefineCommand(&CommandFunc{"mouse-yank-primary", func() { MouseYankXsel() }, false})
	DefineCommand(&CommandFunc{"save-buffers-kill-emacs", saveBuffersKillEmacs, false})
}

// Load the default keybindings
func LoadKeys(parent *CommandList) {
	parent.PutCommand("C-s", funcnames["isearch"])
	parent.PutCommand("C-x C-c", funcnames["save-buffers-kill-emacs"])
	parent.PutCommand("C-x C-s", funcnames["save-buffer"])
	parent.PutCommand("LEFT", funcnames["backward-char"])
	parent.PutCommand("C-b", funcnames["backward-char"])
	parent.PutCommand("RIGHT", funcnames["forward-char"])
	parent.PutCommand("C-f", funcnames["forward-char"])
	parent.PutCommand("DOWN", funcnames["next-line"])
	parent.PutCommand("C-n", funcnames["next-line"])
	parent.PutCommand("UP", funcnames["previous-line"])
	parent.PutCommand("C-p", funcnames["previous-line"])
	parent.PutCommand("Home", funcnames["move-beginning-of-line"])
	parent.PutCommand("End", funcnames["move-end-of-line"])
	parent.PutCommand("C-a", funcnames["move-beginning-of-line"])
	parent.PutCommand("C-e", funcnames["move-end-of-line"])
	parent.PutCommand("C-v", funcnames["scroll-down-command"])
	parent.PutCommand("M-v", funcnames["scroll-up-command"])
	parent.PutCommand("next", funcnames["scroll-down-command"])
	parent.PutCommand("prior", funcnames["scroll-up-command"])
	parent.PutCommand("DEL", funcnames["delete-backward-char"])
	parent.PutCommand("deletechar", funcnames["delete-char"])
	parent.PutCommand("C-d", funcnames["delete-char"])
	parent.PutCommand("C-j", funcnames["insert-newline-and-indent"])
	parent.PutCommand("RET", funcnames["insert-newline-maybe-indent"])
	parent.PutCommand("C-x C-f", funcnames["find-file"])
	parent.PutCommand("C-x b", funcnames["buffers-list"])
	parent.PutCommand("M-<", funcnames["beginning-of-buffer"])
	parent.PutCommand("M->", funcnames["end-of-buffer"])
	parent.PutCommand("C-_", funcnames["undo"])
	parent.PutCommand("TAB", funcnames["indent"])
	parent.PutCommand("C-x o", funcnames["other-window"])
	parent.PutCommand("C-x 0", funcnames["delete-window"])
	parent.PutCommand("C-x 1", funcnames["delete-other-windows"])
	parent.PutCommand("C-x 2", funcnames["split-window"])
	parent.PutCommand("C-x 4 C-f", funcnames["find-file-other-window"])
	parent.PutCommand("C-x 4 f", funcnames["find-file-other-window"])
	parent.PutCommand("C-x 4 b", funcnames["switch-buffer-other-window"])
	parent.PutCommand("C-@", funcnames["set-mark"])
	parent.PutCommand("C-w", funcnames["kill-region"])
	parent.PutCommand("M-w", funcnames["copy-region"])
	parent.PutCommand("C-y", funcnames["yank-region"])
	parent.PutCommand("M-f", funcnames["forward-word"])
	parent.PutCommand("M-d", funcnames["kill-word"])
	parent.PutCommand("M-b", funcnames["backward-word"])
	parent.PutCommand("M-D", funcnames["backward-kill-word"])
	parent.PutCommand("M-DEL", funcnames["backward-kill-word"])
	parent.PutCommand("C-l", funcnames["recenter-top-bottom"])
	parent.PutCommand("C-x k", funcnames["kill-buffer"])
	parent.PutCommand("C-k", funcnames["kill-line"])
	parent.PutCommand("C-x C-_", funcnames["redo"])
	parent.PutCommand("C-z", funcnames["suspend-emacs"])
	parent.PutCommand("C-h c", funcnames["describe-key-briefly"])
	parent.PutCommand("M-x", funcnames["run-command"])
	parent.PutCommand("C-x C-u", funcnames["upcase-region"])
	parent.PutCommand("C-x C-l", funcnames["downcase-region"])
	parent.PutCommand("M-u", funcnames["upcase-word"])
	parent.PutCommand("M-l", funcnames["downcase-word"])
	parent.PutCommand("C-h m", funcnames["show-modes"])
	parent.PutCommand("C-h b", funcnames["describe-bindings"])
	parent.PutCommand("f1", funcnames["quick-help"])
	parent.PutCommand("C-x d", funcnames["dired-mode"])
	parent.PutCommand("M-g M-g", funcnames["goto-line"])
	parent.PutCommand("M-g g", funcnames["goto-line"])
	parent.PutCommand("M-g c", funcnames["goto-char"])
	parent.PutCommand("C-x (", funcnames["start-macro"])
	parent.PutCommand("C-x )", funcnames["end-macro"])
	parent.PutCommand("C-x e", funcnames["end-macro-and-run"])
	parent.PutCommand("C-x 4 0", funcnames["kill-buffer-and-window"])
	parent.PutCommand("M-%", funcnames["query-replace"])
	parent.PutCommand("C-x =", funcnames["what-cursor-position"])
	parent.PutCommand("C-x s", funcnames["save-some-buffers"])
	parent.PutCommand("C-h a", funcnames["apropos-command"])
	parent.PutCommand("C-q", funcnames["quoted-insert"])
	parent.PutCommand("C-x C-x", funcnames["exchange-point-and-mark"])
	parent.PutCommand("C-u", funcnames["universal-argument"])
	parent.PutCommand("M-{", funcnames["backward-paragraph"])
	parent.PutCommand("M-}", funcnames["forward-paragraph"])
	parent.PutCommand("M-c", funcnames["capitalize-word"])
	parent.PutCommand("M-z", funcnames["zap-to-char"])
	parent.PutCommand("C-x 4 d", funcnames["dired-other-window"])
	parent.PutCommand("C-x C-w", funcnames["write-file"])
	parent.PutCommand("C-x C-v", funcnames["visit-file"])
	parent.PutCommand("C-M-v", funcnames["scroll-other-window"])
	parent.PutCommand("C-M-z", funcnames["scroll-other-window-back"])
	parent.PutCommand("C-x z", funcnames["repeat"])
	parent.PutCommand("C-x 4 C-o", funcnames["display-buffer"])
	parent.PutCommand("C-x r j", funcnames["jump-to-register"])
	parent.PutCommand("C-x r s", funcnames["copy-to-register"])
	parent.PutCommand("C-x r i", funcnames["insert-register"])
	parent.PutCommand("C-x r C-@", funcnames["point-to-register"])
	parent.PutCommand("C-x C-k x", funcnames["kmacro-to-register"])
	parent.PutCommand("M-q", funcnames["fill-paragraph"])
	parent.PutCommand("C-x f", funcnames["set-fill-column"])
	parent.PutCommand("M-~", funcnames["not-modified"])
	parent.PutCommand("M-!", funcnames["shell-command"])
	parent.PutCommand("M-|", funcnames["shell-command-on-region"])
	parent.PutCommand("C-x r t", funcnames["string-rectangle"])
	parent.PutCommand("C-x r M-w", funcnames["copy-rectangle-as-kill"])
	parent.PutCommand("C-x r r", funcnames["copy-rectangle-to-register"])
	parent.PutCommand("C-x r C-w", funcnames["kill-rectangle"])
	parent.PutCommand("C-x r k", funcnames["kill-rectangle"])
	parent.PutCommand("C-x r y", funcnames["yank-rectangle"])
	parent.PutCommand("C-g", funcnames["keyboard-quit"])
	parent.PutCommand("mouse1", funcnames["mouse-drag-region"])
	parent.PutCommand("mouse2", funcnames["mouse-yank-primary"])
	parent.PutCommand("mouse4", funcnames["mwheel-scroll-up"])
	parent.PutCommand("mouse5", funcnames["mwheel-scroll-down"])
	parent.PutCommand("up-mouse", funcnames["mouse-release"])
}
