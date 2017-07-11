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

func lispRunCommandWithUarg(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) != 2 {
		return glisp.SexpNull, glisp.WrongNargs
	}
	switch t := args[0].(type) {
	case glisp.SexpInt:
		Global.Universal = int(t)
	default:
		return glisp.SexpNull, errors.New("Arg needs to be an int")
	}
	switch t := args[1].(type) {
	case glisp.SexpStr:
		cn := StrToCmdName(string(t))
		cmd := funcnames[cn]
		if cmd != nil && cmd.Com != nil {
			Global.SetUniversal = true
			cmd.Com(env)
			Global.SetUniversal = false
		}
	default:
		return glisp.SexpNull, errors.New("Arg 2 needs to be a string")
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
	DefineCommand(&CommandFunc{cmdname, func(env *glisp.Glisp) { f() }, false})
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
	}, false})
	return glisp.SexpNull, nil
}

func lispBindMajorModeKey(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) < 3 {
		return glisp.SexpNull, glisp.WrongNargs
	}
	var mode string
	switch t := args[0].(type) {
	case glisp.SexpStr:
		mode = string(t)
	default:
		return glisp.SexpNull, errors.New("Arg 1 needs to be a string")
	}
	var arg1 string
	switch t := args[1].(type) {
	case glisp.SexpStr:
		arg1 = string(t)
	default:
		return glisp.SexpNull, errors.New("Arg 2 needs to be a string")
	}
	var arg2 glisp.SexpFunction
	switch t := args[2].(type) {
	case glisp.SexpFunction:
		arg2 = t
	case glisp.SexpStr:
		cmdname := StrToCmdName(string(t))
		cmd := funcnames[cmdname]
		if cmd == nil {
			return glisp.SexpNull, errors.New("Unknown command: " + cmdname)
		} else {
			BindKeyMajorMode(mode, arg1, cmd)
			return glisp.SexpNull, nil
		}
	default:
		return glisp.SexpNull, errors.New("Arg 3 needs to be a string or function")
	}
	av := []glisp.Sexp{}
	if len(args) > 3 {
		av = args[3:]
	}
	BindKeyMajorMode(mode, arg1, &CommandFunc{"lisp code", func(env *glisp.Glisp) {
		env.Apply(arg2, av)
	}, false})
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
	}, false})
	return glisp.SexpNull, nil
}

func lispAddHook(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) != 2 {
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
	RegisterLispHookForMode(arg1, arg2)
	return glisp.SexpNull, nil
}

func lispAddSaveHook(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	if len(args) != 2 {
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
	RegisterLispSaveHookForMode(arg1, arg2)
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

func lispGetUniversalArgument(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	return glisp.SexpInt(Global.Universal), nil
}

func lispIsUniversalArgumentSet(env *glisp.Glisp, name string, args []glisp.Sexp) (glisp.Sexp, error) {
	return glisp.SexpBool(Global.SetUniversal), nil
}

func loadLispFunctions(env *glisp.Glisp) {
	env.AddFunction("emacsprint", lispPrint)
	cmdAndLispFunc(env, "save-buffers-kill-emacs", "emacsquit", func() { saveBuffersKillEmacs(env) })
	env.AddFunction("emacsbindkey", lispBindKey)
	env.AddFunction("emacsonlywindow", lispOnlyWindow)
	env.AddFunction("settabstop", lispSetTabStop)
	env.AddFunction("gettabstr", lispGetTabStr)
	env.AddFunction("setsofttab", lispSetSoftTab)
	env.AddFunction("disablesyntax", lispSetSyntaxOff)
	env.AddFunction("unbindall", lispSingleton(func() { Emacs.UnbindAll() }))
	env.AddFunction("emacsdefinecmd", lispDefineCmd)
	env.AddFunction("runemacscmd", lispRunCommand)
	env.AddFunction("cmduarg", lispRunCommandWithUarg)
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
	env.AddFunction("getuniversal", lispGetUniversalArgument)
	env.AddFunction("isuniversalset", lispIsUniversalArgumentSet)
	env.AddFunction("addhook", lispAddHook)
	env.AddFunction("addsavehook", lispAddSaveHook)
	env.AddFunction("bindkeymode", lispBindMajorModeKey)
	LoadDefaultCommands()
}

func NewLispInterp(loaduser bool) *glisp.Glisp {
	ret := glisp.NewGlisp()
	loadLispFunctions(ret)
	LoadDefaultConfig(ret)
	if loaduser {
		LoadUserConfig(ret)
	}
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
(emacsbindkey "C-x C-x" "exchange-point-and-mark")
(emacsbindkey "C-u" "universal-argument")
(emacsbindkey "M-{" "backward-paragraph")
(emacsbindkey "M-}" "forward-paragraph")
(emacsbindkey "M-c" "capitalize-word")
(emacsbindkey "M-z" "zap-to-char")
(emacsbindkey "C-x 4 d" "dired-other-window")
(emacsbindkey "C-x C-w" "write-file")
(emacsbindkey "C-x C-v" "visit-file")
(emacsbindkey "C-M-v" "scroll-other-window")
(emacsbindkey "C-M-z" "scroll-other-window-back")
(emacsbindkey "C-x z" "repeat")
(emacsbindkey "C-x 4 C-o" "display-buffer")
(emacsbindkey "C-x r j" "jump-to-register")
(emacsbindkey "C-x r s" "copy-to-register")
(emacsbindkey "C-x r i" "insert-register")
(emacsbindkey "C-x r C-@" "point-to-register")
(emacsbindkey "C-x C-k x" "kmacro-to-register")
(emacsbindkey "M-q" "fill-paragraph")
(emacsbindkey "C-x f" "set-fill-column")
(emacsbindkey "M-~" "not-modified")
(emacsbindkey "M-!" "shell-command")
(emacsbindkey "M-|" "shell-command-on-region")
`)
	if err != nil {
		fmt.Println(err.Error())
	}
}
