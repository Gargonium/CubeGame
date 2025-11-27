package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

// Простая игра на Ebiten по спецификации пользователя.
// Как запускать:
// go get github.com/hajimehoshi/ebiten/v2
// go get golang.org/x/image
// go run golang_ebiten_cubes.go

const (
	windowW     = 800
	windowH     = 600
	shrunkRatio = 0.2
)

type Screen int

const (
	ScreenMenu Screen = iota
	ScreenExitConfirm
	ScreenGame
	ScreenGameOver
)

var (
	redColor        = color.RGBA{R: 0xFF, G: 0x00, B: 0x00, A: 0xFF}
	lightRedColor   = color.RGBA{R: 0xF3, G: 0x64, B: 0x68, A: 0xFF} //#f36468
	greenColor      = color.RGBA{R: 0x00, G: 0xC8, B: 0x00, A: 0xFF}
	lightGreenColor = color.RGBA{R: 0x8C, G: 0xF3, B: 0x8B, A: 0xFF} //#8cf38b
	whiteColor      = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	blackColor      = color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}
	grayColor       = color.RGBA{R: 0x20, G: 0x20, B: 0x20, A: 0xFF}
)

type Button struct {
	Rect image.Rectangle
	Text string
}

func (b *Button) Contains(x, y int) bool {
	return x >= b.Rect.Min.X && x <= b.Rect.Max.X && y >= b.Rect.Min.Y && y <= b.Rect.Max.Y
}

type Square struct {
	X, Y   float64
	W      float64 // current size (width == height)
	InitW  float64 // initial size
	VX, VY float64
	Col    color.Color
	Name   string // "Красный" или "Зелёный"
}

type CirclePickup struct {
	X, Y float64
	R    float64
	Col  color.Color
}

type Game struct {
	screen Screen

	// Menu buttons
	btnStart Button
	btnExit  Button

	// Exit confirm
	btnBack Button
	btnYes  Button

	// Game
	fieldX, fieldY, fieldW, fieldH float64
	bgColor                        color.Color
	sq1, sq2                       *Square
	pickups                        []*CirclePickup
	ticks                          int

	// GameOver
	winnerName string
	btnRetry   Button
	btnMenu    Button
}

func NewGame() *Game {
	g := &Game{}
	g.screen = ScreenMenu
	g.btnStart = Button{Rect: image.Rect(windowW/2-100, 120, windowW/2+100, 170), Text: "Start"}
	g.btnExit = Button{Rect: image.Rect(windowW/2-100, 190, windowW/2+100, 240), Text: "Exit"}
	g.btnBack = Button{Rect: image.Rect(windowW/2-120, 320, windowW/2-10, 370), Text: "Go back"}
	g.btnYes = Button{Rect: image.Rect(windowW/2+10, 320, windowW/2+120, 370), Text: "Yes"}
	g.btnRetry = Button{Rect: image.Rect(windowW/2-120, 360, windowW/2-10, 410), Text: "Start again"}
	g.btnMenu = Button{Rect: image.Rect(windowW/2+10, 360, windowW/2+120, 410), Text: "To menu"}
	g.fieldX = 50
	g.fieldY = 120
	g.fieldW = windowW - 2*g.fieldX
	g.fieldH = float64(windowH - int(g.fieldY) - 40)
	g.bgColor = grayColor
	g.resetGameState()
	return g
}

func (g *Game) resetGameState() {
	// initial squares
	initSize := 60.0
	g.sq1 = &Square{X: g.fieldX + 80, Y: g.fieldY + 60, W: initSize, InitW: initSize, VX: 3.2, VY: 2.6, Col: redColor, Name: "Red"}
	g.sq2 = &Square{X: g.fieldX + g.fieldW - 140, Y: g.fieldY + g.fieldH - 140, W: initSize, InitW: initSize, VX: -3.0, VY: -2.8, Col: greenColor, Name: "Green"}
	g.pickups = []*CirclePickup{}
	g.ticks = 0
	g.bgColor = color.RGBA{R: 0x20, G: 0x20, B: 0x20, A: 0xFF}
	g.winnerName = ""
}

func (g *Game) Update() error {
	// handle input common
	mx, my := ebiten.CursorPosition()
	mouseClicked := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)

	switch g.screen {
	case ScreenMenu:
		if mouseClicked {
			if g.btnStart.Contains(mx, my) {
				g.screen = ScreenGame
				g.resetGameState()
			}
			if g.btnExit.Contains(mx, my) {
				g.screen = ScreenExitConfirm
			}
		}
	case ScreenExitConfirm:
		if mouseClicked {
			if g.btnBack.Contains(mx, my) {
				g.screen = ScreenMenu
			}
			if g.btnYes.Contains(mx, my) {
				// выйти
				os.Exit(0)
			}
		}
	case ScreenGame:
		// update physics
		g.updateGameLogic()
		// allow pressing Escape to go back to menu
		if ebiten.IsKeyPressed(ebiten.KeyEscape) {
			g.screen = ScreenMenu
		}
	case ScreenGameOver:
		if mouseClicked {
			if g.btnRetry.Contains(mx, my) {
				g.resetGameState()
				g.screen = ScreenGame
			}
			if g.btnMenu.Contains(mx, my) {
				g.screen = ScreenMenu
			}
		}
	}

	return nil
}

func (g *Game) updateGameLogic() {
	// spawn pickups occasionally
	g.ticks++

	if g.ticks%360 == 0 {
		if len(g.pickups) != 0 {
			g.pickups = []*CirclePickup{}
		}
		g.spawnPickup()
	}

	// update squares
	g.moveSquare(g.sq1)
	g.moveSquare(g.sq2)

	// check square-pickup collisions
	var remaining []*CirclePickup
	for _, p := range g.pickups {
		if overlapsSquareCircle(g.sq1, p) || overlapsSquareCircle(g.sq2, p) {
			if p.Col == greenColor {
				g.bgColor = lightGreenColor
			} else if p.Col == redColor {
				g.bgColor = lightRedColor
			} else {
				g.bgColor = grayColor
			}
			continue
		}
		remaining = append(remaining, p)
	}
	g.pickups = remaining

	// square-square collision
	if aabbOverlap(g.sq1, g.sq2) {
		// simple elastic response: swap velocities
		tx, ty := g.sq1.VX, g.sq1.VY
		g.sq1.VX, g.sq1.VY = g.sq2.VX, g.sq2.VY
		g.sq2.VX, g.sq2.VY = tx, ty

		// determine vulnerable one (background color not equal to that square)
		var bgCol color.Color
		if g.bgColor == lightRedColor {
			bgCol = redColor
		} else if g.bgColor == lightGreenColor {
			bgCol = greenColor
		} else {
			bgCol = grayColor
		}
		v1 := !colorsEqual(bgCol, g.sq1.Col)
		v2 := !colorsEqual(bgCol, g.sq2.Col)

		// If both vulnerable or both invulnerable, apply shrink to both vulnerable (if any)
		if v1 && !v2 {
			g.sq1.W -= g.sq1.InitW * shrunkRatio
		} else if v2 && !v1 {
			g.sq2.W -= g.sq2.InitW * shrunkRatio
		} else {
			g.sq1.W -= g.sq1.InitW * shrunkRatio
			g.sq2.W -= g.sq2.InitW * shrunkRatio
		}
	}

	// check for death
	if g.sq1.W <= 0 || g.sq2.W <= 0 {
		if g.sq1.W <= 0 && g.sq2.W <= 0 {
			g.winnerName = "Pair"
		} else if g.sq1.W <= 0 {
			g.winnerName = g.sq2.Name
		} else {
			g.winnerName = g.sq1.Name
		}
		g.screen = ScreenGameOver
	}
}

func (g *Game) moveSquare(s *Square) {
	// move
	s.X += s.VX
	s.Y += s.VY

	// walls collision: within fieldX..fieldX+fieldW and fieldY..fieldY+fieldH
	if s.X < g.fieldX {
		s.X = g.fieldX
		s.VX = math.Abs(s.VX)
	}
	if s.Y < g.fieldY {
		s.Y = g.fieldY
		s.VY = math.Abs(s.VY)
	}
	if s.X+s.W > g.fieldX+g.fieldW {
		s.X = g.fieldX + g.fieldW - s.W
		s.VX = -math.Abs(s.VX)
	}
	if s.Y+s.W > g.fieldY+g.fieldH {
		s.Y = g.fieldY + g.fieldH - s.W
		s.VY = -math.Abs(s.VY)
	}
}

func (g *Game) spawnPickup() {
	var col color.Color
	randNum := rand.Intn(3)
	if randNum == 0 {
		col = redColor
	} else if randNum == 1 {
		col = greenColor
	} else {
		col = grayColor
	}
	r := 10.0
	x := g.fieldX + r + rand.Float64()*(g.fieldW-2*r)
	y := g.fieldY + r + rand.Float64()*(g.fieldH-2*r)
	p := &CirclePickup{X: x, Y: y, R: r, Col: col}
	g.pickups = append(g.pickups, p)
}

func overlapsSquareCircle(s *Square, c *CirclePickup) bool {
	// find closest point on square to circle center
	cx := clamp(c.X, s.X, s.X+s.W)
	cy := clamp(c.Y, s.Y, s.Y+s.W)
	dx := cx - c.X
	dy := cy - c.Y
	return dx*dx+dy*dy <= c.R*c.R
}

func clamp(v, a, b float64) float64 {
	if v < a {
		return a
	}
	if v > b {
		return b
	}
	return v
}

func aabbOverlap(a, b *Square) bool {
	return !(a.X+a.W < b.X || b.X+b.W < a.X || a.Y+a.W < b.Y || b.Y+b.W < a.Y)
}

func colorsEqual(a, b color.Color) bool {
	r1, g1, b1, _ := a.RGBA()
	r2, g2, b2, _ := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2
}

func (g *Game) Draw(screen *ebiten.Image) {
	// clear
	screen.Fill(blackColor)

	sz := ebiten.DeviceScaleFactor()
	_ = sz

	switch g.screen {
	case ScreenMenu:
		text.Draw(screen, "Cubes", basicfont.Face7x13, windowW/2-18, 80, whiteColor)
		// buttons
		drawButton(screen, &g.btnStart)
		drawButton(screen, &g.btnExit)
	case ScreenExitConfirm:
		text.Draw(screen, "Cubes", basicfont.Face7x13, windowW/2-18, 80, whiteColor)
		drawButton(screen, &g.btnStart)
		drawButton(screen, &g.btnExit)
		// modal
		drawModal(screen, "Are you sure?", &g.btnBack, &g.btnYes)
	case ScreenGame:
		// draw field
		field := image.Rect(int(g.fieldX), int(g.fieldY), int(g.fieldX+g.fieldW), int(g.fieldY+g.fieldH))
		// fill with bgColor
		ebitenutil.DrawRect(screen, float64(field.Min.X), float64(field.Min.Y), float64(field.Dx()), float64(field.Dy()), g.bgColor)

		// draw pickups
		for _, p := range g.pickups {
			ebitenutil.DrawCircle(screen, p.X, p.Y, p.R+2, blackColor)
			ebitenutil.DrawCircle(screen, p.X, p.Y, p.R, p.Col)
		}

		// draw squares
		drawSquare(screen, g.sq1)
		drawSquare(screen, g.sq2)
	case ScreenGameOver:
		// draw field faded
		ebitenutil.DrawRect(screen, g.fieldX, g.fieldY, g.fieldW, g.fieldH, g.bgColor)
		msg := ""
		if g.winnerName == "Pair" {
			msg = fmt.Sprintf("%s!", g.winnerName)
		} else {
			msg = fmt.Sprintf("%s won!", g.winnerName)
		}
		text.Draw(screen, msg, basicfont.Face7x13, windowW/2-60, windowH/2-40, whiteColor)
		drawButton(screen, &g.btnRetry)
		drawButton(screen, &g.btnMenu)
	}

	// debug FPS
	ebitenutil.DebugPrint(screen, fmt.Sprintf("FPS: %.2f", ebiten.CurrentTPS()))
}

func drawButton(screen *ebiten.Image, b *Button) {
	ebitenutil.DrawRect(screen, float64(b.Rect.Min.X), float64(b.Rect.Min.Y), float64(b.Rect.Dx()), float64(b.Rect.Dy()), color.RGBA{R: 0x30, G: 0x30, B: 0x30, A: 0xFF})
	text.Draw(screen, b.Text, basicfont.Face7x13, b.Rect.Min.X+10, b.Rect.Min.Y+28, whiteColor)
}

func drawModal(screen *ebiten.Image, title string, back, yes *Button) {
	// overlay
	ebitenutil.DrawRect(screen, 100, 200, windowW-200, 180, color.RGBA{R: 0, G: 0, B: 0, A: 200})
	text.Draw(screen, title, basicfont.Face7x13, windowW/2-120, 250, whiteColor)
	drawButton(screen, back)
	drawButton(screen, yes)
}

func drawSquare(screen *ebiten.Image, s *Square) {
	ebitenutil.DrawRect(screen, s.X-2, s.Y-2, s.W+4, s.W+4, blackColor)
	ebitenutil.DrawRect(screen, s.X, s.Y, s.W, s.W, s.Col)
}

func (g *Game) Layout(int, int) (int, int) {
	return windowW, windowH
}

func main() {
	rand.Seed(time.Now().UnixNano())
	g := NewGame()
	ebiten.SetWindowSize(windowW, windowH)
	ebiten.SetWindowTitle("Cubes")
	if err := ebiten.RunGame(g); err != nil {
		fmt.Println("Launch Error:", err)
		os.Exit(1)
	}
}
