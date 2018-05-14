package main

type EditorAction struct {
	HasUniversal bool
	Universal    int
	Command      *CommandFunc
}

type EditorMacro []*EditorAction

var macro EditorMacro
var macrorec bool = false

func runMacroOnce(m []*EditorAction) {
	if m == nil || len(m) <= 0 {
		Global.Input = "Zero length or unset macro"
		return
	}
	for _, act := range m {
		if act != nil && act.Command != nil && act.Command.Com != nil {
			Global.Universal = act.Universal
			Global.SetUniversal = act.HasUniversal
			act.Command.Com()
			Global.SetUniversal = false
		}
	}
}

func micromode(repeatkey string, msg string, f func()) {
	f()
	Global.Input = msg
	editorRefreshScreen()
	key, drhl := editorGetKey()
	for key == repeatkey {
		f()
		editorRefreshScreen()
		key, drhl = editorGetKey()
	}
	Global.SetUniversal = false
	RunCommandForKey(key)
	if drhl {
		editorRefreshScreen()
		Global.CurrentB.updateHighlighting()
	}
}

func doRunMacro() {
	stopRecMacro()
	micromode("e", "Press e to run macro again", func() {
		runMacroOnce(macro)
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
