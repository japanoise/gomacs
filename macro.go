package main

import glisp "github.com/glycerine/zygomys/zygo"

type EditorAction struct {
	HasUniversal bool
	Universal    int
	Command      *CommandFunc
}

type EditorMacro []*EditorAction

var macro EditorMacro
var macrorec bool = false

func runMacroOnce(env *glisp.Zlisp, m []*EditorAction) {
	if m == nil || len(m) <= 0 {
		Global.Input = "Zero length or unset macro"
		return
	}
	for _, act := range m {
		if act != nil && act.Command != nil && act.Command.Com != nil {
			Global.Universal = act.Universal
			Global.SetUniversal = act.HasUniversal
			act.Command.Com(env)
			Global.SetUniversal = false
		}
	}
}

func micromode(repeatkey string, msg string, env *glisp.Zlisp, f func(*glisp.Zlisp)) {
	f(env)
	Global.Input = msg
	editorRefreshScreen()
	key := editorGetKey()
	for key == repeatkey {
		f(env)
		editorRefreshScreen()
		key = editorGetKey()
	}
	Global.SetUniversal = false
	RunCommandForKey(key, env)
	editorRefreshScreen()
}

func doRunMacro(env *glisp.Zlisp) {
	stopRecMacro()
	micromode("e", "Press e to run macro again", env, func(e *glisp.Zlisp) {
		runMacroOnce(e, macro)
	})
}

func recMacro() {
	macrorec = true
	macro = EditorMacro{}
	Global.Input = "Recording macro..."
}

func stopRecMacro() {
	macrorec = false
	Global.Input = "Stopped recording"
}
