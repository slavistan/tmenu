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
var w int             // term width
var h int             // term height
var selTop int        // table index of topmost choice in selection area
var selBottom int     // table index of last choice in selection area

func minInt(x, y int) int {
	if x <= y {
		return x
	}
	return y
}

func redrawChoice2(prow, vindex int, selected bool) {
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
	for ii := x; ii < w; ii++ {
		termbox.SetCell(x, prow, ' ', fg, bg)
		x++
	}
}

/* TODO: Refactor into using a range slice for better performance */
func redrawChoices() {
	for ii := 0; ii < minInt(len(choices)-vtop, pheight); ii++ {
		redrawChoice2(ptop+ii, vtop+ii, false)
	}
	redrawChoice2(psel, vsel, true)
}

func drawString(x, y int, s string, fg, bg termbox.Attribute) {
	for _, c := range s {
		termbox.SetCell(x, y, c, fg, bg)
		x += runewidth.RuneWidth(c)
	}
}

func clearToEndOfRow(x, y int) {
	for ; x < w; x++ {
		termbox.SetCell(x, y, ' ', defaultfg, defaultbg)
	}
}

func redrawCommandLine() {
	redrawCursorIndex()
	drawString(0, h-1, " \u21a9 Accept   \u2423 Select   \u241b Abort", defaultfg, defaultbg)
}

func redrawCursorIndex() {
	nStr := fmt.Sprintf("%d", len(choices))
	clearToEndOfRow(w-(len(nStr)*2+1), h-1)
	selnumStr := fmt.Sprintf("%d/%s", vsel+1, nStr)
	drawString(w-len(selnumStr), h-1, selnumStr, defaultfg, defaultbg)
}

func redrawPrompt() {
	drawString(0, 0, prompt, defaultfg, defaultbg)
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
	isSel := isSelected[vsel]
	if isSel {
		numSelected--
	} else {
		numSelected++
	}
	isSelected[vsel] = !isSel
	redrawSelectionIndicator(psel, isSelected[vsel])
	redrawCommandLine()
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
	isSelected = make([]bool, len(choices))
	w, h = termbox.Size()
	prompt = *argPrompt

	vsel = 0
	vtop = 0
	if len(*argPrompt) == 0 {
		ptop = 0
	} else {
		redrawPrompt()
		ptop = 1
	}
	psel = ptop
	pheight = h - ptop - 1 // last line is for instructions

	redrawChoices()
	redrawCommandLine()
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
						redrawChoice2(psel-1, vsel-1, false)
						redrawChoice2(psel, vsel, true)
					}
					redrawCursorIndex()
				}
			} else if ev.Ch == 'k' {
				if vsel > 0 {
					vsel--
					if psel == ptop {
						vtop--
						redrawChoices()
					} else {
						psel--
						redrawChoice2(psel+1, vsel+1, false)
						redrawChoice2(psel, vsel, true)
					}
					redrawCursorIndex()
				}
			} else if ev.Key == termbox.KeySpace {
				uiToggleSelection()
			} else if ev.Ch == 'q' || ev.Key == termbox.KeyEsc {
				exitCode = 1
				return
			} else if ev.Key == termbox.KeyEnter {
				break mainloop
			}
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

// TODO: Resize event
// TODO: -s: Single selection only
// TODO: Check whether attached to terminal
// TODO: Input/Output delimiters
