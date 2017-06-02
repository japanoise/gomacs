package main

import (
	"github.com/zhemao/glisp/interpreter"
)

var macro []*CommandFunc
var macrorec bool = false

func runMacroOnce(env *glisp.Glisp) {
	if macro == nil || len(macro) <= 0 {
		Global.Input = "Zero length or unset macro"
		return
	}
	for _, cmd := range macro {
		cmd.Com(env)
	}
}

func micromode(repeatkey string, msg string, env *glisp.Glisp, f func(*glisp.Glisp)) {
	f(env)
	Global.Input = msg
	editorRefreshScreen()
	key := editorGetKey()
	for key == repeatkey {
		f(env)
		editorRefreshScreen()
		key = editorGetKey()
	}
	RunCommandForKey(key, env)
}

func doRunMacro(env *glisp.Glisp) {
	stopRecMacro()
	micromode("e", "Press e to run macro again", env, runMacroOnce)
}

func recMacro() {
	macrorec = true
	macro = []*CommandFunc{}
	Global.Input = "Recording macro..."
}

func stopRecMacro() {
	if macrorec {
		macro = macro[:len(macro)-1]
	}
	macrorec = false
	Global.Input = "Stopped recording"
}
