package main

import (
	"bytes"
	"errors"
	"github.com/zhemao/glisp/interpreter"
	"strings"
)

type CommandList struct {
	Parent   bool
	Command  *CommandFunc
	Children map[string]*CommandList
}

var funcnames map[string]*CommandFunc

type CommandFunc struct {
	Name string
	Com  func(env *glisp.Glisp)
}

func WalkCommandTree(root *CommandList, pre string) string {
	buf := bytes.Buffer{}
	for k, v := range root.Children {
		if v.Parent {
			buf.WriteString(WalkCommandTree(v, pre+" "+k))
		} else {
			buf.WriteString(pre + " " + k + " - " + v.Command.Name)
			buf.WriteRune('\n')
		}
	}
	return buf.String()
}

func DefineCommand(command *CommandFunc) {
	funcnames[command.Name] = command
}

func (c *CommandList) PutCommand(key string, command *CommandFunc) {
	if command.Name != "lisp code" && funcnames[command.Name] == nil {
		DefineCommand(command)
	}
	if c.Children == nil {
		c.Children = make(map[string]*CommandList)
	}
	keys := strings.Split(key, " ")
	if c.Children[keys[0]] == nil {
		c.Children[keys[0]] = &CommandList{false, nil, nil}
	}
	if len(keys) > 1 {
		c.Children[keys[0]].Parent = true
		c.Children[keys[0]].PutCommand(strings.Join(keys[1:], " "), command)
	} else {
		c.Children[keys[0]].Command = command
	}
}

func (c *CommandList) GetCommand(key string) (*CommandFunc, error) {
	Global.Input += key + " "
	editorRefreshScreen()
	child := c.Children[key]
	if child == nil {
		return nil, errors.New("Bad command: " + Global.Input)
	}
	if child.Parent {
		nextkey := editorGetKey()
		s, e := child.GetCommand(nextkey)
		return s, e
	} else {
		return child.Command, nil
	}
}

func (c *CommandList) UnbindAll() {
	c.Children = make(map[string]*CommandList)
}

func DescribeKeyBriefly() {
	editorSetPrompt("Describe key sequence")
	Global.Input = ""
	editorRefreshScreen()
	com, comerr := Emacs.GetCommand(editorGetKey())
	if comerr != nil {
		Global.Input += "is not bound to a command"
	} else if com != nil {
		if com.Name == "lisp code" {
			Global.Input += "runs anonymous lisp code"
		} else {
			Global.Input += "runs the command " + com.Name
		}
	} else {
		Global.Input += "is a null command"
	}
	editorSetPrompt("")
}

func RunCommand(env *glisp.Glisp) {
	cmdname := StrToCmdName(editorPrompt("Run command", nil))
	err := RunNamedCommand(env, cmdname)
	if err != nil {
		Global.Input = cmdname + err.Error()
	}
}

func RunNamedCommand(env *glisp.Glisp, cmdname string) error {
	cmd := funcnames[cmdname]
	if cmd == nil && strings.HasSuffix(cmdname, "mode") {
		cmd = &CommandFunc{
			cmdname,
			func(*glisp.Glisp) {
				doToggleMode(cmdname)
			},
		}
	}
	if cmd != nil && cmd.Com != nil {
		cmd.Com(env)
		return nil
	} else {
		return errors.New("no such command")
	}
}

func StrToCmdName(s string) string {
	return strings.Replace(strings.ToLower(s), " ", "-", -1)
}
