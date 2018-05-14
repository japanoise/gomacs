package main

// Minor Modes
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
	modes := Global.CurrentB.getEnabledModes()
	if len(modes) == 0 {
		Global.Input = "Current buffer has no modes enabled."
	} else {
		showMessages(append([]string{"Modes for " +
			Global.CurrentB.getFilename(), ""}, modes...)...)
	}
}

func addDefaultMode(mode string) {
	Global.DefaultModes[mode] = true
}

func remDefaultMode(mode string) {
	Global.DefaultModes[mode] = false
}

//Major Modes
type Hooks struct {
	GoHooks []func()
}

type HookList map[string]*Hooks

func loadDefaultHooks() HookList {
	ret := make(HookList)
	return ret
}

func ExecHooksForMode(mode string) {
	hooks := Global.MajorHooks[mode]
	if hooks != nil {
		for _, hook := range hooks.GoHooks {
			hook()
		}
	}
}

func RegisterGoHookForMode(mode string, hook func()) {
	hooks := Global.MajorHooks[mode]
	if hooks == nil {
		gohooks := make([]func(), 1)
		gohooks[0] = hook
		Global.MajorHooks[mode] = &Hooks{gohooks}
	} else {
		hooks.GoHooks = append(hooks.GoHooks, hook)
	}
}

func RegisterGoSaveHookForMode(mode string, hook func()) {
	RegisterGoHookForMode(mode+"-save", hook)
}

func ExecSaveHooksForMode(mode string) {
	ExecHooksForMode(mode + "-save")
}

func BindKeyMajorMode(mode, key string, cmd *CommandFunc) {
	AddErrorMessage(mode + key + cmd.Name)
	if Global.MajorBindings[mode] == nil {
		Global.MajorBindings[mode] = new(CommandList)
		Global.MajorBindings[mode].Parent = true
	}
	Global.MajorBindings[mode].PutCommand(key, cmd)
}
