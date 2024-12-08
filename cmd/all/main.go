package main

import (
	"errors"
	"image"
	"log"
	"time"

	"github.com/ebitengine/debugui"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	game_width    = 768
	game_height   = 512
	viewport_size = min(game_width, game_height)
	game_scale    = 2
)

type subgame interface {
	Load() error
	Update() error
	Draw(*ebiten.Image)
	Menu(*debugui.Context)
}

var main_game *game

func close_button(ctx *debugui.Context) {
	if ctx.Button("Close") == debugui.ResponseSubmit {
		main_game.subgame = nil
	}
}

func center(r image.Rectangle) (x, y int) {
	x = r.Min.X + r.Dx()/2
	y = r.Min.Y + r.Dy()/2
	return
}

type game struct {
	ui        *debugui.DebugUI
	subgame   subgame
	last_time time.Time
}

func (g *game) Update() error {
	if g.subgame != nil {
		if err := g.subgame.Update(); err != nil {
			return err
		}
	}
	g.ui.Update(func(ctx *debugui.Context) {
		ctx.Window("Settings", image.Rect(grid_size_px+1, 0, game_width-2, game_height-2), func(res debugui.Response, layout debugui.Layout) {
			g.last_time = time.Now()
			if g.subgame != nil {
				g.subgame.Menu(ctx)
			} else {
				ctx.SetLayoutRow([]int{-1}, 14)
				if ctx.Button("Distance Fields") == debugui.ResponseSubmit {
					g.subgame = &distance_field{}
				}
				if ctx.Button("Shaped Tile") == debugui.ResponseSubmit {
					g.subgame = &shape_tile_demo{}
				}
				if ctx.Button("Marching Squares") == debugui.ResponseSubmit {
					g.subgame = &marching_squares{}
				}

				if g.subgame != nil {
					if err := g.subgame.Load(); err != nil {
						panic(err)
					}
				}
			}
		})
	})
	if time.Since(g.last_time) > time.Second {
		return errors.New("window closed")
	}
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	if g.subgame != nil {
		g.subgame.Draw(screen.SubImage(image.Rect(0, 0, viewport_size, viewport_size)).(*ebiten.Image))
	}

	g.ui.Draw(screen)
}

func (a *game) Layout(outside_width, outside_height int) (width, height int) {
	return game_width, game_height
}

func main() {
	ebiten.SetWindowTitle("Playground")
	ebiten.SetWindowSize(game_width*game_scale, game_height*game_scale)
	main_game = &game{
		ui: debugui.New(),
	}
	if err := ebiten.RunGame(main_game); err != nil {
		log.Fatal(err)
	}
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}
