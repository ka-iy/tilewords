// SPDX-FileCopyrightText: 2026 Kartikeya IYER
// SPDX-License-Identifier: GPL-3.0-or-later

// Package ui is documented in doc.go.
package ui

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"tilewords/cpu"
	"tilewords/dictionary"
	"tilewords/engine"
)

// cpuTimeoutSecs is the maximum time the UI waits for the CPU to return a move
// before applying a pass on its behalf.
const cpuTimeoutSecs = 10

// doubleTapWindow is the maximum gap between two presses on the same staged board tile
// for them to register as a double-press, which returns the tile to the rack. The board
// cells deliberately do not implement fyne.DoubleTappable — on mobile that delays every
// single tap by tapDoubleDelay (500ms) — so the double-press is detected in the
// controller, keeping tile placement instant.
const doubleTapWindow = 350 * time.Millisecond

// pressFlashDuration is how long a control button shows its transient "pressed" colour
// after a tap. It gives a visible press cue on touch screens, where there is no hover
// highlight to confirm the tap landed.
const pressFlashDuration = 140 * time.Millisecond

// pressFlashImportance is the colour a control button flashes to when tapped. It is
// deliberately different from every button's resting importance, so the flash is visible
// on all of them.
const pressFlashImportance = widget.WarningImportance

// Labels for the control button that toggles the CPU rack, chosen by applyCPURackBtnLabel.
//
// Once the rack is face up the button hides it again in both arrangements, so it reads the
// same in both. While the rack is face down the button does different things in each — the
// narrow layout has to bring the whole block back, the wide layout only turns the tiles
// over — so each arrangement names it for what it will do.
const (
	// cpuRackBtnLabelShown is the label in either arrangement while the rack is face up.
	cpuRackBtnLabelShown = "Hide CPU"
	// cpuRackBtnLabelNarrow is the label in the narrow arrangement while the rack is hidden,
	// where pressing the button restores the block as well as the tile faces.
	cpuRackBtnLabelNarrow = "CPU Rack"
	// cpuRackBtnLabelWide is the label in the wide arrangement while the rack is face down,
	// where the block is already on screen and pressing the button only reveals the tiles.
	cpuRackBtnLabelWide = "Show CPU"
)

// stagedTile holds a tile taken from the human rack and tentatively placed on the
// board. fromRackIdx records the original rack slot so the tile can be recalled.
type stagedTile struct {
	Tile        engine.Tile
	Row, Col    int
	FromRackIdx int
}

// historyEntry is one executed turn shown in the move-history panel. cmd is the executed
// command for undo — nil for entries restored from a save, which are not undoable. points
// and cells are self-contained copies of the move's score contribution and placed cells, so
// the status summary and the CPU-word highlight can be derived without the command (which
// restored entries lack).
type historyEntry struct {
	cmd    engine.Command // nil for entries restored from a save (not undoable)
	player string         // "You" or "CPU"
	line   string
	points int      // score this move contributed (a play's score; 0 for pass/exchange)
	cells  [][2]int // board cells this move placed; nil for pass/exchange
	words  []string // words this play formed (main + cross), for the definitions panel; nil for pass/exchange
}

// gameScreen is the gameplay controller. It owns all mutable game state for one
// session plus the widgets that visualise it.
type gameScreen struct {
	app   *App
	state *engine.GameState
	dict  *dictionary.Dictionary
	rng   *rand.Rand

	// scrabbleNotation selects the move-history format: Scrabble coordinate notation
	// (e.g. "8D UNMIX +28") when true, otherwise the plain word list.
	scrabbleNotation bool

	// Interaction state.
	staged       []stagedTile
	rackSelected int          // selected human rack index; -1 = none
	exchangeMode bool         // selecting tiles to exchange
	exchangeSel  map[int]bool // rack indices chosen for exchange
	showCPURack  bool
	cpuThinking  bool
	blankOpen    bool // a blank-letter dialog is currently shown
	abandoned    bool // set when the player leaves this screen (stops stale CPU callbacks)
	gameOver     bool // set when the game has ended; the screen stays up for review

	statusMsg   string
	statusIsErr bool

	// lastHumanPts and lastCPUPts hold each player's most-recent-move points — a play's
	// score, or +0 for a pass or exchange — or -1 before that player has moved. The status
	// line shows them as "You <pts> / CPU <pts>" — the human's in green, the CPU's in amber —
	// whenever no transient message or CPU-thinking notice is being shown. Derived from the
	// move history by recomputeLastPoints so they stay correct across undo.
	lastHumanPts int
	lastCPUPts   int

	// Widgets.
	cells     [boardDim * boardDim]*cellWidget
	humanRack [engine.MaxRackSize]*rackSlotWidget
	cpuRack   [engine.MaxRackSize]*rackSlotWidget

	// boardBox and humanRackBoxC are the laid-out containers for the board grid and
	// the human rack; drag-and-drop hit-tests pointer positions against their geometry.
	boardBox      *fyne.Container
	humanRackBoxC *fyne.Container

	// cpuRackBox is the CPU rack block (its heading plus the rack row), shown or hidden as a
	// whole by applyCPURackVisibility. It is held here because the narrow layout leaves it out
	// entirely until the player asks for it, which is a change of visibility rather than of
	// the arrangement the widgets sit in.
	cpuRackBox *fyne.Container

	// boardLabels holds the board's row and column labels (A–O then 1–15). They are kept
	// here because their colour is baked in at construction: the game screen is refreshed
	// rather than rebuilt when the theme variant changes, so refresh must recolour them or
	// they keep the previous variant's colour and can end up unreadable.
	boardLabels []*canvas.Text

	rackLabel  *canvas.Text    // "Your rack" / red "GAME OVER" — green on the human's turn
	playIcon   *widget.Icon    // green play triangle beside the rack on the human's turn
	rackHeader *fyne.Container // wraps the rack label row; relaid out when the label text changes

	// Drag ghost: a floating tile that follows the cursor during a drag. It lives in a
	// no-layout overlay above the content (added in App.showGame) and is moved directly.
	ghost       *fyne.Container
	ghostBg     *canvas.Rectangle
	ghostLetter *canvas.Text
	ghostPoints *canvas.Text
	// Source of the in-progress drag, hidden in the rendering so the tile looks lifted:
	// dragRackSrc is the rack slot (-1 = none); dragBoardSrc is the board cell ({-1,-1}).
	dragRackSrc  int
	dragBoardSrc [2]int

	// pickedUp is the board cell of a staged tile tapped to move ({-1,-1} = none): the
	// next tap on an empty cell relocates it (tap-to-move).
	pickedUp [2]int

	// lastPressCell/lastPressAt track the previous press on a staged tile so a second
	// press on the same tile within doubleTapWindow registers as a double-press (which
	// returns the tile to the rack). Tracked here rather than via fyne.DoubleTappable so
	// single taps stay instant, and checked in both the tap and the touch drag-release
	// paths so the gesture works whether the OS reports a tap or a micro-drag.
	lastPressCell [2]int
	lastPressAt   time.Time

	// afterFunc schedules the flash-press revert (see flashPress). It defaults to
	// time.AfterFunc; tests override it to run the revert synchronously on the test
	// goroutine, since Fyne's test driver runs the revert's fyne.Do inline on the timer
	// goroutine rather than marshalling it, which would otherwise race the test's reads.
	afterFunc func(time.Duration, func()) *time.Timer

	youLabel   *widget.Label
	cpuLabel   *widget.Label
	bagLabel   *widget.Label
	moveLabel  *widget.Label
	levelLabel *widget.Label // CPU difficulty, shown in the top counters row
	statusRT   *widget.RichText

	// page is the phone arrangement's scrolling page, used to pan it for gestures another handler
	// cannot use. It is nil in the wide arrangement, whose panes scroll individually.
	page *phoneColumnScroll

	// gesture records which widget the current touch gesture began on, so a drag can be delivered
	// there even when the driver hands it elsewhere; see gestureOwner.
	gesture gestureOwner

	// Move-history log + undo stack (right of the board). Each entry is one executed
	// command; undo pops entries and reverses them, one turn per press.
	history       []historyEntry
	historyLabel  *widget.Label
	historyScroll *container.Scroll
	// openingLine is a fixed first line in the move history summarising the opening draw.
	// It is not a move, so it is not part of the undo stack and is never popped.
	openingLine string

	// Definitions panel (second tab beside the move history). Each word a move forms
	// is dispatched on defsWordCh to defsWorker, which looks its meaning up and appends
	// a formatted entry to defsEntries; defsLabel shows the entries separated by blank
	// lines. defsWordCh is nil when the definitions asset is not embedded in the build.
	defsLabel  *widget.Label
	defsScroll *container.Scroll
	// defsEntries holds the rendered definition entries shown in the Definitions tab, each
	// tagged with the history turn that produced it so an undone turn's entries can be
	// dropped. Without the tag the panel would keep describing words no longer on the board,
	// and list a word twice once it was replayed.
	defsEntries []defsEntry
	defsWordCh  chan defsRequest
	// defsClosed guards against closing defsWordCh more than once when the screen is left.
	defsClosed bool

	// cpuLastPlaced holds the board cells of the CPU's most recently played word; the
	// board outlines these tiles in red. Derived from history by recomputeCPUHighlight.
	cpuLastPlaced map[[2]int]bool

	playBtn    *touchButton
	exchBtn    *touchButton
	passBtn    *touchButton
	undoBtn    *touchButton
	saveBtn    *touchButton
	toggleBtn  *touchButton
	menuBtn    *touchButton
	shuffleBtn *touchButton
	recallBtn  *touchButton
}

// newGameScreen constructs the controller (no widgets yet; see build). The move-history
// format is taken from state.ScrabbleNotation.
func newGameScreen(a *App, state *engine.GameState, dict *dictionary.Dictionary) *gameScreen {
	gs := &gameScreen{
		app:              a,
		state:            state,
		dict:             dict,
		scrabbleNotation: state.ScrabbleNotation,
		openingLine:      openingDrawLine(state.OpeningDraw),
		history:          restoreHistory(state.History),
		rng:              newGameRNG(),
		rackSelected:     -1,
		exchangeSel:      make(map[int]bool),
		dragRackSrc:      -1,
		dragBoardSrc:     [2]int{-1, -1},
		pickedUp:         [2]int{-1, -1},
		lastPressCell:    [2]int{-1, -1},
		lastHumanPts:     -1,
		lastCPUPts:       -1,
	}
	// Derive the status summary and the CPU-word highlight from the (possibly restored)
	// history so a resumed game shows them immediately.
	gs.recomputeLastPoints()
	gs.recomputeCPUHighlight()
	return gs
}

// restoreHistory reconstructs the move-history entries from a saved game's records. The
// entries carry no command, so they are shown and their points/cells are usable but they
// are not undoable (undo is not restored across a save/load).
func restoreHistory(records []engine.MoveRecord) []historyEntry {
	if len(records) == 0 {
		return nil
	}
	h := make([]historyEntry, len(records))
	for i, r := range records {
		h[i] = historyEntry{player: r.Player, line: r.Line, points: r.Points, cells: r.Cells, words: r.Words}
	}
	return h
}

// moveRecords converts the live move history into the serialisable form persisted with the
// game.
func (gs *gameScreen) moveRecords() []engine.MoveRecord {
	if len(gs.history) == 0 {
		return nil
	}
	recs := make([]engine.MoveRecord, len(gs.history))
	for i, e := range gs.history {
		recs[i] = engine.MoveRecord{Player: e.player, Line: e.line, Points: e.points, Cells: e.cells, Words: e.words}
	}
	return recs
}

// build constructs every widget, assembles the layout and returns the content.
func (gs *gameScreen) build() fyne.CanvasObject {
	// Board.
	boardObjs := make([]fyne.CanvasObject, 0, boardDim*boardDim)
	for row := 0; row < boardDim; row++ {
		for col := 0; col < boardDim; col++ {
			sq := gs.state.Board.Cell(row, col).Square
			c := newCellWidget(row, col, sq, gs.onBoardTap)
			c.onDrag = gs.onBoardDrag
			c.onDragEnd = gs.onBoardDragEnd
			c.gesture = &gs.gesture
			gs.cells[row*boardDim+col] = c
			boardObjs = append(boardObjs, c)
		}
	}
	// Row/column labels follow the cells; boardLayout positions them in the gutters
	// reserved along the top (A–O) and left (1–15) edges.
	colLabels, rowLabels := newBoardLabels()
	gs.boardLabels = append(append(gs.boardLabels[:0], colLabels...), rowLabels...)
	for _, l := range gs.boardLabels {
		boardObjs = append(boardObjs, l)
	}
	board := container.New(boardLayout{}, boardObjs...)
	gs.boardBox = board

	// Racks.
	humanObjs := make([]fyne.CanvasObject, engine.MaxRackSize)
	for i := 0; i < engine.MaxRackSize; i++ {
		s := newRackSlotWidget(i, gs.onRackTap)
		s.onDrag = gs.onRackDrag
		s.onDragEnd = gs.onRackDragEnd
		s.gesture = &gs.gesture
		gs.humanRack[i] = s
		humanObjs[i] = s
	}
	humanRack := container.New(rackLayout{}, humanObjs...)
	gs.humanRackBoxC = humanRack

	cpuObjs := make([]fyne.CanvasObject, engine.MaxRackSize)
	for i := 0; i < engine.MaxRackSize; i++ {
		s := newRackSlotWidget(i, nil) // CPU rack is not interactive
		// Nothing here can be dragged, so a drag on this row goes to the page instead of dying.
		s.onUnusableDrag = gs.panPage
		gs.cpuRack[i] = s
		cpuObjs[i] = s
	}
	cpuRack := container.New(rackLayout{}, cpuObjs...)

	// Score/status bar.
	gs.youLabel = widget.NewLabel("")
	gs.cpuLabel = widget.NewLabel("")
	gs.bagLabel = widget.NewLabel("")
	gs.moveLabel = widget.NewLabel("")
	gs.levelLabel = widget.NewLabel(fmt.Sprintf("CPU Lv: %d", gs.state.CPULevel))
	gs.statusRT = widget.NewRichTextWithText("")
	gs.statusRT.Wrapping = fyne.TextWrapWord

	// The score counters show as one centred row, wrapping to two rows (the two scores,
	// then the bag/move/level counters) only when the width cannot fit the single row.
	counters := newStatusCounters(gs.youLabel, gs.cpuLabel, gs.bagLabel, gs.moveLabel, gs.levelLabel)
	statusItems := make([]fyne.CanvasObject, 0, 3)
	// statusGaps[i] is the vertical gap between status row i and row i+1 (see
	// tightColumnLayout). The word-list/counters gap is tight (statusRowGap); the gap above
	// the current-move line is looser (statusMoveGap) so it sits centred between the
	// counters and the rack.
	statusGaps := make([]float32, 0, 2)
	// The word list is fixed for the whole game; show it above the score counters.
	if gs.dict != nil {
		wordList := widget.NewLabelWithStyle("Word list: "+dictShortName(gs.dict.Name()),
			fyne.TextAlignCenter, fyne.TextStyle{Italic: true})
		wordList.Wrapping = fyne.TextWrapWord
		statusItems = append(statusItems, wordList)
		statusGaps = append(statusGaps, statusRowGap) // word list -> counters
	}
	statusItems = append(statusItems, counters, gs.statusRT)
	statusGaps = append(statusGaps, statusMoveGap) // counters -> current-move line
	// A tight column (rather than a plain VBox) closes most of the vertical gap between
	// the status rows, per statusGaps.
	statusBar := container.New(tightColumnLayout{gaps: statusGaps}, statusItems...)

	// Control buttons (shared across layouts; arranged differently per layout). Each
	// flashes briefly when tapped (see newControlButton / flashPress).
	gs.playBtn = gs.newControlButton("Play", gs.commitPlay)
	gs.playBtn.Importance = widget.HighImportance
	gs.exchBtn = gs.newControlButton("Exchange", gs.onExchange)
	gs.passBtn = gs.newControlButton("Pass", gs.commitPass)
	gs.undoBtn = gs.newControlButton("Undo", gs.doUndo)
	gs.saveBtn = gs.newControlButton("Save", gs.doSave)
	// The arrangement decides the final label (see applyCPURackBtnLabel); this is the
	// starting text, replaced as soon as an arrangement is built.
	gs.toggleBtn = gs.newControlButton(cpuRackBtnLabelNarrow, gs.toggleCPURack)
	gs.menuBtn = gs.newControlButton("Menu", gs.goMainMenu)
	buttons := []fyne.CanvasObject{
		gs.playBtn, gs.exchBtn, gs.passBtn, gs.undoBtn, gs.saveBtn, gs.toggleBtn, gs.menuBtn,
	}

	// Rack header: a green play icon (shown on the human's turn) and the "Your rack"
	// label (green on the human's turn), with the shuffle and recall buttons grouped
	// immediately to its right. refresh() drives the turn cue.
	gs.playIcon = widget.NewIcon(blankIconResource)
	gs.rackLabel = canvas.NewText("Your rack", bodyTextColor())
	gs.rackLabel.TextStyle = fyne.TextStyle{Bold: true}
	gs.rackLabel.TextSize = theme.TextSize()
	gs.shuffleBtn = gs.newControlIconButton(shuffleIconResource, gs.doShuffle)
	gs.shuffleBtn.Importance = widget.LowImportance
	gs.recallBtn = gs.newControlIconButton(recallIconResource, gs.doRecallAll)
	gs.recallBtn.Importance = widget.LowImportance
	rackHeaderRow := container.NewHBox(
		container.NewCenter(gs.playIcon),
		container.NewCenter(gs.rackLabel),
		gs.shuffleBtn,
		gs.recallBtn,
	)
	gs.rackHeader = container.NewCenter(rackHeaderRow)
	humanRackBox := container.NewVBox(gs.rackHeader, humanRack)
	cpuRackBox := container.NewVBox(
		widget.NewLabelWithStyle(fmt.Sprintf("CPU rack - Lv %d", gs.state.CPULevel), fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		cpuRack,
	)
	gs.cpuRackBox = cpuRackBox

	// Right-hand panel: two tabs sharing the space beside the board. Both are
	// non-editable, scrollable, and selectable so their text can be copied.
	//   - "Move history" (shown first): a log of each turn — player, word(s), score.
	//   - "Definitions": the meaning of every word played, one entry per word.
	// On touch a finger drag scrolls the panel rather than selecting text, so the whole
	// panel is copied via the tab bar's Copy button or a long press on the panel itself
	// (see dragscroll.go); both report through onCopied.
	onCopied := func() {
		gs.setStatus("Copied to clipboard.", false)
		gs.refresh()
	}

	gs.historyLabel = widget.NewLabel("")
	gs.historyLabel.Wrapping = fyne.TextWrapWord
	gs.historyLabel.Selectable = true
	// A fixed-width font lines the coordinates, words and scores up column-wise down the log,
	// so successive entries can be compared by eye instead of read one at a time.
	gs.historyLabel.TextStyle = fyne.TextStyle{Monospace: true}
	gs.historyScroll = container.NewVScroll(gs.historyLabel)
	historyText := func() string { return gs.historyLabel.Text }
	// Touch: drag pans, long press copies. A pane too short to scroll hands the drag to the page.
	enableTouchScroll(gs.historyScroll, historyText, onCopied, gs.panPage)
	gs.refreshHistory() // show the opening-draw line before any move is made

	gs.defsLabel = widget.NewLabel("")
	gs.defsLabel.Wrapping = fyne.TextWrapWord
	gs.defsLabel.Selectable = true
	// The same fixed-width font as the move history: the two tabs share one panel, so a
	// switch between them must not change the look of the text.
	gs.defsLabel.TextStyle = fyne.TextStyle{Monospace: true}
	gs.defsScroll = container.NewVScroll(gs.defsLabel)
	defsText := func() string { return gs.defsLabel.Text }
	enableTouchScroll(gs.defsScroll, defsText, onCopied, gs.panPage)
	// The definitions lookup worker is started from App.showGame (not here), so tests
	// that build a screen directly do not spawn it or load the definitions asset.

	// Each panel is wrapped so it returns to its newest entry when the viewport changes
	// shape, which re-wraps the text under it; see endFollowLayout.
	historyBox := newTabPanel(
		onCopied,
		tabItem{
			title:    "Move history",
			content:  container.New(&endFollowLayout{}, gs.historyScroll),
			copyText: historyText,
		},
		tabItem{
			title:    "Definitions",
			content:  container.New(&endFollowLayout{}, gs.defsScroll),
			copyText: defsText,
		},
	).root

	// Drag ghost: a floating tile that follows the cursor; hidden until a drag begins.
	// App.showGame stacks it in a no-layout overlay above this content.
	gs.ghostBg, gs.ghostLetter, gs.ghostPoints = newTileObjects()
	gs.ghost = container.New(tileFillLayout{}, gs.ghostBg, gs.ghostLetter, gs.ghostPoints)
	gs.ghost.Hide()

	gs.refresh()

	// Two arrangements of the same widgets, chosen by available size:
	//   wide   — a draggable split with the move-history panel filling the full screen
	//            height on the right; the left pane holds the board centred above the
	//            status/rack/control stack, with the action buttons stretched across the
	//            full width of the left pane.
	//   narrow — one scrollable column (board, racks, controls, history) for phone
	//            portrait, so nothing is clipped and the page scrolls if needed.
	buildArrangement := func(narrow bool) fyne.CanvasObject {
		if narrow {
			// Three buttons per row keeps each one narrower than a half-width two-column
			// grid; the seventh (Menu) sits alone on the last row. The grid is grouped so a
			// press near a button's top edge runs that button rather than the one above it or
			// nothing at all; see touchGroup.
			controls := newTouchGroup(container.NewGridWithColumns(3, buttons...), buttons...)
			// The board fills the column width (scaling up from its tappable minimum); the
			// rest stack below at their natural heights. The history/definitions pane
			// stretches to fill any spare viewport height (down to portraitHistoryMinH), and
			// the page scrolls only when the column is taller than the viewport.
			histWrap := container.New(minHeightLayout{minH: portraitHistoryMinH}, historyBox)
			// The status block and the rack are grouped with a tightened (statusRackGap) gap
			// so the rack sits closer to the current-move line than the other sections.
			statusAndRack := container.New(tightColumnLayout{gaps: []float32{statusRackGap}}, statusBar, humanRackBox)
			column := container.New(
				phoneColumnLayout{board: board, minBoard: minBoardPx},
				board,
				statusAndRack,
				controls,
				cpuRackBox,
				histWrap,
			)
			gs.page = newPhoneColumnScroll(column, board, histWrap, &gs.gesture)
			// Set after gs.page, which is what both of these read to tell the arrangements
			// apart.
			gs.applyCPURackVisibility()
			gs.applyCPURackBtnLabel()
			return gs.page
		}
		// The action buttons stretch across the full width of the left pane: the grid
		// sits in the full-width bottom block, so each button is a 1/len(buttons) slice
		// of the pane width.
		// The wide arrangement has no page-level scroll: the split's panes scroll individually.
		gs.page = nil
		// The wide layout always shows the CPU rack, including when the window was narrow with
		// it hidden — the two arrangements share the widget, as they do the toggle button.
		gs.applyCPURackVisibility()
		gs.applyCPURackBtnLabel()
		controls := newTouchGroup(container.NewGridWithColumns(len(buttons), buttons...), buttons...)
		// The board fills the left pane above the status/rack/control stack; boardLayout
		// centres the board square within that space.
		bottom := container.NewVBox(statusBar, humanRackBox, controls, cpuRackBox)
		left := container.NewBorder(nil, bottom, nil, nil, board)
		// The history panel is the split's right pane, so it runs the full screen height.
		split := container.NewHSplit(left, historyBox)
		split.SetOffset(0.72)
		return split
	}

	return newResponsiveContainer(buildArrangement)
}

// ---------- Refresh ----------

// refresh pushes the current game state into every widget and updates the
// enabled state of the control buttons. It must run on the UI goroutine.
func (gs *gameScreen) refresh() {
	// Board: committed tiles, then staged tiles override empty cells.
	// Drop a stale tap-to-move pickup whose tile is no longer staged (recalled, moved,
	// committed, or undone).
	if gs.pickedUp != [2]int{-1, -1} {
		if _, ok := gs.stagedAt(gs.pickedUp[0], gs.pickedUp[1]); !ok {
			gs.pickedUp = [2]int{-1, -1}
		}
	}

	stagedAt := make(map[[2]int]engine.Tile, len(gs.staged))
	for _, st := range gs.staged {
		stagedAt[[2]int{st.Row, st.Col}] = st.Tile
	}
	for row := 0; row < boardDim; row++ {
		for col := 0; col < boardDim; col++ {
			cell := gs.cells[row*boardDim+col]
			committed := gs.state.Board.Cell(row, col).Tile
			if committed != nil {
				t := *committed
				cell.setContent(&t, false, gs.cpuLastPlaced[[2]int{row, col}], false)
				continue
			}
			if t, ok := stagedAt[[2]int{row, col}]; ok && [2]int{row, col} != gs.dragBoardSrc {
				st := t
				cell.setContent(&st, true, false, [2]int{row, col} == gs.pickedUp)
				continue
			}
			cell.setContent(nil, false, false, false)
		}
	}

	// Human rack.
	stagedOut := make(map[int]bool, len(gs.staged))
	for _, st := range gs.staged {
		stagedOut[st.FromRackIdx] = true
	}
	humanTiles := gs.state.HumanRack.Tiles()
	for i := 0; i < engine.MaxRackSize; i++ {
		slot := gs.humanRack[i]
		if i >= len(humanTiles) || stagedOut[i] || i == gs.dragRackSrc {
			slot.setContent(nil, false, false, false)
			continue
		}
		t := humanTiles[i]
		selected := !gs.exchangeMode && i == gs.rackSelected
		slot.setContent(&t, false, selected, gs.exchangeSel[i])
	}

	// CPU rack (face-down unless revealed).
	cpuTiles := gs.state.CPURack.Tiles()
	for i := 0; i < engine.MaxRackSize; i++ {
		slot := gs.cpuRack[i]
		if i >= len(cpuTiles) {
			slot.setContent(nil, false, false, false)
			continue
		}
		t := cpuTiles[i]
		slot.setContent(&t, !gs.showCPURack, false, false)
	}

	// Scores / counters.
	youMark, cpuMark := "", ""
	if gs.state.CurrentTurn == engine.HumanTurn {
		youMark = "▶ "
	} else {
		cpuMark = "▶ "
	}
	gs.youLabel.SetText(fmt.Sprintf("%sYou: %d", youMark, gs.state.HumanScore))
	gs.cpuLabel.SetText(fmt.Sprintf("%sCPU: %d", cpuMark, gs.state.CPUScore))

	// Board labels follow the current theme variant. Only the ones whose colour actually
	// changed are refreshed, so an ordinary refresh (every tap) does no work here.
	if col := bodyTextColor(); len(gs.boardLabels) > 0 && gs.boardLabels[0].Color != col {
		for _, l := range gs.boardLabels {
			l.Color = col
			l.Refresh()
		}
	}

	// Rack label: red "GAME OVER" when the game has ended; otherwise "Your rack",
	// green with a play icon on the human's turn and the normal colour otherwise. The
	// header is relaid out only when the text (hence width) changes.
	if gs.rackLabel != nil {
		text, col := "Your rack", bodyTextColor()
		icon := blankIconResource
		switch {
		case gs.gameOver:
			text, col = "GAME OVER", gameOverColor()
		case gs.state.CurrentTurn == engine.HumanTurn && !gs.cpuThinking:
			col = turnCueColor()
			icon = playIconResource
		}
		relayout := gs.rackLabel.Text != text
		gs.rackLabel.Text = text
		gs.rackLabel.Color = col
		gs.playIcon.SetResource(icon)
		gs.rackLabel.Refresh()
		if relayout && gs.rackHeader != nil {
			gs.rackHeader.Refresh()
		}
	}
	gs.bagLabel.SetText(fmt.Sprintf("Bag: %d", gs.state.Bag.Count()))
	gs.moveLabel.SetText(fmt.Sprintf("Move: %d", gs.state.MoveNumber))

	// Status line.
	gs.statusRT.Segments = gs.statusSegments()
	gs.statusRT.Refresh()

	gs.syncButtons()
}

// syncButtons enables/disables the control buttons based on the current state.
func (gs *gameScreen) syncButtons() {
	// While the blank-letter dialog is open, no action button may fire — a staged
	// blank has no assigned letter yet, so committing now would form an invalid word.
	humanTurn := gs.state.CurrentTurn == engine.HumanTurn && !gs.cpuThinking && !gs.blankOpen && !gs.gameOver
	hasStaged := len(gs.staged) > 0

	setEnabled(gs.playBtn, humanTurn && hasStaged)
	setEnabled(gs.exchBtn, humanTurn && !hasStaged)
	setEnabled(gs.passBtn, humanTurn && !hasStaged && !gs.exchangeMode)
	setEnabled(gs.undoBtn, humanTurn && gs.canUndo() && !hasStaged && !gs.exchangeMode)
	setEnabled(gs.saveBtn, humanTurn)
	setEnabled(gs.toggleBtn, true)
	setEnabled(gs.menuBtn, true)
	setEnabled(gs.shuffleBtn, humanTurn && !gs.exchangeMode)
	setEnabled(gs.recallBtn, humanTurn && hasStaged)

	switch {
	case gs.exchangeMode && len(gs.exchangeSel) > 0:
		gs.exchBtn.SetText("Confirm")
	case gs.exchangeMode:
		gs.exchBtn.SetText("Cancel")
	default:
		gs.exchBtn.SetText("Exchange")
	}
}

// setEnabled toggles a button's enabled state.
func setEnabled(b *touchButton, enabled bool) {
	if enabled {
		b.Enable()
	} else {
		b.Disable()
	}
}

// newControlButton creates a text control button whose tap first flashes the button (a
// momentary colour change — see flashPress) and then runs tapped.
func (gs *gameScreen) newControlButton(label string, tapped func()) *touchButton {
	var b *touchButton
	b = newTouchButton(label, func() {
		gs.flashPress(b)
		tapped()
	})
	return b
}

// newControlIconButton is newControlButton for an icon-only control button.
func (gs *gameScreen) newControlIconButton(icon fyne.Resource, tapped func()) *touchButton {
	var b *touchButton
	b = newTouchButtonWithIcon("", icon, func() {
		gs.flashPress(b)
		tapped()
	})
	return b
}

// flashPress briefly switches b to the pressed-flash colour, then schedules a revert to
// its resting importance after pressFlashDuration. Importance is mutated only on the UI
// goroutine (the tap handler and the fyne.Do revert), so no lock is needed. A re-tap
// while a flash is in progress is ignored for the flash (its pending revert restores the
// resting colour) but still runs the button's action.
func (gs *gameScreen) flashPress(b *touchButton) {
	base := b.Importance
	if base == pressFlashImportance {
		return
	}
	b.Importance = pressFlashImportance
	b.Refresh()
	after := gs.afterFunc
	if after == nil {
		after = time.AfterFunc
	}
	after(pressFlashDuration, func() {
		fyne.Do(func() {
			b.Importance = base
			b.Refresh()
		})
	})
}

// ---------- Tile input ----------

// registerStagedPress records a press on the staged tile at (row,col) and reports
// whether it completes a double-press: a second press on the same staged cell within
// doubleTapWindow. It is called from both the tap path (onBoardTap) and the touch
// drag-release path (onBoardDragEnd), so a double-press is recognised whether the OS
// classifies the presses as taps or micro-drags.
func (gs *gameScreen) registerStagedPress(row, col int) (doublePress bool) {
	if gs.lastPressCell == [2]int{row, col} && time.Since(gs.lastPressAt) < doubleTapWindow {
		gs.lastPressCell = [2]int{-1, -1}
		return true
	}
	gs.lastPressCell = [2]int{row, col}
	gs.lastPressAt = time.Now()
	return false
}

func (gs *gameScreen) onBoardTap(row, col int) {
	if !gs.humanInputAllowed() {
		return
	}
	gs.dragRackSrc = -1 // a tap is not a rack drag; drop any leaked rack-drag source
	if !gs.state.Board.IsEmpty(row, col) {
		return // occupied by a committed tile
	}
	// A double-press on a staged tile returns it to the rack. Otherwise a single press
	// picks the tile up to move (tap-to-move), or puts a held tile back down.
	if _, ok := gs.stagedAt(row, col); ok {
		if gs.registerStagedPress(row, col) {
			gs.onBoardDoubleTap(row, col) // double-press → recall to the rack
			return
		}
		if gs.pickedUp == [2]int{row, col} {
			gs.pickedUp = [2]int{-1, -1}
			gs.refresh() // tapping the held tile again puts it back down
			return
		}
		gs.pickedUp = [2]int{row, col}
		gs.rackSelected = -1
		gs.refresh()
		return
	}
	// Empty cell. If a staged tile is picked up, move it here.
	if gs.pickedUp != [2]int{-1, -1} {
		src := gs.pickedUp
		gs.pickedUp = [2]int{-1, -1}
		if st, ok := gs.stagedAt(src[0], src[1]); ok {
			gs.moveStagedTile(st, row, col)
		} else {
			gs.refresh()
		}
		return
	}
	// Otherwise place the selected rack tile here.
	if gs.rackSelected >= 0 {
		gs.stageTile(gs.rackSelected, row, col)
	}
}

func (gs *gameScreen) onRackTap(idx int) {
	if !gs.humanInputAllowed() {
		return
	}
	gs.clearDragState() // a tap is not a drag; drop any leaked drag source
	tiles := gs.state.HumanRack.Tiles()
	for _, st := range gs.staged {
		if st.FromRackIdx == idx {
			return // this slot is staged on the board
		}
	}
	if idx >= len(tiles) {
		return
	}

	if gs.exchangeMode {
		if gs.exchangeSel[idx] {
			delete(gs.exchangeSel, idx)
		} else {
			gs.exchangeSel[idx] = true
		}
		gs.refresh()
		return
	}

	gs.pickedUp = [2]int{-1, -1} // selecting a rack tile cancels a board tap-to-move
	if gs.rackSelected == idx {
		gs.rackSelected = -1
	} else {
		gs.rackSelected = idx
	}
	gs.refresh()
}

// humanInputAllowed reports whether board/rack taps should be processed.
func (gs *gameScreen) humanInputAllowed() bool {
	return !gs.abandoned && !gs.cpuThinking && !gs.blankOpen && !gs.gameOver &&
		gs.state.CurrentTurn == engine.HumanTurn
}

// stageTile moves the tile at rackIdx onto board cell (row, col). If it is an
// unassigned blank, the blank-letter picker is shown.
func (gs *gameScreen) stageTile(rackIdx, row, col int) {
	tiles := gs.state.HumanRack.Tiles()
	if rackIdx < 0 || rackIdx >= len(tiles) {
		return
	}
	if gs.isStagedSlot(rackIdx) {
		// This rack slot is already staged — staging it again would duplicate the
		// FromRackIdx and leave a phantom staged entry keyed to the same slot.
		return
	}
	t := tiles[rackIdx]
	gs.staged = append(gs.staged, stagedTile{Tile: t, Row: row, Col: col, FromRackIdx: rackIdx})
	gs.rackSelected = -1
	gs.refresh()

	if t.IsBlank && t.AssignedLetter == 0 {
		gs.promptBlank(rackIdx)
	}
}

// recallStagedTile removes the staged tile that came from fromRackIdx.
func (gs *gameScreen) recallStagedTile(fromRackIdx int) {
	for i, st := range gs.staged {
		if st.FromRackIdx == fromRackIdx {
			gs.staged = append(gs.staged[:i], gs.staged[i+1:]...)
			break
		}
	}
}

// recallAll returns every staged tile to the rack and cancels any exchange.
func (gs *gameScreen) recallAll() {
	gs.staged = gs.staged[:0]
	gs.rackSelected = -1
	gs.exchangeMode = false
	gs.exchangeSel = make(map[int]bool)
	gs.pickedUp = [2]int{-1, -1}
	gs.clearDragState()
}

// clearDragState resets the transient drag sources. A touch drag that never delivered
// its DragEnd (interrupted gesture) would otherwise leave dragRackSrc/dragBoardSrc
// pointing at a slot, which refresh() renders as a permanent phantom empty slot —
// making the rack appear to be missing a tile. Clearing it at turn boundaries and on
// any tap guarantees a leaked drag source cannot outlive the gesture.
//
// The widgets' own drag tracking is reset too. On touch a DragEvent carries no absolute
// position, so the pointer is followed by accumulating deltas from where the gesture began; a
// widget left believing it is mid-drag would seed the next gesture from the abandoned one's
// end point, drawing the ghost away from the finger and hit-testing the drop to the wrong
// cell — or to none, which recalls the tile.
func (gs *gameScreen) clearDragState() {
	gs.dragRackSrc = -1
	gs.dragBoardSrc = [2]int{-1, -1}
	for _, c := range gs.cells {
		if c != nil {
			c.cancelDrag()
		}
	}
	for _, s := range gs.humanRack {
		if s != nil {
			s.cancelDrag()
		}
	}
}

// assignBlank sets the chosen letter on the staged blank from rack slot fromRackIdx.
func (gs *gameScreen) assignBlank(fromRackIdx int, letter byte) {
	for i := range gs.staged {
		if gs.staged[i].FromRackIdx == fromRackIdx {
			gs.staged[i].Tile.AssignedLetter = letter
			gs.staged[i].Tile.Letter = letter // used by GADDAG traversal
			return
		}
	}
}

// stagedBlankUnassigned reports whether the staged tile from fromRackIdx is a
// blank that still has no assigned letter.
func (gs *gameScreen) stagedBlankUnassigned(fromRackIdx int) bool {
	for _, st := range gs.staged {
		if st.FromRackIdx == fromRackIdx {
			return st.Tile.IsBlank && st.Tile.AssignedLetter == 0
		}
	}
	return false
}

// ---------- Actions ----------

func (gs *gameScreen) commitPlay() {
	if len(gs.staged) == 0 {
		gs.setStatus("Place at least one tile before playing.", true)
		gs.refresh()
		return
	}
	placed := make([]engine.PlacedTile, len(gs.staged))
	for i, st := range gs.staged {
		placed[i] = engine.PlacedTile{Tile: st.Tile, Row: st.Row, Col: st.Col}
	}
	cmd := &engine.PlayCommand{Move: engine.PlayMove{Placed: placed}}
	if err := cmd.Execute(gs.state, gs.dict, gs.rng); err != nil {
		gs.setStatus(sanitiseError(err), true)
		gs.refresh()
		return
	}
	gs.staged = gs.staged[:0]
	gs.rackSelected = -1
	gs.clearDragState()
	// logCommand records the points and clears the transient message so the status line
	// shows the score summary (your points in green / the CPU's in amber).
	gs.logCommand("You", cmd)
	gs.afterHumanMove()
}

func (gs *gameScreen) onExchange() {
	if !gs.exchangeMode {
		gs.exchangeMode = true
		gs.rackSelected = -1
		gs.exchangeSel = make(map[int]bool)
		gs.setStatus("Tap tiles to select, then Confirm Exchange.", false)
		gs.refresh()
		return
	}
	if len(gs.exchangeSel) > 0 {
		gs.commitExchange()
		return
	}
	gs.exchangeMode = false
	gs.setStatus("Exchange cancelled.", false)
	gs.refresh()
}

func (gs *gameScreen) commitExchange() {
	tiles := gs.state.HumanRack.Tiles()
	var toExchange []engine.Tile
	for idx := range gs.exchangeSel {
		if idx < len(tiles) {
			toExchange = append(toExchange, tiles[idx])
		}
	}
	if len(toExchange) == 0 {
		gs.setStatus("Select tiles on your rack to exchange.", true)
		gs.refresh()
		return
	}
	cmd := &engine.ExchangeCommand{Move: engine.ExchangeMove{Tiles: toExchange}}
	if err := cmd.Execute(gs.state, gs.dict, gs.rng); err != nil {
		gs.setStatus(sanitiseError(err), true)
		gs.refresh()
		return
	}
	gs.exchangeMode = false
	gs.rackSelected = -1
	gs.exchangeSel = make(map[int]bool)
	gs.clearDragState()
	// logCommand records the move (+0) and clears the transient message so the status line
	// shows the score summary. The exchange is also recorded in the move history.
	gs.logCommand("You", cmd)
	gs.afterHumanMove()
}

func (gs *gameScreen) commitPass() {
	gs.recallAll()
	cmd := &engine.PassCommand{}
	if err := cmd.Execute(gs.state, gs.dict, gs.rng); err != nil {
		gs.setStatus(sanitiseError(err), true)
		gs.refresh()
		return
	}
	// logCommand records the move (+0) and clears the transient message so the status line
	// shows the score summary. The pass is also recorded in the move history.
	gs.logCommand("You", cmd)
	gs.afterHumanMove()
}

func (gs *gameScreen) doUndo() {
	if !gs.canUndo() {
		gs.setStatus("Nothing to undo.", true)
		gs.refresh()
		return
	}
	// Undo one of the player's turns: reverse the trailing commands (the CPU's reply,
	// if any, then the human's move) so the state lands back on the human's turn.
	// Pressing Undo again steps back another turn.
	for len(gs.history) > 0 {
		e := gs.history[len(gs.history)-1]
		if e.cmd == nil {
			break // reached a restored entry (no command); those are not undoable
		}
		// gs.rng reshuffles the bag as part of the undo, so replaying the turn cannot draw
		// the tiles this move already revealed.
		e.cmd.Undo(gs.state, gs.rng)
		gs.history = gs.history[:len(gs.history)-1]
		if e.player == "You" {
			break
		}
	}
	gs.recomputeCPUHighlight()
	gs.recomputeLastPoints()
	gs.refreshHistory()
	gs.dropUndoneDefinitions()
	gs.recallAll()
	gs.setStatus("Move undone.", false)
	gs.refresh()
}

func (gs *gameScreen) doSave() {
	// Persist the move-history log alongside the game state so it is restored on load.
	gs.state.History = gs.moveRecords()
	if err := gs.app.sm.Save(gs.state); err != nil {
		gs.setStatus(sanitiseError(err), true)
		gs.refresh()
		return
	}
	gs.setStatus("Game saved.", false)
	gs.refresh()
}

func (gs *gameScreen) toggleCPURack() {
	gs.showCPURack = !gs.showCPURack
	gs.applyCPURackVisibility()
	gs.applyCPURackBtnLabel()
	// The narrow column has just gained or lost the rack's height, so the history pane's
	// share of the viewport has to be recomputed; nothing resizes the page here to do it.
	if gs.page != nil {
		gs.page.refit()
	}
	gs.refresh()
}

// applyCPURackVisibility shows or hides the CPU rack block for the arrangement now in use.
//
//   - Narrow (the stacked phone column): the block is present only once the player has asked
//     for it, so the move-history/definitions pane starts directly under the action buttons
//     and keeps the height the rack would otherwise have taken.
//   - Wide: the block is always present. The left pane has room for it below the controls,
//     and the history pane is the split's other side, so hiding it would buy that pane
//     nothing.
//
// Both arrangements re-parent the same widgets, so a block hidden in one would stay hidden
// in the other. This runs whenever the arrangement is built, which is what keeps the wide
// layout's rack from disappearing because it was hidden while the window was narrow.
func (gs *gameScreen) applyCPURackVisibility() {
	if gs.cpuRackBox == nil {
		return
	}
	if gs.page == nil || gs.showCPURack {
		gs.cpuRackBox.Show()
		return
	}
	gs.cpuRackBox.Hide()
}

// applyCPURackBtnLabel names the CPU rack toggle for what pressing it would do now: hide a
// rack that is face up, or reveal one that is not — which differs by arrangement (see the
// cpuRackBtnLabel constants).
//
// The label is derived from the rack's face-up state and the arrangement, and each of those
// changes in exactly one place: toggleCPURack and buildArrangement respectively. Both call
// this, which is what keeps the label correct without recomputing it on every refresh.
// buildArrangement has to because the two arrangements share the one button and name it
// differently, so a window crossing the wide/narrow threshold would otherwise keep the name
// the arrangement it was built in gave it.
func (gs *gameScreen) applyCPURackBtnLabel() {
	if gs.toggleBtn == nil {
		return
	}
	label := cpuRackBtnLabelShown
	if !gs.showCPURack {
		// gs.page is set only by the narrow arrangement, which is what tells the two apart.
		label = cpuRackBtnLabelWide
		if gs.page != nil {
			label = cpuRackBtnLabelNarrow
		}
	}
	gs.toggleBtn.SetText(label)
}

func (gs *gameScreen) goMainMenu() {
	// showMainMenu tears this screen down (marking it abandoned so any in-flight CPU callback
	// is ignored, and stopping the definitions worker) as part of leaving it.
	gs.app.showMainMenu("")
}

// ---------- Rack tools & drag-and-drop ----------

// doShuffle recalls any staged tiles, then randomises the rack order.
func (gs *gameScreen) doShuffle() {
	if !gs.humanInputAllowed() {
		return
	}
	gs.recallAll()
	gs.state.HumanRack.Shuffle(gs.rng)
	gs.setStatus("Rack shuffled.", false)
	gs.refresh()
}

// doRecallAll returns every staged tile to the rack.
func (gs *gameScreen) doRecallAll() {
	if !gs.humanInputAllowed() || len(gs.staged) == 0 {
		return
	}
	gs.recallAll()
	gs.setStatus("Tiles recalled to your rack.", false)
	gs.refresh()
}

// onRackDrag lifts the dragged tile (hiding it in the rack) and drives the ghost that
// follows the cursor. It fires repeatedly during a drag. As the pointer passes over other
// rack slots the intervening tiles slide into the vacated gap (a live reorder), so the
// rack previews the new order before release. During exchange mode a drag carries no
// placement meaning, so it is ignored here (resolved as a tap on end).
func (gs *gameScreen) onRackDrag(idx int, abs fyne.Position) {
	if !gs.humanInputAllowed() || gs.exchangeMode {
		return
	}
	if gs.dragRackSrc < 0 {
		// First event of the gesture: lift the tile at idx if it can be dragged. On later
		// events idx is the (now stale) start slot, so the lifted tile is tracked by
		// dragRackSrc, which the live reorder keeps pointing at it.
		if gs.isStagedSlot(idx) || idx >= gs.state.HumanRack.Count() {
			return
		}
		gs.rackSelected = idx
		gs.dragRackSrc = idx
		gs.pickedUp = [2]int{-1, -1}
		gs.refresh()
	}
	// Live reorder: while the pointer is over the rack (not the board), shift the lifted
	// tile to the hovered slot so the gap follows the pointer. Board placements are
	// resolved on release, so reordering is skipped once the pointer is over the board.
	if _, _, onBoard := gs.cellAt(abs); !onBoard {
		if hoverIdx, ok := gs.rackSlotAt(abs); ok {
			if n := gs.state.HumanRack.Count(); hoverIdx >= n {
				hoverIdx = n - 1
			}
			if hoverIdx >= 0 && hoverIdx != gs.dragRackSrc {
				gs.reorderRack(gs.dragRackSrc, hoverIdx)
			}
		}
	}
	gs.showGhost(gs.state.HumanRack.Tiles()[gs.dragRackSrc], abs, true)
}

// onRackDragEnd resolves a drag that started on the rack by its release position: an empty
// board cell places the lifted tile; a release over the rack keeps the order the live drag
// already produced. In exchange mode the gesture is treated as a tap, toggling the tile's
// exchange selection.
func (gs *gameScreen) onRackDragEnd(fromIdx int, abs fyne.Position) {
	gs.endGhost()
	src := gs.dragRackSrc // current slot of the lifted tile after any live reorder
	gs.dragRackSrc = -1
	if !gs.humanInputAllowed() {
		gs.refresh()
		return
	}
	if gs.exchangeMode {
		gs.onRackTap(fromIdx)
		return
	}
	if src < 0 {
		// The gesture never lifted a tile (started on a staged or empty slot).
		gs.refresh()
		return
	}
	if row, col, ok := gs.cellAt(abs); ok {
		gs.placeFromRack(src, row, col)
		return
	}
	// Released over the rack (or off both): the live drag already applied the reorder.
	gs.refresh()
}

// panPage scrolls the whole page by a drag that no other handler could use: one on the CPU's rack
// row, which has nothing to move, or on a panel too short to scroll. Without this such a gesture is
// absorbed by the widget the driver happened to hand it to, making that area a dead zone.
//
// It does nothing in an arrangement with no page scroll, or while the page fits its viewport.
func (gs *gameScreen) panPage(e *fyne.DragEvent) {
	if gs.page == nil {
		return
	}
	gs.page.panPage(e)
}

// onBoardDoubleTap recalls a tile the player staged this turn; committed tiles (played
// in a previous turn or by the CPU) are left untouched.
func (gs *gameScreen) onBoardDoubleTap(row, col int) {
	if !gs.humanInputAllowed() {
		return
	}
	if st, ok := gs.stagedAt(row, col); ok {
		gs.recallStagedTile(st.FromRackIdx)
		gs.refresh()
	}
}

// onBoardDrag starts and tracks a drag of a staged tile, hiding it on the board and
// driving the ghost. Only tiles staged this turn are draggable; a drag that begins on
// an empty or committed cell does nothing here and is resolved as a tap on release.
func (gs *gameScreen) onBoardDrag(row, col int, abs fyne.Position) {
	if !gs.humanInputAllowed() {
		return
	}
	st, ok := gs.stagedAt(row, col)
	if !ok {
		return
	}
	if gs.dragBoardSrc != [2]int{row, col} {
		gs.dragBoardSrc = [2]int{row, col}
		gs.pickedUp = [2]int{-1, -1}
		gs.refresh()
	}
	gs.showGhost(st.Tile, abs, false)
}

// onBoardDragEnd resolves a board-tile drag by release position: another empty cell
// moves the tile; off the board recalls it to the rack; its own cell leaves it put. A
// gesture that was not a staged-tile drag is handled as a tap.
func (gs *gameScreen) onBoardDragEnd(row, col int, abs fyne.Position) {
	gs.endGhost()
	wasDrag := gs.dragBoardSrc == [2]int{row, col}
	gs.dragBoardSrc = [2]int{-1, -1}
	if !gs.humanInputAllowed() {
		gs.refresh()
		return
	}
	st, ok := gs.stagedAt(row, col)
	if !ok {
		// The gesture began on a cell holding no staged tile, so there was never anything to
		// move: resolve it as a tap, which is how a slight finger movement on a cell arrives.
		gs.refresh()
		gs.onBoardTap(row, col)
		return
	}
	if !wasDrag {
		// A staged tile is here, but this gesture is no longer the recorded drag — its source
		// was cleared while the gesture was still in flight. On touch that happens when a tap
		// arrives during the driver's post-release momentum, and that tap has already been
		// handled. Resolving this late DragEnd as another tap would count the gesture twice,
		// and a synthesised tap on the source cell can land inside the double-press window and
		// recall the tile the player was only trying to place.
		gs.refresh()
		return
	}
	if r2, c2, okB := gs.cellAt(abs); okB {
		if r2 == row && c2 == col {
			// Released on its own cell — a tap, not a move. On touch, taps on a staged
			// tile arrive here (as micro-drags) rather than as Tapped, so detect the
			// double-press here too: a second such tap returns the tile to the rack.
			if gs.registerStagedPress(row, col) {
				gs.onBoardDoubleTap(row, col)
				return
			}
			gs.refresh()
			return
		}
		gs.moveStagedTile(st, r2, c2)
		return
	}
	// Released off the board (over the rack or anywhere else): recall to the rack.
	gs.recallStagedTile(st.FromRackIdx)
	gs.refresh()
}

// moveStagedTile relocates a staged tile to (row,col) when that cell is free; if the
// target is occupied the move is cancelled and the tile stays where it was.
func (gs *gameScreen) moveStagedTile(st stagedTile, row, col int) {
	if !gs.state.Board.IsEmpty(row, col) || gs.cellHasStaged(row, col) {
		gs.refresh()
		return
	}
	for i := range gs.staged {
		if gs.staged[i].FromRackIdx == st.FromRackIdx {
			gs.staged[i].Row, gs.staged[i].Col = row, col
			break
		}
	}
	gs.refresh()
}

// stagedAt returns the staged tile occupying (row,col), if any.
func (gs *gameScreen) stagedAt(row, col int) (stagedTile, bool) {
	for _, st := range gs.staged {
		if st.Row == row && st.Col == col {
			return st, true
		}
	}
	return stagedTile{}, false
}

// showGhost styles the floating ghost tile as t and centres it on the pointer. fromRack
// controls sizing: a rack tile stays rack-slot sized while over the rack and shrinks to a
// board cell once the pointer is over the board (previewing its placed size); a board tile
// (fromRack false) is always board-cell sized.
func (gs *gameScreen) showGhost(t engine.Tile, abs fyne.Position, fromRack bool) {
	if gs.ghost == nil {
		return
	}
	styleAsTile(gs.ghostBg, gs.ghostLetter, gs.ghostPoints, t, true)
	side := gs.ghostSizeAt(abs, fromRack)
	gs.ghost.Resize(fyne.NewSize(side, side))
	gs.ghost.Move(fyne.NewPos(abs.X-side/2, abs.Y-side/2))
	gs.ghost.Show()
	gs.ghost.Refresh()
}

// endGhost hides the floating ghost tile.
func (gs *gameScreen) endGhost() {
	if gs.ghost != nil {
		gs.ghost.Hide()
	}
}

// ghostSizeAt returns the ghost's side length for a pointer at abs. A rack tile (fromRack)
// keeps rack-slot size while over the rack and shrinks to board-cell size once over the
// board; a board tile is always board-cell sized.
func (gs *gameScreen) ghostSizeAt(abs fyne.Position, fromRack bool) float32 {
	if fromRack {
		if _, _, onBoard := gs.cellAt(abs); !onBoard {
			return gs.rackSlotSize()
		}
	}
	return gs.boardCellSize()
}

// boardCellSize returns the side length of a board cell at the current board size.
func (gs *gameScreen) boardCellSize() float32 {
	if gs.boardBox != nil {
		if cell, _, _ := boardGeometry(gs.boardBox.Size().Width, gs.boardBox.Size().Height); cell > 0 {
			return cell
		}
	}
	return minCellPx
}

// rackSlotSize returns the side length of a human rack slot at the current rack size.
func (gs *gameScreen) rackSlotSize() float32 {
	if gs.humanRackBoxC != nil {
		if slot, _ := rackGeometry(gs.humanRackBoxC.Size().Width, gs.humanRackBoxC.Size().Height, engine.MaxRackSize); slot > 0 {
			return slot
		}
	}
	return minRackSlotPx
}

// placeFromRack stages the tile from rack slot fromIdx onto (row,col) when that cell
// is a free placement target; otherwise the drag is cancelled.
func (gs *gameScreen) placeFromRack(fromIdx, row, col int) {
	if !gs.state.Board.IsEmpty(row, col) || gs.cellHasStaged(row, col) {
		gs.refresh()
		return
	}
	gs.stageTile(fromIdx, row, col)
}

// reorderRack moves the tile at rack slot fromIdx to slot toIdx, keeping staged tiles
// and the current selection pointing at the same tiles after the shift.
func (gs *gameScreen) reorderRack(fromIdx, toIdx int) {
	if n := gs.state.HumanRack.Count(); toIdx >= n {
		toIdx = n - 1
	}
	if fromIdx == toIdx {
		gs.refresh()
		return
	}
	gs.state.HumanRack.MoveTile(fromIdx, toIdx)
	for i := range gs.staged {
		gs.staged[i].FromRackIdx = moveIndex(gs.staged[i].FromRackIdx, fromIdx, toIdx)
	}
	if gs.rackSelected >= 0 {
		gs.rackSelected = moveIndex(gs.rackSelected, fromIdx, toIdx)
	}
	// Keep the in-progress drag's lifted-tile slot pointing at the same tile after the
	// shift, so a live reorder mid-drag leaves the ghost floating over it.
	if gs.dragRackSrc >= 0 {
		gs.dragRackSrc = moveIndex(gs.dragRackSrc, fromIdx, toIdx)
	}
	gs.refresh()
}

// cellAt maps an absolute window position to a board cell, reporting ok=false when the
// position is outside the board grid.
func (gs *gameScreen) cellAt(abs fyne.Position) (row, col int, ok bool) {
	if gs.boardBox == nil {
		return 0, 0, false
	}
	origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(gs.boardBox)
	return cellAtRel(abs.Subtract(origin), gs.boardBox.Size())
}

// cellAtRel maps a position relative to the board container's top-left to a cell,
// using the same geometry as boardLayout. ok is false outside the grid.
func cellAtRel(rel fyne.Position, size fyne.Size) (row, col int, ok bool) {
	cell, offX, offY := boardGeometry(size.Width, size.Height)
	if cell <= 0 {
		return 0, 0, false
	}
	x, y := rel.X-offX, rel.Y-offY
	side := cell * boardDim
	if x < 0 || y < 0 || x >= side || y >= side {
		return 0, 0, false
	}
	return int(y / cell), int(x / cell), true
}

// rackSlotAt maps an absolute window position to a human rack slot index, reporting
// ok=false when the position is outside the rack row.
func (gs *gameScreen) rackSlotAt(abs fyne.Position) (idx int, ok bool) {
	if gs.humanRackBoxC == nil {
		return 0, false
	}
	origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(gs.humanRackBoxC)
	return rackSlotAtRel(abs.Subtract(origin), gs.humanRackBoxC.Size())
}

// rackSlotAtRel maps a position relative to the rack container's top-left to a slot
// index, using the same geometry as rackLayout. ok is false outside the rack row.
func rackSlotAtRel(rel fyne.Position, size fyne.Size) (idx int, ok bool) {
	slot, offX := rackGeometry(size.Width, size.Height, engine.MaxRackSize)
	if slot <= 0 {
		return 0, false
	}
	stride := slot + rackGapPx
	x := rel.X - offX
	// The horizontal position selects the slot. The vertical position is given a full
	// row-height of slack above and below the rack, so a reorder drag that drifts
	// vertically — as finger and mouse drags do — still lands on a slot rather than being
	// silently dropped. Board placements are resolved first (via cellAt), so this generous
	// band never steals a drop meant for the board.
	if rel.Y < -size.Height || rel.Y >= 2*size.Height || x < 0 || x >= stride*engine.MaxRackSize {
		return 0, false
	}
	idx = int(x / stride)
	if idx < 0 || idx >= engine.MaxRackSize {
		return 0, false
	}
	return idx, true
}

// isStagedSlot reports whether rack slot idx currently has its tile staged on the board.
func (gs *gameScreen) isStagedSlot(idx int) bool {
	for _, st := range gs.staged {
		if st.FromRackIdx == idx {
			return true
		}
	}
	return false
}

// cellHasStaged reports whether a staged tile already occupies (row,col).
func (gs *gameScreen) cellHasStaged(row, col int) bool {
	for _, st := range gs.staged {
		if st.Row == row && st.Col == col {
			return true
		}
	}
	return false
}

// moveIndex returns the new index of the element originally at i after the element at
// f is moved to position t, matching engine.Rack.MoveTile's reindexing.
func moveIndex(i, f, t int) int {
	switch {
	case i == f:
		return t
	case f < t && i > f && i <= t:
		return i - 1
	case f > t && i >= t && i < f:
		return i + 1
	default:
		return i
	}
}

// afterHumanMove checks for game over, otherwise starts the CPU turn.
func (gs *gameScreen) afterHumanMove() {
	if gs.checkEndGame() {
		return
	}
	gs.startCPUTurn()
}

// checkEndGame ends the game in place when an end condition is met: it applies the
// endgame scoring and refreshes so the final board (including the move that ended the
// game) stays on screen under a GAME OVER banner with the result. Returns true when
// the game is over.
func (gs *gameScreen) checkEndGame() bool {
	over, reason := engine.IsGameOver(gs.state)
	if !over {
		return false
	}
	engine.ApplyEndgameScoring(gs.state, reason)
	gs.gameOver = true
	gs.setStatus(gs.endGameMessage(), false)
	gs.refresh()
	return true
}

// endGameMessage reports the winner and the final scores.
func (gs *gameScreen) endGameMessage() string {
	winner := "It's a tie!"
	switch {
	case gs.state.HumanScore > gs.state.CPUScore:
		winner = "You win!"
	case gs.state.CPUScore > gs.state.HumanScore:
		winner = "CPU wins!"
	}
	return fmt.Sprintf("Game over - %s  (You %d, CPU %d)", winner, gs.state.HumanScore, gs.state.CPUScore)
}

// ---------- CPU turn ----------

// startCPUTurn computes the CPU move on a background goroutine and applies the
// result on the UI goroutine. A clone of the state is handed to the CPU so it
// never shares mutable data with the UI; a timeout converts a stuck CPU to a pass.
func (gs *gameScreen) startCPUTurn() {
	gs.cpuThinking = true
	gs.refresh()

	snapshot := gs.state.Clone()
	dict := gs.dict
	level := gs.state.CPULevel

	go func() {
		result := make(chan engine.Move, 1)
		go func() {
			rng := newGameRNG()
			result <- cpu.ChooseMove(snapshot, dict, level, rng)
		}()

		var move engine.Move
		timedOut := false
		select {
		case move = <-result:
		case <-time.After(cpuTimeoutSecs * time.Second):
			move = engine.PassMove{}
			timedOut = true
		}

		fyne.Do(func() { gs.applyCPUMove(move, timedOut) })
	}()
}

func (gs *gameScreen) applyCPUMove(move engine.Move, timedOut bool) {
	if gs.abandoned {
		return // the player left this game; drop the result
	}
	gs.cpuThinking = false

	// Anything that went wrong with the CPU's turn is held here rather than shown straight
	// away: logCommand clears the status line so a completed move shows the score summary
	// instead of the previous turn's transient message, which would also discard a notice
	// set before it. The notice is re-applied afterwards so the player is actually told.
	notice := ""

	var cmd engine.Command
	switch m := move.(type) {
	case engine.PlayMove:
		cmd = &engine.PlayCommand{Move: m}
	case engine.ExchangeMove:
		cmd = &engine.ExchangeCommand{Move: m}
	case engine.PassMove:
		cmd = &engine.PassCommand{}
	default:
		// Unreachable: engine.Move's marker method is unexported, so the three cases above
		// are the only implementations, and ChooseMove always returns one of them.
		notice = "CPU returned an unknown move - passing."
		cmd = &engine.PassCommand{}
	}

	executed := cmd
	if err := cmd.Execute(gs.state, gs.dict, gs.rng); err != nil {
		notice = fmt.Sprintf("CPU move invalid (%s) - passing.", sanitiseError(err))
		fallback := &engine.PassCommand{}
		_ = fallback.Execute(gs.state, gs.dict, gs.rng)
		executed = fallback
	} else if timedOut {
		notice = "CPU timed out - pass applied."
	}

	gs.logCommand("CPU", executed)

	if notice != "" {
		gs.setStatus(notice, true)
	}

	// The human's turn now begins, so guarantee a clean slate. The human cannot act
	// during the CPU turn, so any staged tile, selection, or drag state lingering here is
	// stale (e.g. from a gesture that outlived the previous turn). Left in place, a stale
	// staged entry blanks its rack slot and makes the rack look like it is missing a tile
	// — and enables the recall button even though nothing was played this turn.
	gs.recallAll()

	if gs.checkEndGame() {
		return
	}
	gs.refresh()
}

// ---------- Utilities ----------

func (gs *gameScreen) setStatus(msg string, isErr bool) {
	gs.statusMsg = msg
	gs.statusIsErr = isErr
}

// statusSegments builds the status line. While the CPU is thinking, or a transient message
// (an error, or feedback like "Game saved.") is set, that single message is shown.
// Otherwise the line shows the most-recent-play score summary: the human's points in green
// and the CPU's in amber, separated by " / ".
func (gs *gameScreen) statusSegments() []widget.RichTextSegment {
	seg := func(text string, colorName fyne.ThemeColorName) *widget.TextSegment {
		return &widget.TextSegment{Text: text, Style: widget.RichTextStyle{
			ColorName: colorName,
			SizeName:  theme.SizeNameSubHeadingText, // larger than body text for legibility
			Alignment: fyne.TextAlignCenter,
			Inline:    true,
		}}
	}

	switch {
	case gs.cpuThinking:
		return []widget.RichTextSegment{seg("CPU is thinking…", theme.ColorNameForeground)}
	case gs.statusMsg != "":
		c := theme.ColorNameSuccess
		if gs.statusIsErr {
			c = theme.ColorNameError
		}
		return []widget.RichTextSegment{seg(gs.statusMsg, c)}
	default:
		return []widget.RichTextSegment{
			seg("You "+playPts(gs.lastHumanPts), theme.ColorNameSuccess),
			seg(" / ", theme.ColorNameForeground),
			seg("CPU "+playPts(gs.lastCPUPts), theme.ColorNameWarning),
		}
	}
}

// playPts formats a most-recent-move score for the status line; the -1 sentinel (no move
// yet) is shown as an em dash.
func playPts(pts int) string {
	if pts < 0 {
		return "—"
	}
	return fmt.Sprintf("+%d", pts)
}

// ---------- Move history ----------

// playLine formats a play's move-history line: Scrabble coordinate notation (e.g.
// "You: 8D UNMIX +28", with any cross-words listed after the main word) when
// scrabbleNotation is set, otherwise the plain word list (e.g. "You: UNMIX, CROSS (+28)").
// It is called after the move is committed, so AnnotatedWords reads the board with the tiles
// in place.
func (gs *gameScreen) playLine(player string, move *engine.PlayMove) string {
	if gs.scrabbleNotation {
		if words := engine.AnnotatedWords(gs.state.Board, move); len(words) > 0 {
			return fmt.Sprintf("%s: %s +%d", player, strings.Join(words, ", "), move.Score)
		}
	}
	words := strings.Join(move.WordsFormed, ", ")
	return fmt.Sprintf("%s: %s (+%d)", player, words, move.Score)
}

// logCommand records one executed turn: it pushes the command onto the undo stack
// and appends its line to the move-history panel.
func (gs *gameScreen) logCommand(player string, cmd engine.Command) {
	var line string
	var words []string
	switch c := cmd.(type) {
	case *engine.PlayCommand:
		line = gs.playLine(player, &c.Move)
		words = c.Move.WordsFormed
		gs.dispatchDefinitions(words)
	case *engine.ExchangeCommand:
		line = fmt.Sprintf("%s: exchanged %d tile(s)", player, len(c.Move.Tiles))
	case *engine.PassCommand:
		line = player + ": passed"
	default:
		return
	}
	gs.history = append(gs.history, historyEntry{
		cmd: cmd, player: player, line: line,
		points: movePoints(cmd), cells: playCells(cmd), words: words,
	})
	gs.recomputeLastPoints()
	// A completed move clears any transient message so the score summary is shown.
	gs.statusMsg = ""
	gs.statusIsErr = false
	gs.recomputeCPUHighlight()
	gs.refreshHistory()
}

// playCells returns the board cells a play placed, for the CPU-word highlight and the
// persisted history; nil for a pass or exchange.
func playCells(cmd engine.Command) [][2]int {
	pc, ok := cmd.(*engine.PlayCommand)
	if !ok {
		return nil
	}
	cells := make([][2]int, len(pc.Move.Placed))
	for i, pt := range pc.Move.Placed {
		cells[i] = [2]int{pt.Row, pt.Col}
	}
	return cells
}

// recomputeLastPoints derives each player's most-recent-move points from the history: a
// play's score, or +0 for a pass or exchange, and the -1 sentinel when a player has not
// moved yet. Deriving from the history keeps the status summary correct across undo.
func (gs *gameScreen) recomputeLastPoints() {
	gs.lastHumanPts = -1
	gs.lastCPUPts = -1
	for i := len(gs.history) - 1; i >= 0; i-- {
		e := gs.history[i]
		if e.player == "You" && gs.lastHumanPts < 0 {
			gs.lastHumanPts = e.points
		} else if e.player == "CPU" && gs.lastCPUPts < 0 {
			gs.lastCPUPts = e.points
		}
		if gs.lastHumanPts >= 0 && gs.lastCPUPts >= 0 {
			break
		}
	}
}

// movePoints is the points a move contributes to the status summary: a play's score, or 0
// for a pass or exchange.
func movePoints(cmd engine.Command) int {
	if pc, ok := cmd.(*engine.PlayCommand); ok {
		return pc.Move.Score
	}
	return 0
}

// recomputeCPUHighlight derives cpuLastPlaced from the move history: the cells of the
// CPU's most recent play that is still on the board. It is called whenever the
// history changes (a move is logged or undone) so the red outline always tracks the
// CPU's latest word; a CPU pass or exchange is skipped because it places no tiles.
func (gs *gameScreen) recomputeCPUHighlight() {
	gs.cpuLastPlaced = nil
	for i := len(gs.history) - 1; i >= 0; i-- {
		e := gs.history[i]
		if e.player != "CPU" || len(e.cells) == 0 {
			continue // not the CPU, or a pass/exchange that places no tiles; keep looking back
		}
		gs.cpuLastPlaced = make(map[[2]int]bool, len(e.cells))
		for _, cell := range e.cells {
			gs.cpuLastPlaced[[2]int{cell[0], cell[1]}] = true
		}
		return
	}
}

// canUndo reports whether there is a human turn made this session available to undo.
// Entries restored from a save carry no command and are not undoable.
func (gs *gameScreen) canUndo() bool {
	for _, e := range gs.history {
		if e.player == "You" && e.cmd != nil {
			return true
		}
	}
	return false
}

// refreshHistory rewrites the history label from the stack and scrolls to the newest
// line so the latest move stays in view.
func (gs *gameScreen) refreshHistory() {
	if gs.historyLabel == nil {
		return
	}
	lines := make([]string, 0, len(gs.history)+1)
	if gs.openingLine != "" {
		lines = append(lines, gs.openingLine)
	}
	for _, e := range gs.history {
		lines = append(lines, e.line)
	}
	// Entries are separated by a blank line, matching the definitions panel: a turn's line can
	// wrap onto several rows, so without the gap it is not obvious where one turn ends.
	gs.historyLabel.SetText(strings.Join(lines, "\n\n"))
	gs.scrollHistoryToEnd()
}

// openingDrawLine returns the move-history line summarising the opening draw — the letter
// each player drew and who plays first — or "" when there is no recorded opening draw (e.g.
// a directly-constructed state in tests, or a saved game from before it was recorded).
func openingDrawLine(od *engine.OpeningDraw) string {
	if od == nil {
		return ""
	}
	first := "you go first"
	if od.First == engine.CPUTurn {
		first = "CPU goes first"
	}
	return fmt.Sprintf("Opening draw: you drew %s, CPU drew %s - %s",
		drawnLetterName(od.HumanLetter), drawnLetterName(od.CPULetter), first)
}

// scrollHistoryToEnd keeps the newest move-history line in view after the log changes.
// Refresh forces the scroll to re-measure the label's now-taller content synchronously, so
// ScrollToBottom targets the new bottom rather than the pre-update height.
func (gs *gameScreen) scrollHistoryToEnd() {
	if gs.historyScroll == nil {
		return
	}
	gs.historyScroll.Refresh()
	gs.historyScroll.ScrollToBottom()
}
