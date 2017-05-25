package main

import (
	"math"
	"strconv"
)

func NumStrWidth(num int) int {
	return int(math.Log10(float64(num))) + 1
}

func GetGutterWidth(NumRows int) int {
	return NumStrWidth(NumRows) + 2
}

func LineNrToString(num int) string {
	return strconv.Itoa(num)
}
