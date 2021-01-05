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

var prompt string
var choices []string // list of choices
var vsel int         // list index of current selection [0; num choices]
var psel int         // terminal row of where current selection is
var vtop int         // list index of topmost element in selection view
var ptop int         // terminal row of where topmost choice is
var pheight int      // number of rows which show selections
var promptfg termbox.Attribute
var promptbg termbox.Attribute
var defaultfg termbox.Attribute
var defaultbg termbox.Attribute
var selfg termbox.Attribute
var selbg termbox.Attribute

var w int         // term width
var h int         // term height
var selTop int    // table index of topmost choice in selection area
var selBottom int // table index of last choice in selection area

func minInt(x, y int) int {
	if x <= y {
		return x
	}
	return y
}

func drawChoice(prow int, s string, selected bool) {
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
	for _, c := range s {
		termbox.SetCell(x, prow, c, fg, bg)
		x += runewidth.RuneWidth(c)
	}

	/* clear rest of line */
	for ii := x; ii < w; ii++ {
		termbox.SetCell(x, prow, ' ', fg, bg)
		x++
	}
}

func redrawChoices() {
	for ii := 0; ii < minInt(len(choices)-vtop, pheight); ii++ {
		drawChoice(ptop+ii, choices[vtop+ii], false)
	}
	drawChoice(psel, choices[vsel], true)
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
	redrawSelectionIndex()
}

func redrawSelectionIndex() {
	nStr := fmt.Sprintf("%d", len(choices))
	clearToEndOfRow(w-(len(nStr)*2+1), h-1)
	selnumStr := fmt.Sprintf("%d/%s", vsel+1, nStr)
	drawString(w-len(selnumStr), h-1, selnumStr, defaultfg, defaultbg)
}

func redrawPrompt() {
	drawString(0, 0, prompt, defaultfg, defaultbg)
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
	w, h = termbox.Size()
	prompt = *argPrompt
	promptfg = termbox.ColorDefault
	promptbg = termbox.ColorDefault
	defaultfg = termbox.ColorDefault
	defaultbg = termbox.ColorDefault
	selfg = termbox.ColorBlack
	selbg = termbox.ColorWhite

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
		case termbox.EventKey:
			if ev.Ch == 'j' {
				if vsel < len(choices)-1 {
					vsel++
					if psel == ptop+pheight-1 {
						vtop++
						redrawChoices()
					} else {
						psel++
						drawChoice(psel-1, choices[vsel-1], false)
						drawChoice(psel, choices[vsel], true)
					}
					redrawSelectionIndex()
				}
			} else if ev.Ch == 'k' {
				if vsel > 0 {
					vsel--
					if psel == ptop {
						vtop--
						redrawChoices()
					} else {
						psel--
						drawChoice(psel+1, choices[vsel+1], false)
						drawChoice(psel, choices[vsel], true)
					}
					redrawSelectionIndex()
				}
			} else if ev.Ch == 'q' || ev.Key == termbox.KeyEsc {
				exitCode = 1
				return
			} else if ev.Key == termbox.KeyEnter {
				break mainloop
			}
		}
	}
	termbox.Close()
	fmt.Printf("%s\n", choices[vsel])
}

// TODO: Selection of multiple elements
// TODO: Wrapping of long lines
// TODO: -n: Omit trailing newline
// TODO: -m: Allow multiple selection
// TODO: Check whether attached to terminal
// TODO: Input/Output delimiters
