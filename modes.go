package main

type ModeList map[string]bool

func (e *EditorBuffer) hasMode(mode string) bool {
	if e.Modes == nil {
		e.AddDefaultModes()
	}
	return e.Modes[mode]
}

func (e *EditorBuffer) AddDefaultModes() {
	e.Modes = make(map[string]bool)
	for mode, enabled := range Global.DefaultModes {
		e.Modes[mode] = enabled
	}
}

func (e *EditorBuffer) toggleMode(mode string) bool {
	if e.hasMode(mode) {
		e.Modes[mode] = false
	} else {
		e.Modes[mode] = true
	}
	return e.Modes[mode]
}

func (e *EditorBuffer) setMode(mode string, enabled bool) {
	if e.Modes == nil {
		e.Modes = make(ModeList)
	}
	e.Modes[mode] = enabled
}

func doToggleMode(mode string) {
	enabled := Global.CurrentB.toggleMode(mode)
	if enabled {
		Global.Input = mode + " enabled"
	} else {
		Global.Input = mode + " disabled"
	}
}

func (e *EditorBuffer) getEnabledModes() []string {
	enmodes := []string{}
	for mode, enabled := range Global.CurrentB.Modes {
		if enabled {
			enmodes = append(enmodes, mode)
		}
	}
	return enmodes
}

func showModes() {
	editorChoiceIndex("Modes in current buffer", Global.CurrentB.getEnabledModes(), 0)
}

func addDefaultMode(mode string) {
	Global.DefaultModes[mode] = true
}

func remDefaultMode(mode string) {
	Global.DefaultModes[mode] = false
}
