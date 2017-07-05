package main

import (
	"github.com/japanoise/termbox-util"
	"github.com/nsf/termbox-go"
	"github.com/zhemao/glisp/interpreter"
	"github.com/zyedidia/highlight"
	"strings"
)

var defs []*highlight.Def

// These functions implement highlight's LineStates interface for EditorBuffer
func (buf *EditorBuffer) Line(n int) string {
	return buf.Rows[n].Render
}

func (buf *EditorBuffer) LinesNum() int {
	return buf.NumRows
}

func (buf *EditorBuffer) State(n int) highlight.State {
	return buf.Rows[n].HlState
}

func (buf *EditorBuffer) SetState(n int, s highlight.State) {
	buf.Rows[n].HlState = s
}

func (buf *EditorBuffer) SetMatch(n int, m highlight.LineMatch) {
	buf.Rows[n].HlMatches = m
}

// End interface functions

func (buf *EditorBuffer) Highlight() {
	if buf.Highlighter == nil {
		return
	}
	buf.Highlighter.HighlightStates(buf)
	buf.Highlighter.HighlightMatches(buf, 0, buf.NumRows)
}

func getColorForGroup(group highlight.Group) termbox.Attribute {
	color := termbox.ColorDefault
	switch group {
	case 255:
		// Special case here for search results
		color = termbox.AttrReverse
	case highlight.Groups["type.extended"]:
		color = termbox.ColorDefault
	case highlight.Groups["preproc"], highlight.Groups["special"]:
		color = termbox.ColorYellow
	case highlight.Groups["comment"], highlight.Groups["preproc.shebang"]:
		color = termbox.ColorBlue
	case highlight.Groups["constant.string"], highlight.Groups["constant"], highlight.Groups["constant.number"], highlight.Groups["constant.specialChar"], highlight.Groups["constant.bool"]:
		color = termbox.ColorRed
	case highlight.Groups["type"]:
		color = termbox.ColorGreen
	case highlight.Groups["identifier"]:
		color = termbox.ColorCyan
	case highlight.Groups["statement"]:
		color = termbox.ColorMagenta
	default:
		color = termbox.ColorDefault
	}
	return color
}

func (row *EditorRow) HlPrint(x, y, offset, runeoff int, ts string) {
	color := termbox.ColorDefault
	os := 0
	ri := 0
	for in, ru := range ts {
		if group, ok := row.HlMatches[ri+offset]; ok {
			color = getColorForGroup(group)
		} else if in == 0 && runeoff != 0 {
			groupi, oki := row.HlMatches[offset]
			for i := 1; !oki && i <= offset; i++ {
				groupi, oki = row.HlMatches[offset-i]
			}
			color = getColorForGroup(groupi)
		}
		termutil.PrintRune(x+os, y, ru, color)
		os += termutil.Runewidth(ru)
		ri++
	}
}

func LoadSyntaxDefs() {
	for _, fname := range AssetNames() {
		if strings.HasSuffix(fname, ".yaml") {
			input := MustAsset(fname)
			d, err := highlight.ParseDef(input)
			if err != nil {
				continue
			}
			defs = append(defs, d)
		}
	}
	highlight.ResolveIncludes(defs)
}

func editorSelectSyntaxHighlight(buf *EditorBuffer, env *glisp.Glisp) {
	var first []byte
	if buf.NumRows > 0 {
		first = []byte(buf.Rows[0].Data)
	}
	buf.Highlighter = highlight.NewHighlighter(highlight.DetectFiletype(defs, buf.Filename, first))
	if buf.Highlighter != nil {
		ExecHooksForMode(env, buf.Highlighter.Def.FileType)
	}
	buf.Highlight()
}
