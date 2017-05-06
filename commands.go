package main

import (
	"errors"
	"strings"
)

type CommandList struct {
	Parent   bool
	Command  string
	Children map[string]*CommandList
}

func (c *CommandList) PutCommand(key string, command string) {
	if c.Children == nil {
		c.Children = make(map[string]*CommandList)
	}
	keys := strings.Split(key, " ")
	if c.Children[keys[0]] == nil {
		c.Children[keys[0]] = &CommandList{false, "", nil}
	}
	if len(keys) > 1 {
		c.Children[keys[0]].Parent = true
		c.Children[keys[0]].PutCommand(strings.Join(keys[1:], " "), command)
	} else {
		c.Children[keys[0]].Command = command
	}
}

func (c *CommandList) GetCommand(key string) (string, error) {
	Global.Input += key + " "
	editorRefreshScreen()
	child := c.Children[key]
	if child == nil {
		return "", errors.New("Bad command: " + Global.Input)
	}
	if child.Parent {
		nextkey := editorGetKey()
		s, e := child.GetCommand(nextkey)
		return s, e
	} else {
		return child.Command, nil
	}
}
