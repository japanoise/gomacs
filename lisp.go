package main

import (
	"errors"
	"github.com/glycerine/zygomys/repl"
	"io/ioutil"
	"os/user"
)

func lispPrint(env *zygo.Glisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
	if len(args) != 1 {
		return zygo.SexpNull, zygo.WrongNargs
	}
	switch t := args[0].(type) {
	case *zygo.SexpStr:
		Global.Input = t.S
	default:
		return zygo.SexpNull, errors.New("Arg needs to be a string")
	}
	return zygo.SexpNull, nil
}

func lispSingleton(f func()) zygo.GlispUserFunction {
	return func(env *zygo.Glisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
		f()
		return zygo.SexpNull, nil
	}
}

func lispMvCurs(env *zygo.Glisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
	if len(args) != 2 {
		return zygo.SexpNull, zygo.WrongNargs
	}
	var x, y int
	switch t := args[0].(type) {
	case *zygo.SexpInt:
		x = int(t.Val)
	default:
		return zygo.SexpNull, errors.New("Arg 1 needs to be an int")
	}
	switch t := args[1].(type) {
	case *zygo.SexpInt:
		y = int(t.Val)
	default:
		return zygo.SexpNull, errors.New("Arg 2 needs to be an int")
	}
	MoveCursor(x, y)
	return zygo.SexpNull, nil
}

func lispBindKey(env *zygo.Glisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
	if len(args) != 2 {
		return zygo.SexpNull, zygo.WrongNargs
	}
	var arg1, arg2 string
	switch t := args[0].(type) {
	case *zygo.SexpStr:
		arg1 = t.S
	default:
		return zygo.SexpNull, errors.New("Arg 1 needs to be a string")
	}
	switch t := args[1].(type) {
	case *zygo.SexpStr:
		arg2 = t.S
	default:
		return zygo.SexpNull, errors.New("Arg 2 needs to be a string")
	}
	Emacs.PutCommand(arg1, arg2)
	return zygo.SexpNull, nil
}

func lispOnlyWindow(env *zygo.Glisp, name string, args []zygo.Sexp) (zygo.Sexp, error) {
	return zygo.GoToSexp(len(Global.Windows) == 1, env)
}

func loadLispFunctions(env *zygo.Glisp) {
	env.AddFunction("emacsprint", lispPrint)
	env.AddFunction("emacsquit", lispSingleton(EditorQuit))
	env.AddFunction("emacsmvcurs", lispMvCurs)
	env.AddFunction("emacsbindkey", lispBindKey)
	env.AddFunction("emacseol", lispSingleton(MoveCursorToEol))
	env.AddFunction("emacsbol", lispSingleton(MoveCursorToBol))
	env.AddFunction("emacsbackpage", lispSingleton(MoveCursorBackPage))
	env.AddFunction("emacsforwardpage", lispSingleton(MoveCursorForthPage))
	env.AddFunction("emacssave", lispSingleton(EditorSave))
	env.AddFunction("emacsdelchar", lispSingleton(editorDelChar))
	env.AddFunction("emacsaddnl", lispSingleton(editorInsertNewline))
	env.AddFunction("emacssearch", lispSingleton(editorFind))
	env.AddFunction("emacsfindfile", lispSingleton(editorFindFile))
	env.AddFunction("emacsswitchbuffer", lispSingleton(editorSwitchBuffer))
	env.AddFunction("emacseof", lispSingleton(func() { Global.CurrentB.cy = Global.CurrentB.NumRows }))
	env.AddFunction("emacsbof", lispSingleton(func() { Global.CurrentB.cy = 0 }))
	env.AddFunction("emacsundo", lispSingleton(editorUndoAction))
	env.AddFunction("emacsindent", lispSingleton(func() { editorInsertStr("\t") }))
	env.AddFunction("emacsswitchwindow", lispSingleton(switchWindow))
	env.AddFunction("emacsclosewindow", lispSingleton(closeThisWindow))
	env.AddFunction("emacscloseotherwindows", lispSingleton(closeOtherWindows))
	env.AddFunction("emacssplit", lispSingleton(splitWindows))
	env.AddFunction("emacsonlywindow", lispOnlyWindow)
	env.AddFunction("emacsopenotherwindow", lispSingleton(func() { callFunOtherWindow(editorFindFile) }))
	env.AddFunction("emacsswitchbufferotherwindow", lispSingleton(func() { callFunOtherWindow(editorSwitchBuffer) }))
	env.AddFunction("emacssetmark", lispSingleton(func() { setMark(Global.CurrentB) }))
	env.AddFunction("emacskillregion", lispSingleton(doKillRegion))
	env.AddFunction("emacsyankregion", lispSingleton(doYankRegion))
	env.AddFunction("emacscopyregion", lispSingleton(doCopyRegion))
}

func NewLispInterp() *zygo.Glisp {
	ret := zygo.NewGlisp()
	loadLispFunctions(ret)
	LoadDefaultConfig(ret)
	LoadUserConfig(ret)
	return ret
}

func LoadUserConfig(env *zygo.Glisp) {
	usr, ue := user.Current()
	if ue != nil {
		Global.Input = "Error getting current user: " + ue.Error()
		return
	}
	rc, err := ioutil.ReadFile(usr.HomeDir + "/.gomacs.lisp")
	if err != nil {
		Global.Input = "Error loading rc file: " + err.Error()
		return
	}
	env.LoadString(string(rc))
	env.Run()
}

func LoadDefaultConfig(env *zygo.Glisp) {
	env.LoadString(`
(emacsbindkey "C-s" "(emacssearch)")
(emacsbindkey "C-x C-c" "(emacsquit)")
(emacsbindkey "C-x C-s" "(emacssave)")
(emacsbindkey "LEFT" "(emacsmvcurs -1 0)")
(emacsbindkey "C-b" "(emacsmvcurs -1 0)")
(emacsbindkey "RIGHT" "(emacsmvcurs 1 0)")
(emacsbindkey "C-f" "(emacsmvcurs 1 0)")
(emacsbindkey "DOWN" "(emacsmvcurs 0 1)")
(emacsbindkey "C-n" "(emacsmvcurs 0 1)")
(emacsbindkey "UP" "(emacsmvcurs 0 -1)")
(emacsbindkey "C-p" "(emacsmvcurs 0 -1)")
(emacsbindkey "Home" "(emacsbol)")
(emacsbindkey "End" "(emacseol)")
(emacsbindkey "C-a" "(emacsbol)")
(emacsbindkey "C-e" "(emacseol)")
(emacsbindkey "C-v" "(emacsforwardpage)")
(emacsbindkey "M-v" "(emacsbackpage)")
(emacsbindkey "next" "(emacsforwardpage)")
(emacsbindkey "prior" "(emacsbackpage)")
(emacsbindkey "DEL" "(emacsdelchar)")
(emacsbindkey "deletechar" "(emacsmvcurs 1 0) (emacsdelchar)")
(emacsbindkey "RET" "(emacsaddnl)")
(emacsbindkey "C-x C-f" "(emacsfindfile)")
(emacsbindkey "C-x b" "(emacsswitchbuffer)")
(emacsbindkey "M-<" "(emacsbof)")
(emacsbindkey "M->" "(emacseof)")
(emacsbindkey "C-_" "(emacsundo)")
(emacsbindkey "TAB" "(emacsindent)")
(emacsbindkey "C-x o" "(emacsswitchwindow)")
(emacsbindkey "C-x 0" "(emacsclosewindow)")
(emacsbindkey "C-x 1" "(emacscloseotherwindows)")
(emacsbindkey "C-x 2" "(emacssplit)")
(emacsbindkey "C-x 4 C-f" "(emacsopenotherwindow)")
(emacsbindkey "C-x 4 f" "(emacsopenotherwindow)")
(emacsbindkey "C-x 4 b" "(emacsswitchbufferotherwindow)")
(emacsbindkey "C-@" "(emacssetmark)")
(emacsbindkey "C-w" "(emacskillregion)")
(emacsbindkey "M-w" "(emacscopyregion)")
(emacsbindkey "C-y" "(emacsyankregion)")
`)
	env.Run()
}
