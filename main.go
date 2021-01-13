package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/mattn/go-runewidth"
	"github.com/nsf/termbox-go"
)

const promptfg = termbox.ColorDefault
const promptbg = termbox.ColorDefault
const defaultFg = termbox.ColorDefault
const defaultBg = termbox.ColorDefault
const cursorLineFg = termbox.ColorBlack
const cursorLineBg = termbox.ColorWhite
const selIndicatorBg = termbox.ColorMagenta
const selIndicatorFg = termbox.ColorBlack

var prompt string
var choices []string  // list of choices
var isSelected []bool // list of flags of selected choices
var numSelected int   // count of all selected choices
var cursorIndex int   // list index of currently selected choice
var cursorRow int     // terminal row of where current selection is
var viewTopIndex int  // list index of topmost element in selection view
var viewTopRow int    // terminal row of topmost element in selection view
var viewHeight int    // number of rows displaying choices
var termWidth int     // terminal number of columns
var termHeight int    // terminal number of rows

func minInt(x, y int) int {
	if x <= y {
		return x
	}
	return y
}

func drawString(x, y int, s string, fg, bg termbox.Attribute) {
	for _, c := range s {
		termbox.SetCell(x, y, c, fg, bg)
		x += runewidth.RuneWidth(c)
	}
}

func termClear() {
	for y := 0; y < termHeight; y++ {
		termClearRow(y)
	}
}

func termClearRow(row int) {
	for x := 0; x < termWidth; x++ {
		termbox.SetCell(x, row, ' ', defaultFg, defaultBg)
	}
}

func termClearRect(x, y, w, h int) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			termbox.SetCell(col, row, ' ', defaultFg, defaultBg)
		}
	}
}

func clearToEndOfRow(x, y int) {
	for ; x < termWidth; x++ {
		termbox.SetCell(x, y, ' ', defaultFg, defaultBg)
	}
}

func redrawAll(clear bool) {
	if clear {
		termClear()
	}

	termClear()
	redrawPromptLine()
	redrawChoices()
	redrawCommandLine()
}

func redrawChoice(prow, vindex int, underCursor, selected bool) {
	var fg termbox.Attribute
	var bg termbox.Attribute
	if underCursor {
		fg = cursorLineFg
		bg = cursorLineBg
	} else {
		fg = defaultFg
		bg = defaultBg
	}

	/* selection indicator */
	if selected {
		termbox.SetCell(0, prow, ' ', selIndicatorFg, selIndicatorBg)
	} else {
		termbox.SetCell(0, prow, ' ', defaultFg, defaultBg)
	}

	/* draw leading whitespace */
	termbox.SetCell(1, prow, ' ', fg, bg)

	/* draw text */
	x := 2
	for _, c := range choices[vindex] {
		termbox.SetCell(x, prow, c, fg, bg)
		x += runewidth.RuneWidth(c)
	}

	/* clear rest of line */
	for ii := x; ii < termWidth; ii++ {
		termbox.SetCell(x, prow, ' ', fg, bg)
		x++
	}
}

/* TODO: Refactor into using a range slice for better performance */
func redrawChoices() {
	for ii := 0; ii < minInt(len(choices)-viewTopIndex, viewHeight); ii++ {
		redrawChoice(viewTopRow+ii, viewTopIndex+ii, false, isSelected[viewTopIndex+ii])
	}
	redrawChoice(cursorRow, cursorIndex, true, isSelected[cursorIndex])
}

func redrawCommandLine() {
	redrawCurrentLineIndex()
}

func redrawCurrentLineIndex() {
	// TODO: Omit if terminal width too narrow
	log.Printf("redrawCurrentLineIndex()\n")

	nStr := fmt.Sprintf("%d", len(choices))
	log.Printf("nstr = %s\n", nStr)
	clearToEndOfRow(termWidth-(len(nStr)*2+1), termHeight-1)
	selnumStr := fmt.Sprintf("%d/%s", cursorIndex+1, nStr)
	drawString(termWidth-len(selnumStr), termHeight-1, selnumStr, defaultFg, defaultBg)
}

func redrawPromptLine() {
	if viewTopRow != 0 {
		drawString(0, 0, prompt, defaultFg, defaultBg)
	}
}

func redrawSelectionIndicator(prow int, selected bool) {
	var fg termbox.Attribute
	var bg termbox.Attribute
	if selected {
		fg = selIndicatorFg
		bg = selIndicatorBg
	} else {
		fg = defaultBg
		bg = defaultBg
	}
	termbox.SetCell(0, prow, ' ', fg, bg)
}

func uiToggleSelection() {
	if isSelected[cursorIndex] {
		numSelected--
	} else {
		numSelected++
	}
	isSelected[cursorIndex] = !isSelected[cursorIndex]
	redrawSelectionIndicator(cursorRow, isSelected[cursorIndex])
}

func main() {
	exitCode := 0
	argLog := flag.String("l", "/dev/null", "Logging output sink")
	argPrompt := flag.String("p", "", "Prompt string")
	flag.Parse()

	/* Return code idiom. */
	defer func() { os.Exit(exitCode) }()

	/* Disable logging by default. TODO: Reroute functions */
	if len(*argLog) == 0 {
		log.SetFlags(0)
	} else {
		logfile, err := os.OpenFile(*argLog, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
		if err != nil {
			log.Println(err)
			exitCode = 1
			return
		}
		defer logfile.Close()
		log.SetOutput(logfile)
	}

	/* Read lines from stdin */
	s := bufio.NewScanner(os.Stdin)
	s.Split(bufio.ScanLines)
	for s.Scan() {
		choices = append(choices, s.Text())
	}

	if err := termbox.Init(); err != nil {
		log.Println(err)
		exitCode = 1
		return
	}
	defer termbox.Close()
	termbox.SetInputMode(termbox.InputEsc)

	// init
	prompt = *argPrompt
	isSelected = make([]bool, len(choices))
	cursorIndex = 0
	viewTopIndex = 0
	termWidth, termHeight = termbox.Size()
	if len(prompt) == 0 {
		viewTopRow = 0
	} else {
		viewTopRow = 1
	}
	cursorRow = viewTopRow
	viewHeight = termHeight - viewTopRow - 1 // last line is for instructions

	redrawAll(false)
mainloop:
	for {
		termbox.Flush()
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey: // TODO: uiFocusNext, uiFocusPrev
			if ev.Ch == 'j' || ev.Key == termbox.KeyArrowDown {
				if cursorIndex < len(choices)-1 {
					cursorIndex++
					if cursorRow == viewTopRow+viewHeight-1 {
						viewTopIndex++
						redrawChoices()
					} else {
						cursorRow++
						redrawChoice(cursorRow-1, cursorIndex-1, false, isSelected[cursorIndex-1])
						redrawChoice(cursorRow, cursorIndex, true, isSelected[cursorIndex])
					}
					redrawCurrentLineIndex()
				}
			} else if ev.Ch == 'k' || ev.Key == termbox.KeyArrowUp {
				if cursorIndex > 0 {
					cursorIndex--
					if cursorRow == viewTopRow {
						viewTopIndex--
						redrawChoices()
					} else {
						cursorRow--
						redrawChoice(cursorRow+1, cursorIndex+1, false, isSelected[cursorIndex+1])
						redrawChoice(cursorRow, cursorIndex, true, isSelected[cursorIndex])
					}
					redrawCurrentLineIndex()
				}
			} else if ev.Key == termbox.KeySpace {
				uiToggleSelection()
			} else if ev.Ch == 'q' || ev.Key == termbox.KeyEsc {
				exitCode = 1
				return
			} else if ev.Key == termbox.KeyEnter {
				break mainloop
			}
		case termbox.EventResize:
			// resize internal buffer; see termbox.Size()
			termbox.Sync()

			// update geometry state variables. Note that none other
			// than the variables below change.
			termWidth, termHeight = ev.Width, ev.Height
			viewHeight = termHeight - viewTopRow - 1

			// shift cursor line into view if hidden by resize
			cursorRow = minInt(viewTopRow+viewHeight-1, cursorRow)
			cursorIndex = viewTopIndex + (cursorRow - viewTopRow)

			redrawAll(true)
		}
	}
	termbox.Close()
	if numSelected > 0 {
		for i, v := range isSelected {
			if v {
				fmt.Printf("%s\n", choices[i])
			}
		}
	} else {
		fmt.Printf("%s\n", choices[cursorIndex])
	}
}

// BUG: Erroneous behavior when line count is to small to contain prompt, status and
//   at least one line of selection.
// TODO: -s: Single selection only
// TODO: Check whether attached to terminal (what for?)
// TODO: F1 Help
// TODO: Ctrl+u/d, PgUp/PgDown 50% scroll
// TODO: Input/Output delimiters
