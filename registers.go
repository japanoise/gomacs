package main

import (
	"errors"
	"fmt"
	"strings"

	glisp "github.com/zhemao/glisp/interpreter"
)

type RegisterType uint8

const (
	RegisterInvalid RegisterType = iota
	RegisterText
	RegisterPos
	RegisterMacro
	RegisterRect
)

type Register struct {
	Type      RegisterType
	Text      string
	Posx      int
	Posy      int
	PosBuffer *EditorBuffer
	Macro     EditorMacro
}

type RegisterList struct {
	Registers map[string]*Register
}

func (r *RegisterList) getRegisterOrCreate(register string) *Register {
	ret := r.Registers[register]
	if ret == nil {
		ret = &Register{}
		r.Registers[register] = ret
	}
	return ret
}

func (r *RegisterList) getPositionRegister(register string) (*Register, error) {
	ret := r.Registers[register]
	if ret == nil {
		return nil, errors.New("No data in register")
	} else if ret.Type == RegisterPos {
		return ret, nil
	} else {
		return nil, errors.New("Register is not a position register")
	}
}

func (r *RegisterList) setPositionRegister(register string) {
	ret := r.getRegisterOrCreate(register)
	ret.Type = RegisterPos
	ret.PosBuffer = Global.CurrentB
	ret.Posx = Global.CurrentB.cx
	ret.Posy = Global.CurrentB.cy
}

func (r *RegisterList) storeMacroToRegister(register string) {
	stopRecMacro()
	ret := r.getRegisterOrCreate(register)
	ret.Type = RegisterMacro
	ret.Macro = make([]*EditorAction, len(macro))
	for i := range ret.Macro {
		ret.Macro[i] = macro[i]
	}
}

func (r *RegisterList) runMacroFromRegister(env *glisp.Glisp, register string) {
	ret := r.Registers[register]
	if ret == nil || ret.Type != RegisterMacro {
		Global.Input = register + " is not a macro register"
		return
	}
	stopRecMacro()
	micromode("e", "Press e to run macro again", env, func(e *glisp.Glisp) {
		runMacroOnce(e, ret.Macro)
	})
}

func (r *RegisterList) saveTextToRegister(register string) {
	reg := r.getRegisterOrCreate(register)
	reg.Type = RegisterText
	res, err := regionCmd(func(buf *EditorBuffer, startc, endc, startl, endl int) string {
		return getRegionText(buf, startc, endc, startl, endl)
	})
	if err == nil {
		reg.Text = res
		Global.CurrentB.regionActive = false
	}
}

func (r *RegisterList) jumpToPositionRegister(regname string) {
	reg := r.Registers[regname]
	if reg == nil {
		return
	} else if reg.Type == RegisterPos {
		if reg.PosBuffer != Global.CurrentB {
			win := getFocusWindow()
			win.buf = reg.PosBuffer
			Global.CurrentB = reg.PosBuffer
		}
		if reg.Posy >= Global.CurrentB.NumRows {
			Global.CurrentB.cy = Global.CurrentB.NumRows
			Global.CurrentB.cx = 0
		} else {
			Global.CurrentB.cy = reg.Posy
			row := Global.CurrentB.Rows[reg.Posy]
			if reg.Posx > row.Size {
				Global.CurrentB.cx = row.Size
			} else {
				Global.CurrentB.cx = reg.Posx
			}
		}
	}
}

func InteractiveGetRegister(prompt string) (*Register, string) {
	Global.Input = prompt
	editorRefreshScreen()
	reg := editorGetKey()
	Global.Input += reg
	return Global.Registers.Registers[reg], reg
}

func DoJumpRegister(env *glisp.Glisp) {
	register, regname := InteractiveGetRegister("Jump to register: ")
	if register == nil {
		Global.Input = "No such register " + regname
	} else if register.Type == RegisterMacro {
		Global.Registers.runMacroFromRegister(env, regname)
	} else if register.Type == RegisterPos {
		Global.Registers.jumpToPositionRegister(regname)
	} else {
		Global.Input = "Register " + regname + " can't be jumped to"
	}
}

func DoSavePositionToRegister() {
	_, regname := InteractiveGetRegister("Save position to register: ")
	Global.Registers.setPositionRegister(regname)
}

func DoSaveMacroToRegister() {
	_, regname := InteractiveGetRegister("Save macro to register: ")
	Global.Registers.storeMacroToRegister(regname)
}

func DoSaveTextToRegister() {
	_, regname := InteractiveGetRegister("Save text to register: ")
	Global.Registers.saveTextToRegister(regname)
}

func DoInsertTextFromRegister() {
	register, regname := InteractiveGetRegister("Insert text from register: ")
	if register == nil {
		Global.Input = "No such register " + regname
	} else if register.Type == RegisterText {
		doYankText(register.Text)
		Global.CurrentB.regionActive = false
	} else if register.Type == RegisterRect {
		yankRectangle(Global.CurrentB, register.Text)
		Global.CurrentB.regionActive = false
	} else {
		Global.Input = "Register " + regname + " is not a text register"
	}
}

func DoDescribeRegister() {
	register, regname := InteractiveGetRegister("Describe register: ")
	if register == nil || register.Type == RegisterInvalid {
		Global.Input = "Register " + regname + " is empty."
	} else {
		switch register.Type {
		case RegisterText:
			showMessages(regname+" is a text register.", "",
				"The data stored in this register is:",
				register.Text)
		case RegisterRect:
			showMessages(regname+" is a rectangle register.", "",
				"The data stored in this register is:",
				register.Text)
		case RegisterMacro:
			cmds := make([]string, len(register.Macro))
			for i, act := range register.Macro {
				if act != nil && act.Command != nil {
					if act.HasUniversal {
						cmds[i] = fmt.Sprintf("%s %d", act.Command.Name, act.Universal)
					} else {
						cmds[i] = act.Command.Name
					}
				}
			}
			showMessages(regname+" is a macro register.", "",
				"The macro stored in this register is:",
				strings.Join(cmds, "\n"))
		case RegisterPos:
			showMessages(regname+" is a position register.", "",
				"The buffer stored in this register is "+register.PosBuffer.Rendername,
				fmt.Sprintf("The position in the buffer is line %d, character %d",
					register.Posy+1, register.Posx),
			)
		}
	}
}

func NewRegisterList() *RegisterList {
	return &RegisterList{make(map[string]*Register)}
}
