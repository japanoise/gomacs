package main

func indexPreviousBlankLine() int {
	if Global.CurrentB.cy == 0 {
		Global.Input = "Beginning of buffer"
		return 0
	}
	for i := Global.CurrentB.cy - 1; 0 < i; i-- {
		if Global.CurrentB.Rows[i].Size == 0 {
			return i
		}
	}
	return 0
}

func indexNextBlankLine() int {
	if Global.CurrentB.cy == Global.CurrentB.NumRows {
		Global.Input = "End of buffer"
		return Global.CurrentB.NumRows
	} else if Global.CurrentB.cy == Global.CurrentB.NumRows-1 {
		return Global.CurrentB.NumRows
	}
	for i := Global.CurrentB.cy + 1; i < Global.CurrentB.NumRows; i++ {
		if Global.CurrentB.Rows[i].Size == 0 {
			return i
		}
	}
	return Global.CurrentB.NumRows
}

func backwardParagraph() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		cy := indexPreviousBlankLine()
		Global.CurrentB.cy = cy
		Global.CurrentB.cx = 0
	}
}

func forwardParagraph() {
	times := getRepeatTimes()
	for i := 0; i < times; i++ {
		cy := indexNextBlankLine()
		Global.CurrentB.cy = cy
		Global.CurrentB.cx = 0
	}
}

func doFillParagraph() {
	startl := indexPreviousBlankLine()
	endl := indexNextBlankLine() - 1
	if Global.CurrentB.NumRows == 0 {
		return
	} else {
		transposeRegion(Global.CurrentB, 0, Global.CurrentB.Rows[endl].Size, startl, endl, FillString)
	}
}
