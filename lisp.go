package main

import (
	"errors"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/zhemao/glisp/interpreter"
	"io/ioutil"
)

func lispGetKey(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	return glisp.SexpStr(editorGetKey()), nil
}

func lispChoiceIndex(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) != 3 {
		return glisp.SexpNull, glisp.WrongNargs
	}
	var prompt string
	switch t := args[0].(type) {
	case glisp.SexpStr:
		prompt = string(t)
	default:
		return glisp.SexpNull, errors.New("Arg 1 needs to be a string")
	}
	var choices []string
	switch t := args[1].(type) {
	case glisp.SexpArray:
		choices = make([]string, len(t))
		for i, csexp := range t {
			switch choice := csexp.(type) {
			case glisp.SexpStr:
				choices[i] = string(choice)
			default:
				return glisp.SexpNull, errors.New("Arg 2 needs to be a list of strings")
			}
		}
	default:
		return glisp.SexpNull, errors.New("Arg 2 needs to be a list")
	}
	var def int
	switch t := args[2].(type) {
	case glisp.SexpInt:
		def = int(t)
	default:
		return glisp.SexpNull, errors.New("Arg 3 needs to be an int")
	}
	return glisp.SexpInt(editorChoiceIndex(prompt, choices, def)), nil
}

func lispPrompt(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) != 1 {
		return glisp.SexpNull, glisp.WrongNargs
	}
	var prompt string
	switch t := args[0].(type) {
	case glisp.SexpStr:
		prompt = string(t)
	default:
		return glisp.SexpNull, errors.New("Arg needs to be a string")
	}
	return glisp.SexpStr(editorPrompt(prompt, nil)), nil
}

func lispPromptWithCallback(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) != 2 {
		return glisp.SexpNull, glisp.WrongNargs
	}
	var prompt string
	switch t := args[0].(type) {
	case glisp.SexpStr:
		prompt = string(t)
	default:
		return glisp.SexpNull, errors.New("Arg 1 needs to be a string")
	}
	var callback glisp.SexpFunction
	switch t := args[1].(type) {
	case glisp.SexpFunction:
		callback = t
	default:
		return glisp.SexpNull, errors.New("Arg 2 needs to be a function")
	}
	return glisp.SexpStr(editorPrompt(prompt, func(a, b string) {
		env.Apply(callback, []glisp.Sexp{glisp.SexpStr(a), glisp.SexpStr(b)})
	})), nil
}

func lispYesNoCancelPrompt(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) != 1 {
		return glisp.SexpNull, glisp.WrongNargs
	}
	var prompt string
	switch t := args[0].(type) {
	case glisp.SexpStr:
		prompt = string(t)
	default:
		return glisp.SexpNull, errors.New("Arg needs to be a string")
	}
	res, err := editorYesNoPrompt(prompt, true)
	if err == nil {
		return glisp.SexpBool(res), nil
	} else {
		return glisp.SexpStr("Cancelled"), nil
	}
}

func lispYesNoPrompt(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) != 1 {
		return glisp.SexpNull, glisp.WrongNargs
	}
	var prompt string
	switch t := args[0].(type) {
	case glisp.SexpStr:
		prompt = string(t)
	default:
		return glisp.SexpNull, errors.New("Arg needs to be a string")
	}
	res, _ := editorYesNoPrompt(prompt, false)
	return glisp.SexpBool(res), nil
}

func lispPrint(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) != 1 {
		return glisp.SexpNull, glisp.WrongNargs
	}
	switch t := args[0].(type) {
	case glisp.SexpStr:
		Global.Input = string(t)
	default:
		return glisp.SexpNull, errors.New("Arg needs to be a string")
	}
	return glisp.SexpNull, nil
}

func lispRunCommand(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) != 1 {
		return glisp.SexpNull, glisp.WrongNargs
	}
	switch t := args[0].(type) {
	case glisp.SexpStr:
		cn := StrToCmdName(string(t))
		cmd := funcnames[cn]
		if cmd != nil && cmd.Com != nil {
			cmd.Com(env)
		}
	default:
		return glisp.SexpNull, errors.New("Arg needs to be a string")
	}
	return glisp.SexpNull, nil
}

func lispSingleton(f func()) glisp.GlispUserFunction {
	return func(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
		f()
		return glisp.SexpNull, nil
	}
}

func cmdAndLispFunc(e *glisp.Glisp, cmdname, lispname string, f func()) {
	e.AddFunction(lispname, lispSingleton(f))
	DefineCommand(&CommandFunc{cmdname, func(env *glisp.Glisp) { f() }})
}

func lispBindKey(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) < 2 {
		return glisp.SexpNull, glisp.WrongNargs
	}
	var arg1 string
	switch t := args[0].(type) {
	case glisp.SexpStr:
		arg1 = string(t)
	default:
		return glisp.SexpNull, errors.New("Arg 1 needs to be a string")
	}
	var arg2 glisp.SexpFunction
	switch t := args[1].(type) {
	case glisp.SexpFunction:
		arg2 = t
	case glisp.SexpStr:
		cmdname := StrToCmdName(string(t))
		cmd := funcnames[cmdname]
		if cmd == nil {
			return glisp.SexpNull, errors.New("Unknown command: " + cmdname)
		} else {
			Emacs.PutCommand(arg1, cmd)
			return glisp.SexpNull, nil
		}
	default:
		return glisp.SexpNull, errors.New("Arg 2 needs to be a string or function")
	}
	av := []glisp.Sexp{}
	if len(args) > 2 {
		av = args[2:]
	}
	Emacs.PutCommand(arg1, &CommandFunc{"lisp code", func(env *glisp.Glisp) {
		env.Apply(arg2, av)
	}})
	return glisp.SexpNull, nil
}

func lispDefineCmd(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) < 2 {
		return glisp.SexpNull, glisp.WrongNargs
	}
	var arg1 string
	switch t := args[0].(type) {
	case glisp.SexpStr:
		arg1 = StrToCmdName(string(t))
	default:
		return glisp.SexpNull, errors.New("Arg 1 needs to be a string")
	}
	var arg2 glisp.SexpFunction
	switch t := args[1].(type) {
	case glisp.SexpFunction:
		arg2 = t
	default:
		return glisp.SexpNull, errors.New("Arg 2 needs to be a function")
	}
	av := []glisp.Sexp{}
	if len(args) > 2 {
		av = args[2:]
	}
	DefineCommand(&CommandFunc{arg1, func(env *glisp.Glisp) {
		env.Apply(arg2, av)
	}})
	return glisp.SexpNull, nil
}

func lispOnlyWindow(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	return glisp.SexpBool(len(Global.Windows) == 1), nil
}

func lispSetTabStop(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) != 1 {
		return glisp.SexpNull, glisp.WrongNargs
	}
	var x int
	switch t := args[0].(type) {
	case glisp.SexpInt:
		x = int(t)
	default:
		return glisp.SexpNull, errors.New("Arg 1 needs to be an int")
	}
	Global.Tabsize = x
	return glisp.SexpNull, nil
}

func lispSetSoftTab(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) != 1 {
		return glisp.SexpNull, glisp.WrongNargs
	}
	var x bool
	switch t := args[0].(type) {
	case glisp.SexpBool:
		x = bool(t)
	default:
		return glisp.SexpNull, errors.New("Arg 1 needs to be a bool")
	}
	Global.SoftTab = x
	return glisp.SexpNull, nil
}

func lispSetSyntaxOff(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) != 1 {
		return glisp.SexpNull, glisp.WrongNargs
	}
	var x bool
	switch t := args[0].(type) {
	case glisp.SexpBool:
		x = bool(t)
	default:
		return glisp.SexpNull, errors.New("Arg 1 needs to be a bool")
	}
	Global.NoSyntax = x
	return glisp.SexpNull, nil
}

func lispGetTabStr(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	return glisp.SexpStr(getTabString()), nil
}

func lispAddDefaultMode(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) == 1 {
		var modename string
		switch t := args[0].(type) {
		case glisp.SexpStr:
			modename = StrToCmdName(string(t))
		default:
			return glisp.SexpNull, errors.New("Arg needs to be a string")
		}
		addDefaultMode(modename)
		return glisp.SexpNull, nil
	}
	return glisp.SexpNull, glisp.WrongNargs
}

func lispRemDefaultMode(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) == 1 {
		var modename string
		switch t := args[0].(type) {
		case glisp.SexpStr:
			modename = StrToCmdName(string(t))
		default:
			return glisp.SexpNull, errors.New("Arg needs to be a string")
		}
		remDefaultMode(modename)
		return glisp.SexpNull, nil
	}
	return glisp.SexpNull, glisp.WrongNargs
}

func lispSetMode(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) == 1 {
		var modename string
		switch t := args[0].(type) {
		case glisp.SexpStr:
			modename = StrToCmdName(string(t))
		default:
			return glisp.SexpNull, errors.New("Arg needs to be a string")
		}
		Global.CurrentB.toggleMode(modename)
		return glisp.SexpNull, nil
	} else if len(args) == 2 {
		var modename string
		switch t := args[0].(type) {
		case glisp.SexpStr:
			modename = StrToCmdName(string(t))
		default:
			return glisp.SexpNull, errors.New("Arg 1 needs to be a string")
		}
		var enabled bool
		switch t := args[1].(type) {
		case glisp.SexpBool:
			enabled = bool(t)
		default:
			return glisp.SexpNull, errors.New("Arg 2 needs to be a bool")
		}
		Global.CurrentB.setMode(modename, enabled)
		return glisp.SexpNull, nil
	}
	return glisp.SexpNull, glisp.WrongNargs
}

func lispHasMode(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) != 1 {
		return glisp.SexpNull, glisp.WrongNargs
	}
	switch t := args[0].(type) {
	case glisp.SexpStr:
		return glisp.SexpBool(Global.CurrentB.hasMode(StrToCmdName(string(t)))), nil
	default:
		return glisp.SexpNull, errors.New("Arg needs to be a string")
	}
}

func lispListModes(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	modes := []glisp.Sexp{}
	for _, mode := range Global.CurrentB.getEnabledModes() {
		modes = append(modes, glisp.SexpStr(mode))
	}
	return glisp.MakeList(modes), nil
}

func loadLispFunctions(env *glisp.Glisp) {
	env.AddFunction("emacsprint", lispPrint)
	cmdAndLispFunc(env, "save-buffers-kill-emacs", "emacsquit", saveBuffersKillEmacs)
	env.AddFunction("emacsbindkey", lispBindKey)
	env.AddFunction("emacsonlywindow", lispOnlyWindow)
	env.AddFunction("settabstop", lispSetTabStop)
	env.AddFunction("gettabstr", lispGetTabStr)
	env.AddFunction("setsofttab", lispSetSoftTab)
	env.AddFunction("disablesyntax", lispSetSyntaxOff)
	env.AddFunction("unbindall", lispSingleton(func() { Emacs.UnbindAll() }))
	env.AddFunction("emacsdefinecmd", lispDefineCmd)
	env.AddFunction("runemacscmd", lispRunCommand)
	env.AddFunction("setmode", lispSetMode)
	env.AddFunction("hasmode", lispHasMode)
	env.AddFunction("listmodes", lispListModes)
	env.AddFunction("adddefaultmode", lispAddDefaultMode)
	env.AddFunction("remdefaultmode", lispRemDefaultMode)
	env.AddFunction("yesnoprompt", lispYesNoPrompt)
	env.AddFunction("yesnocancelprompt", lispYesNoCancelPrompt)
	env.AddFunction("getkey", lispGetKey)
	env.AddFunction("stringprompt", lispPrompt)
	env.AddFunction("stringpromptcallback", lispPromptWithCallback)
	env.AddFunction("choiceindex", lispChoiceIndex)
	DefineCommand(&CommandFunc{"describe-key-briefly", func(env *glisp.Glisp) { DescribeKeyBriefly() }})
	DefineCommand(&CommandFunc{"run-command", RunCommand})
	DefineCommand(&CommandFunc{"redo", editorRedoAction})
	DefineCommand(&CommandFunc{"suspend-emacs", func(env *glisp.Glisp) { suspend() }})
	DefineCommand(&CommandFunc{"move-end-of-line", func(env *glisp.Glisp) { MoveCursorToEol() }})
	DefineCommand(&CommandFunc{"move-beginning-of-line", func(env *glisp.Glisp) { MoveCursorToBol() }})
	DefineCommand(&CommandFunc{"scroll-up-command", func(env *glisp.Glisp) { MoveCursorBackPage() }})
	DefineCommand(&CommandFunc{"scroll-down-command", func(env *glisp.Glisp) { MoveCursorForthPage() }})
	DefineCommand(&CommandFunc{"save-buffer", func(env *glisp.Glisp) { EditorSave() }})
	DefineCommand(&CommandFunc{"delete-char", func(env *glisp.Glisp) { Global.CurrentB.MoveCursorRight(); editorDelChar() }})
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
	DefineCommand(&CommandFunc{"save-some-buffers", func(*glisp.Glisp) { saveSomeBuffers() }})
	DefineCommand(&CommandFunc{"apropos-command", func(*glisp.Glisp) { AproposCommand() }})
	DefineCommand(&CommandFunc{"quoted-insert", func(*glisp.Glisp) { InsertRaw() }})
	if Global.debug {
		DefineCommand(&CommandFunc{"debug-undo", func(*glisp.Glisp) { showMessages(fmt.Sprint(Global.CurrentB.Undo)) }})
	}
}

func NewLispInterp() *glisp.Glisp {
	ret := glisp.NewGlisp()
	loadLispFunctions(ret)
	LoadDefaultConfig(ret)
	LoadUserConfig(ret)
	return ret
}

func LoadUserConfig(env *glisp.Glisp) {
	usr, ue := homedir.Dir()
	if ue != nil {
		Global.Input = "Error getting current user's home directory: " + ue.Error()
		AddErrorMessage(Global.Input)
		return
	}
	rc, err := ioutil.ReadFile(usr + "/.gomacs.lisp")
	if err != nil {
		AddErrorMessage(err.Error())
		return
	}
	err = env.LoadString(string(rc))
	if err != nil {
		Global.Input = "Error parsing rc file: " + err.Error()
		AddErrorMessage(Global.Input)
		return
	}
	_, err = env.Run()
	if err != nil {
		Global.Input = "Error executing rc file: " + err.Error()
		AddErrorMessage(Global.Input)
		return
	}
}

func LoadDefaultConfig(env *glisp.Glisp) {
	_, err := env.EvalString(`
(emacsbindkey "C-s" "isearch")
(emacsbindkey "C-x C-c" "save-buffers-kill-emacs")
(emacsbindkey "C-x C-s" "save-buffer")
(emacsbindkey "LEFT" "backward-char")
(emacsbindkey "C-b" "backward-char")
(emacsbindkey "RIGHT" "forward-char")
(emacsbindkey "C-f" "forward-char")
(emacsbindkey "DOWN" "next-line")
(emacsbindkey "C-n" "next-line")
(emacsbindkey "UP" "previous-line")
(emacsbindkey "C-p" "previous-line")
(emacsbindkey "Home" "move-beginning-of-line")
(emacsbindkey "End" "move-end-of-line")
(emacsbindkey "C-a" "move-beginning-of-line")
(emacsbindkey "C-e" "move-end-of-line")
(emacsbindkey "C-v" "scroll-down-command")
(emacsbindkey "M-v" "scroll-up-command")
(emacsbindkey "next" "scroll-down-command")
(emacsbindkey "prior" "scroll-up-command")
(emacsbindkey "DEL" "delete-backward-char")
(emacsbindkey "deletechar" "delete-char")
(emacsbindkey "C-d" "delete-char")
(emacsbindkey "C-j" "insert-newline-and-indent")
(emacsbindkey "RET" "insert-newline-maybe-indent")
(emacsbindkey "C-x C-f" "find-file")
(emacsbindkey "C-x b" "buffers-list")
(emacsbindkey "M-<" "beginning-of-buffer")
(emacsbindkey "M->" "end-of-buffer")
(emacsbindkey "C-_" "undo")
(emacsbindkey "TAB" "indent")
(emacsbindkey "C-x o" "other-window")
(emacsbindkey "C-x 0" "delete-window")
(emacsbindkey "C-x 1" "delete-other-windows")
(emacsbindkey "C-x 2" "split-window")
(emacsbindkey "C-x 4 C-f" "find-file-other-window")
(emacsbindkey "C-x 4 f" "find-file-other-window")
(emacsbindkey "C-x 4 b" "switch-buffer-other-window")
(emacsbindkey "C-@" "set-mark")
(emacsbindkey "C-w" "kill-region")
(emacsbindkey "M-w" "copy-region")
(emacsbindkey "C-y" "yank-region")
(emacsbindkey "M-f" "forward-word")
(emacsbindkey "M-d" "kill-word")
(emacsbindkey "M-b" "backward-word")
(emacsbindkey "M-D" "backward-kill-word")
(emacsbindkey "M-DEL" "backward-kill-word")
(emacsbindkey "C-l" "recenter-top-bottom")
(emacsbindkey "C-x k" "kill-buffer")
(emacsbindkey "C-k" "kill-line")
(emacsbindkey "C-x C-_" "redo")
(emacsbindkey "C-z" "suspend-emacs")
(emacsbindkey "C-h c" "describe-key-briefly")
(emacsbindkey "M-x" "run-command")
(emacsbindkey "C-x C-u" "upcase-region")
(emacsbindkey "C-x C-l" "downcase-region")
(emacsbindkey "M-u" "upcase-word")
(emacsbindkey "M-l" "downcase-word")
(emacsbindkey "C-h m" "show-modes")
(emacsbindkey "C-h b" "describe-bindings")
(emacsbindkey "f1" "quick-help")
(emacsbindkey "C-x d" "dired-mode")
(emacsbindkey "M-g M-g" "goto-line")
(emacsbindkey "M-g g" "goto-line")
(emacsbindkey "M-g c" "goto-char")
(emacsbindkey "C-x (" "start-macro")
(emacsbindkey "C-x )" "end-macro")
(emacsbindkey "C-x e" "end-macro-and-run")
(emacsbindkey "C-x 4 0" "kill-buffer-and-window")
(emacsbindkey "M-%" "query-replace")
(emacsbindkey "C-x =" "what-cursor-position")
(emacsbindkey "C-x s" "save-some-buffers")
(emacsbindkey "C-h a" "apropos-command")
(emacsbindkey "C-q" "quoted-insert")
`)
	if err != nil {
		fmt.Println(err.Error())
	}
}
