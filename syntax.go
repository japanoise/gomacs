package main

import (
	"strings"

	glisp "github.com/glycerine/zygomys/zygo"
	termutil "github.com/japanoise/termbox-util"
	"github.com/nsf/termbox-go"
	"github.com/zyedidia/highlight"
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
	case highlight.Groups["preproc"], highlight.Groups["special"], highlight.Groups["constant.specialChar"]:
		color = termbox.ColorLightYellow
	case highlight.Groups["comment"], highlight.Groups["preproc.shebang"]:
		color = termbox.ColorLightBlue
	case highlight.Groups["constant.string"], highlight.Groups["constant"], highlight.Groups["constant.number"], highlight.Groups["constant.specialChar"], highlight.Groups["constant.bool"]:
		color = termbox.ColorLightRed
	case highlight.Groups["type"]:
		color = termbox.ColorLightGreen
	case highlight.Groups["identifier"]:
		color = termbox.ColorLightCyan
	case highlight.Groups["statement"]:
		color = termbox.ColorLightMagenta
	case highlight.Groups["todo"]:
		color = termbox.ColorLightRed | termbox.AttrReverse
	default:
		color = termbox.ColorDefault
	}
	return color
}

func (row *EditorRow) PrintWCursor(x, y, offset, runeoff, sx int, ts string, buf *EditorBuffer) {
	if buf.regionActive && buf.region.startl <= row.idx && row.idx < buf.region.endl {
		for i := x; i <= sx; i++ {
			termbox.SetCell(i, y, ' ', termbox.AttrReverse, termbox.ColorDefault)
		}
		if buf.region.startl < row.idx {
			termutil.PrintstringColored(termbox.AttrReverse, ts, x, y)
			return
		}
	}
	color := termbox.ColorDefault
	os := 0
	ri := 0
	for in, ru := range ts {
		if x+os >= sx {
			termutil.PrintRune(x+os-1, y, '→', termbox.ColorDefault)
			return
		}
		if offset+os == buf.rx {
			termbox.SetCursor(x+os, y)
		}
		if Global.NoSyntax || buf.Highlighter == nil {
			color = termbox.ColorDefault
		} else if group, ok := row.HlMatches[ri+offset]; ok {
			color = getColorForGroup(group)
		} else if in == 0 && runeoff != 0 {
			groupi, oki := row.HlMatches[offset]
			for i := 1; !oki && i <= offset; i++ {
				groupi, oki = row.HlMatches[offset-i]
			}
			color = getColorForGroup(groupi)
		}
		// See comment in original function
		if buf.regionActive &&
			((row.idx == buf.region.startl && buf.region.startl == buf.region.endl && offset+os < buf.region.endc && offset+os >= buf.region.startc) ||
				(buf.region.startl != buf.region.endl && ((row.idx == buf.region.startl && offset+os >= buf.region.startc) || (row.idx == buf.region.endl && offset+os < buf.region.endc)))) {
			termutil.PrintRune(x+os, y, ru, termbox.AttrReverse)
		} else {
			termutil.PrintRune(x+os, y, ru, color)
		}
		os += termutil.Runewidth(ru)
		ri++
	}
	if buf.cx == row.Size {
		termbox.SetCursor(x+os, y)
	}
}

func (row *EditorRow) Print(x, y, offset, runeoff, sx int, ts string, buf *EditorBuffer) {
	if buf.regionActive && buf.region.startl <= row.idx && row.idx < buf.region.endl {
		for i := x; i <= sx; i++ {
			termbox.SetCell(i, y, ' ', termbox.AttrReverse, termbox.ColorDefault)
		}
		if buf.region.startl < row.idx {
			termutil.PrintstringColored(termbox.AttrReverse, ts, x, y)
			return
		}
	}
	color := termbox.ColorDefault
	os := 0
	ri := 0
	for in, ru := range ts {
		if x+os >= sx {
			termutil.PrintRune(x+os-1, y, '→', termbox.ColorDefault)
			return
		}
		if Global.NoSyntax || buf.Highlighter == nil {
			color = termbox.ColorDefault
		} else if group, ok := row.HlMatches[ri+offset]; ok {
			color = getColorForGroup(group)
		} else if in == 0 && runeoff != 0 {
			groupi, oki := row.HlMatches[offset]
			for i := 1; !oki && i <= offset; i++ {
				groupi, oki = row.HlMatches[offset-i]
			}
			color = getColorForGroup(groupi)
		}
		// Extremely insane boolean, but it basically is asking if we're in the region.
		// Could kick this out to a function, but it would be just as unreadable.
		// 1st line is "If the region is active"
		// 2nd line is "If the start & end are the same, and we're in between the first and last character"
		// 3rd line is "If the start & end are not the same and we're within the region"
		if buf.regionActive &&
			((row.idx == buf.region.startl && buf.region.startl == buf.region.endl && offset+os < buf.region.endc && offset+os >= buf.region.startc) ||
				(buf.region.startl != buf.region.endl && ((row.idx == buf.region.startl && offset+os >= buf.region.startc) || (row.idx == buf.region.endl && offset+os < buf.region.endc)))) {
			termutil.PrintRune(x+os, y, ru, termbox.AttrReverse)
		} else {
			termutil.PrintRune(x+os, y, ru, color)
		}
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

func editorSelectSyntaxHighlight(buf *EditorBuffer, env *glisp.Zlisp) {
	var first []byte
	if buf.NumRows > 0 {
		first = []byte(buf.Rows[0].Data)
	}
	buf.Highlighter = highlight.NewHighlighter(highlight.DetectFiletype(defs, buf.Filename, first))
	if buf.Highlighter != nil {
		buf.MajorMode = buf.Highlighter.Def.FileType
		ExecHooksForMode(env, buf.MajorMode)
	} else {
		buf.MajorMode = "Unknown"
	}
	buf.Highlight()
}
