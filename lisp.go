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
`)
	env.Run()
}
