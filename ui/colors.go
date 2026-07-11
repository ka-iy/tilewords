// Package ui is documented in doc.go.
package ui

import "image/color"

// Board and premium-square colours. This palette is independently designed and does
// not reproduce Hasbro's copyrighted board artwork (NFR-09 / BR-UI-13).
var (
	colorBoardBg  = color.RGBA{R: 34, G: 139, B: 34, A: 255}   // forest green
	colorDW       = color.RGBA{R: 244, G: 164, B: 96, A: 255}  // sandy orange
	colorTW       = color.RGBA{R: 205, G: 92, B: 92, A: 255}   // indian red
	colorDL       = color.RGBA{R: 135, G: 206, B: 235, A: 255} // sky blue
	colorTL       = color.RGBA{R: 65, G: 105, B: 225, A: 255}  // royal blue
	colorCentre   = color.RGBA{R: 255, G: 215, B: 0, A: 255}   // gold
	colorGrid     = color.RGBA{R: 0, G: 80, B: 0, A: 255}      // dark green
	colorPremText = color.RGBA{R: 255, G: 255, B: 255, A: 220} // white label on premium square
)

// Tile colours (BR-UI-14 / BR-UI-18).
var (
	colorTileBg           = color.RGBA{R: 245, G: 222, B: 179, A: 255} // wheat — committed tile
	colorTileStagedBg     = color.RGBA{R: 255, G: 250, B: 205, A: 255} // lemon chiffon — staged tile
	colorTileStagedBorder = color.RGBA{R: 218, G: 165, B: 32, A: 255}  // goldenrod border on staged tile
	colorTileLetter       = color.RGBA{R: 30, G: 30, B: 30, A: 255}    // near-black letter
	colorTileBlankLetter  = color.RGBA{R: 0, G: 80, B: 200, A: 255}    // blue — blank tile assigned letter
	colorTilePoints       = color.RGBA{R: 80, G: 60, B: 20, A: 255}    // dark brown points value
	colorTileFaceDown     = color.RGBA{R: 20, G: 80, B: 20, A: 255}    // dark green — hidden AI rack tile
	colorTileBorder       = color.RGBA{R: 160, G: 120, B: 60, A: 255}  // brown border
	colorAILastWord       = color.RGBA{R: 220, G: 20, B: 60, A: 255}   // crimson — border on the AI's most recent word
)

// UI control colours.
var (
	colorBtnEnabled  = color.RGBA{R: 70, G: 130, B: 180, A: 255}  // steel blue
	colorBtnDisabled = color.RGBA{R: 150, G: 150, B: 150, A: 255} // grey
	colorBtnHover    = color.RGBA{R: 100, G: 160, B: 210, A: 255} // lighter blue
	colorBtnText     = color.RGBA{R: 255, G: 255, B: 255, A: 255} // white
	colorBtnBorder   = color.RGBA{R: 40, G: 90, B: 140, A: 255}   // dark blue border
	colorBtnSelected = color.RGBA{R: 34, G: 139, B: 34, A: 255}   // green — selected dict/level

	colorPanel      = color.RGBA{R: 30, G: 30, B: 50, A: 255}    // dark navy panel background
	colorOverlay    = color.RGBA{R: 0, G: 0, B: 0, A: 180}       // semi-transparent overlay
	colorStatusOK   = color.RGBA{R: 200, G: 255, B: 200, A: 255} // light green status
	colorStatusErr  = color.RGBA{R: 255, G: 180, B: 180, A: 255} // light red error
	colorTitle      = color.RGBA{R: 255, G: 215, B: 0, A: 255}   // gold title
	colorText       = color.RGBA{R: 220, G: 220, B: 220, A: 255} // light grey body text
	colorBackground = color.RGBA{R: 20, G: 20, B: 40, A: 255}    // very dark navy — screen bg
)

// Rack slot placeholder (empty rack cell).
var colorRackSlot = color.RGBA{R: 60, G: 100, B: 60, A: 255}

// colorTurnYou tints the rack label (and the play icon) green on the human's turn.
var colorTurnYou = color.RGBA{R: 80, G: 220, B: 90, A: 255} // bright green

// colorPickedUp outlines a staged tile that has been tapped to move (tap-to-move).
var colorPickedUp = color.RGBA{R: 0, G: 200, B: 255, A: 255} // bright cyan

// colorGameOver tints the rack label red when the game has ended.
var colorGameOver = color.RGBA{R: 235, G: 60, B: 60, A: 255} // vivid red

// Exchange-selected tile highlight.
var colorTileExchangeSel = color.RGBA{R: 255, G: 165, B: 0, A: 255} // orange

// Light-theme text colours. The titles, labels, status line, and rack cue are drawn with
// explicit colours chosen for the dark background (gold title, light-grey body text,
// bright green/red cues). On the light theme those wash out against white, so these
// darker, high-contrast counterparts are used instead (see the accessors in theme.go).
var (
	colorTitleLight = color.RGBA{R: 168, G: 116, B: 0, A: 255}  // dark amber — title on white
	colorTextLight  = color.RGBA{R: 45, G: 45, B: 45, A: 255}   // near-black body text on white
	colorGreenLight = color.RGBA{R: 21, G: 122, B: 45, A: 255}  // dark green cue on white
	colorRedLight   = color.RGBA{R: 197, G: 32, B: 32, A: 255}  // dark red cue on white
)
