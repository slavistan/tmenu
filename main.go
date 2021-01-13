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
const defaultfg = termbox.ColorDefault
const defaultbg = termbox.ColorDefault
const selfg = termbox.ColorBlack
const selbg = termbox.ColorWhite
const selIndicatorBg = termbox.ColorMagenta
const selIndicatorFg = termbox.ColorBlack

var prompt string
var choices []string  // list of choices
var isSelected []bool // list of flags of selected choices
var numSelected int   // count of selected choices
var vsel int          // list index of current selection [0; num choices]
var psel int          // terminal row of where current selection is
var vtop int          // list index of topmost element in selection view
var ptop int          // terminal row of where topmost choice is
var pheight int       // number of rows which show selections
var width int         // term width
var height int        // term height
var selTop int        // table index of topmost choice in selection area
var selBottom int     // table index of last choice in selection area

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
	for y := 0; y < height; y++ {
		termClearRow(y)
	}
}

func termClearRow(row int) {
	for x := 0; x < width; x++ {
		termbox.SetCell(x, row, ' ', defaultfg, defaultbg)
	}
}

func termClearRect(x, y, w, h int) {
	for row := y; row < y+h; row++ {
		for col := x; col < x+w; col++ {
			termbox.SetCell(col, row, ' ', defaultfg, defaultbg)
		}
	}
}

func clearToEndOfRow(x, y int) {
	for ; x < width; x++ {
		termbox.SetCell(x, y, ' ', defaultfg, defaultbg)
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

func redrawChoice(prow, vindex int, selected bool) {
	var fg termbox.Attribute
	var bg termbox.Attribute
	if selected {
		fg = selfg
		bg = selbg
	} else {
		fg = defaultfg
		bg = defaultbg
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
	for ii := x; ii < width; ii++ {
		termbox.SetCell(x, prow, ' ', fg, bg)
		x++
	}
}

/* TODO: Refactor into using a range slice for better performance */
func redrawChoices() {
	for ii := 0; ii < minInt(len(choices)-vtop, pheight); ii++ {
		redrawChoice(ptop+ii, vtop+ii, false)
		if isSelected[vtop+ii] {
			termbox.SetCell(0, ptop+ii, ' ', selIndicatorFg, selIndicatorBg)
		} else {
			termbox.SetCell(0, ptop+ii, ' ', defaultfg, defaultbg)
		}
	}
	redrawChoice(psel, vsel, true)
}

func redrawCommandLine() {
	redrawCurrentLineIndex()
}

func redrawCurrentLineIndex() {
	// TODO: Omit if terminal width too narrow
	log.Printf("redrawCurrentLineIndex()\n")

	nStr := fmt.Sprintf("%d", len(choices))
	log.Printf("nstr = %s\n", nStr)
	clearToEndOfRow(width-(len(nStr)*2+1), height-1)
	selnumStr := fmt.Sprintf("%d/%s", vsel+1, nStr)
	drawString(width-len(selnumStr), height-1, selnumStr, defaultfg, defaultbg)
}

func redrawPromptLine() {
	if ptop != 0 {
		drawString(0, 0, prompt, defaultfg, defaultbg)
	}
}

func redrawSelectionIndicator(prow int, selected bool) {
	var fg termbox.Attribute
	var bg termbox.Attribute
	if selected {
		fg = selIndicatorFg
		bg = selIndicatorBg
	} else {
		fg = defaultbg
		bg = defaultbg
	}
	termbox.SetCell(0, prow, ' ', fg, bg)
}

func uiToggleSelection() {
	if isSelected[vsel] {
		numSelected--
	} else {
		numSelected++
	}
	isSelected[vsel] = !isSelected[vsel]
	redrawSelectionIndicator(psel, isSelected[vsel])
}

func main() {
	exitCode := 0
	argLog := flag.String("l", "", "Logging output sink")
	argPrompt := flag.String("p", "", "Prompt string")
	flag.Parse()

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
	vsel = 0
	vtop = 0
	width, height = termbox.Size()
	if len(prompt) == 0 {
		ptop = 0
	} else {
		ptop = 1
	}
	psel = ptop
	pheight = height - ptop - 1 // last line is for instructions

	redrawAll(false)
mainloop:
	for {
		termbox.Flush()
		switch ev := termbox.PollEvent(); ev.Type {
		case termbox.EventKey: // TODO: uiFocusNext, uiFocusPrev
			if ev.Ch == 'j' {
				if vsel < len(choices)-1 {
					vsel++
					if psel == ptop+pheight-1 {
						vtop++
						redrawChoices()
					} else {
						psel++
						redrawChoice(psel-1, vsel-1, false)
						redrawChoice(psel, vsel, true)
					}
					redrawCurrentLineIndex()
				}
			} else if ev.Ch == 'k' {
				if vsel > 0 {
					vsel--
					if psel == ptop {
						vtop--
						redrawChoices()
					} else {
						psel--
						redrawChoice(psel+1, vsel+1, false)
						redrawChoice(psel, vsel, true)
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
			termbox.Sync() // resize internal buffer; see termbox.Size()
			width, height = ev.Width, ev.Height
			pheight = height - ptop - 1
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
		fmt.Printf("%s\n", choices[vsel])
	}
}

// BUG: Erroneous behavior when line count is to small to contain prompt, status and
//   at least one line of selection.
// TODO: -s: Single selection only
// TODO: Check whether attached to terminal (what for?)
// TODO: F1 Help
// TODO: Ctrl+u/d, PgUp/PgDown 50% scroll
// TODO: Input/Output delimiters
