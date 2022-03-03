package main

import (
	"fmt"

	glisp "github.com/glycerine/zygomys/zygo"
)

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

func (e *EditorBuffer) toggleMode(mode string) (bool, error) {
	if e.Modes == nil {
		e.Modes = make(ModeList)
	}
	return e.setMode(mode, !e.hasMode(mode))
}

func defMode(mode string) {
	Global.MinorModes[mode] = true
}

func (e *EditorBuffer) setMode(mode string, enabled bool) (bool, error) {
	if e.Modes == nil {
		e.Modes = make(ModeList)
	}
	if Global.MinorModes[mode] {
		e.Modes[mode] = enabled
		return e.Modes[mode], nil
	}
	return e.Modes[mode], fmt.Errorf("%s is not a defined minor mode", mode)
}

func doToggleMode(mode string) {
	enabled, err := Global.CurrentB.toggleMode(mode)
	if err != nil {
		Global.Input = err.Error()
		return
	}
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
	GoHooks   []func()
	LispHooks []glisp.SexpFunction
}

type HookList map[string]*Hooks

func loadDefaultHooks() HookList {
	ret := make(HookList)
	return ret
}

func ExecHooksForMode(env *glisp.Zlisp, mode string) {
	hooks := Global.MajorHooks[mode]
	if hooks != nil {
		for _, hook := range hooks.GoHooks {
			hook()
		}
		for _, hook := range hooks.LispHooks {
			env.Apply(&hook, []glisp.Sexp{})
		}
	}
}

func RegisterGoHookForMode(mode string, hook func()) {
	hooks := Global.MajorHooks[mode]
	if hooks == nil {
		gohooks := make([]func(), 1)
		gohooks[0] = hook
		Global.MajorHooks[mode] = &Hooks{gohooks, []glisp.SexpFunction{}}
	} else {
		hooks.GoHooks = append(hooks.GoHooks, hook)
	}
}

func RegisterLispHookForMode(mode string, hook glisp.SexpFunction) {
	hooks := Global.MajorHooks[mode]
	if hooks == nil {
		gohooks := make([]func(), 0)
		lisphooks := make([]glisp.SexpFunction, 1)
		lisphooks[0] = hook
		Global.MajorHooks[mode] = &Hooks{gohooks, lisphooks}
	} else {
		hooks.LispHooks = append(hooks.LispHooks, hook)
	}
}

func RegisterGoSaveHookForMode(mode string, hook func()) {
	RegisterGoHookForMode(mode+"-save", hook)
}

func RegisterLispSaveHookForMode(mode string, hook glisp.SexpFunction) {
	RegisterLispHookForMode(mode+"-save", hook)
}

func ExecSaveHooksForMode(env *glisp.Zlisp, mode string) {
	ExecHooksForMode(env, mode+"-save")
}

func BindKeyMajorMode(mode, key string, cmd *CommandFunc) {
	AddErrorMessage(mode + key + cmd.Name)
	if Global.MajorBindings[mode] == nil {
		Global.MajorBindings[mode] = new(CommandList)
		Global.MajorBindings[mode].Parent = true
	}
	Global.MajorBindings[mode].PutCommand(key, cmd)
}
