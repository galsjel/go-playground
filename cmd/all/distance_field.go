package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"slices"

	"github.com/ebitengine/debugui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	tile_size    = 8
	tile_size_px = cell_size * tile_size

	grid_tile_divisions = grid_size / tile_size

	grid_size    = 128
	grid_size_px = min(game_width, game_height)

	cell_size = grid_size_px / grid_size

	max_distance = 15
)

type cell struct {
	// closed describes whether this cell is traversable or not. closed means it blocks traversal.
	closed bool
	// space is a weight used to determine the space of the closest closed cell.
	space int
	// _input_cycle is only for user input handling.
	_input_cycle int
}

type tile struct {
	cells *[64]cell
}

type chunk struct {
	tiles *[64]tile
}

func (c *cell) traversable(min_space int) bool {
	if c == nil {
		return false
	}
	return !c.closed && c.space >= min_space
}

type distance_field struct {
	cycle      int
	move_timer int
	drag_cycle int
	cells      []cell

	draw_distance_field bool
	draw_grids          bool
	grid_dirty          bool
	grid_image          *ebiten.Image

	player_x    int
	player_y    int
	player_size float64

	max_reach float64
	goal      vec2i
	path      []vec2i
	path_ok   bool
}

func (g *distance_field) in_bounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < grid_size && y < grid_size
}

func (g *distance_field) cell_at(x, y int) *cell {
	if g.in_bounds(x, y) {
		return &g.cells[x+(y*grid_size)]
	}
	return nil
}

func (g *distance_field) cell_at_pos(p vec2i) *cell {
	return g.cell_at(p.x, p.y)
}

func (f *distance_field) update() error {
	return nil
}

func (f *distance_field) draw(screen *ebiten.Image) {

}

type path_args struct {
	start vec2i
	goal  vec2i
	// min_space is the minimum required space for any given cell to allow traversal.
	min_space int
	// max_distance determines the farthest from the start position we should be allowed to search. no limit <= 0
	max_distance int
	// max_reach determines the farthest distance from the goal we're allowed to form a path to.
	max_reach int
}

type direction int

const (
	northwest direction = iota
	north
	northeast
	east
	southeast
	south
	southwest
	west
)

var path_directions = [...]direction{
	north,
	east,
	south,
	west,
	northwest,
	northeast,
	southeast,
	southwest,
}

func (d direction) vec2i() vec2i {
	switch d {
	case northwest:
		return vec2i{-1, -1}
	case north:
		return vec2i{0, -1}
	case northeast:
		return vec2i{1, -1}
	case east:
		return vec2i{1, 0}
	case southeast:
		return vec2i{1, 1}
	case south:
		return vec2i{0, 1}
	case southwest:
		return vec2i{-1, 1}
	case west:
		return vec2i{-1, 0}
	default:
		return vec2i{0, 0}
	}
}

func (d direction) rotate_ccw() direction {
	return (d + 7) & 0b111
}

func (d direction) rotate_cw() direction {
	return (d + 1) & 0b111
}

func (d direction) diagonal() bool {
	return d&1 == 0
}

func (g *distance_field) bfs(arg path_args) ([]vec2i, bool) {
	visited := make(map[vec2i]struct{})
	prev := make(map[vec2i]vec2i)

	construct_path := func(end vec2i) (path []vec2i) {
		for end != arg.start {
			path = append(path, end)
			end = prev[end]
		}
		path = append(path, arg.start)
		slices.Reverse(path)
		return
	}

	queue := []vec2i{arg.start}
	visited[arg.start] = struct{}{}

	max_reach := arg.max_reach * arg.max_reach
	max_distance := arg.max_distance * arg.max_distance

	var closest vec2i
	var closest_distance int = math.MaxInt

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		distance := cur.distance_sq(arg.goal)

		if max_distance > 0 && distance > max_distance {
			continue
		}

		if cur == arg.goal {
			return construct_path(arg.goal), true
		}

		if max_reach > 0 && distance < closest_distance && distance <= max_reach {
			closest = cur
			closest_distance = distance
		}

		can_traverse := func(c *cell) bool {
			if c == nil {
				return false
			} else if c.closed {
				return false
			} else if c.space < arg.min_space {
				return false
			}
			return true
		}

		for _, dir := range path_directions {
			next := cur.add(dir.vec2i())

			if _, skip := visited[next]; skip {
				continue
			}

			if dir.diagonal() {
				if cell := g.cell_at_pos(cur.add(dir.rotate_cw().vec2i())); !can_traverse(cell) {
					continue
				} else if cell := g.cell_at_pos(cur.add(dir.rotate_ccw().vec2i())); !can_traverse(cell) {
					continue
				}
			}

			visited[next] = struct{}{}

			if cell := g.cell_at_pos(next); !can_traverse(cell) {
				continue
			}

			prev[next] = cur
			queue = append(queue, next)
		}
	}

	// we didn't reach our goal, but we can still return a sub-optimal.
	if max_reach > 0 && closest_distance != math.MaxInt {
		return construct_path(closest), false
	}

	return nil, false
}

func (g *distance_field) update_cell(x, y int) (ok bool) {
	if cell := g.cell_at(x, y); cell != nil {
		if cell.closed {
			cell.space = 0
		} else {
			cell.space = max_distance
			for dy := -max_distance; dy <= max_distance; dy++ {
				for dx := -max_distance; dx <= max_distance; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					other_x := x + dx
					other_y := y + dy
					distance := min(sqrt_int(dx*dx+dy*dy), max_distance)
					if int(distance) >= cell.space {
						continue
					}
					other := g.cell_at(other_x, other_y)
					if other == nil || other.closed {
						cell.space = int(distance)
					}
				}
			}
		}
		g.grid_dirty = true
		return true
	}
	return false
}

func (g *distance_field) update_cells(x0, y0, x1, y1 int) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			g.update_cell(x, y)
		}
	}
}

func (g *distance_field) update_all_cells() {
	g.update_cells(0, 0, grid_size, grid_size)
}

type tile_data struct {
	X           int
	Y           int
	ClosedCells uint64
}

func (g *distance_field) Load() error {
	g.cells = make([]cell, grid_size*grid_size)
	g.player_size = 1
	g.goal = vec2i{16, 16}
	g.update_all_cells()
	g.update_path()
	return nil
}

func (g *distance_field) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyShift) {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			cx, cy := ebiten.CursorPosition()
			grid_x, grid_y := cx/cell_size, cy/cell_size
			g.goal = vec2i{grid_x, grid_y}
			g.update_path()
		}
	} else {
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			g.drag_cycle = g.cycle
		} else if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			cx, cy := ebiten.CursorPosition()
			grid_x, grid_y := cx/cell_size, cy/cell_size
			if cell := g.cell_at(grid_x, grid_y); cell != nil && cell._input_cycle != g.drag_cycle {
				cell._input_cycle = g.drag_cycle
				cell.closed = !cell.closed
				g.update_cells(grid_x-max_distance, grid_y-max_distance, grid_x+max_distance, grid_y+max_distance)
				g.update_path()
			}
		}
	}

	var dir direction = -1
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		dir = north
	} else if ebiten.IsKeyPressed(ebiten.KeyS) {
		dir = south
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		if dir == north {
			dir = northwest
		} else if dir == south {
			dir = southwest
		} else {
			dir = west
		}
	} else if ebiten.IsKeyPressed(ebiten.KeyD) {
		if dir == north {
			dir = northeast
		} else if dir == south {
			dir = southeast
		} else {
			dir = east
		}
	}

	if g.move_timer == 0 && dir != -1 {
		v := dir.vec2i()
		player_size := int(g.player_size)

		ok := g.cell_at(g.player_x+v.x, g.player_y+v.y).traversable(player_size)

		// diagonal movement
		if ok && dir.diagonal() {
			if !g.cell_at(g.player_x+v.x, g.player_y).traversable(player_size) {
				v.x = 0
			}
			if !g.cell_at(g.player_x, g.player_y+v.y).traversable(player_size) {
				v.y = 0
			}
		}

		if !ok && g.cell_at(g.player_x+v.x, g.player_y).traversable(player_size) {
			ok = true
			v.y = 0
		}

		if !ok && g.cell_at(g.player_x, g.player_y+v.y).traversable(player_size) {
			ok = true
			v.x = 0
		}

		if ok {
			g.player_x += v.x
			g.player_y += v.y
			g.move_timer = 3
			g.update_path()
		}
	}

	g.cycle++

	if g.move_timer > 0 {
		g.move_timer--
	}
	return nil
}

func (g *distance_field) Menu(ctx *debugui.Context) {
	ctx.SetLayoutRow([]int{-1}, 24)
	ctx.LayoutColumn(func() {
		ctx.SetLayoutRow([]int{74, -1}, 16)
		ctx.Label("Player Size")
		if ctx.Slider(&g.player_size, 1, max_distance, 1, 0) == debugui.ResponseChange {
			g.update_path()
		}
		ctx.Label("Max Reach")
		if ctx.Slider(&g.max_reach, 0, 64, 1, 0) == debugui.ResponseChange {
			g.update_path()
		}

		if ctx.Button("Save") == debugui.ResponseSubmit {
			var tiles []tile_data
			for tile_y := 0; tile_y < grid_tile_divisions; tile_y++ {
				for tile_x := 0; tile_x < grid_tile_divisions; tile_x++ {
					grid_x := tile_x * tile_size
					grid_y := tile_y * tile_size
					var closed_cells uint64
					for y0 := 0; y0 < tile_size; y0++ {
						for x0 := 0; x0 < tile_size; x0++ {
							closed_cells <<= 1
							cell_x := grid_x + x0
							cell_y := grid_y + y0
							if cell := g.cell_at(cell_x, cell_y); cell == nil || cell.closed {
								closed_cells |= 1
							}
						}
					}
					tiles = append(tiles, tile_data{
						X:           tile_x,
						Y:           tile_y,
						ClosedCells: closed_cells,
					})
				}
			}

			var buf bytes.Buffer
			enc := gob.NewEncoder(&buf)
			if err := enc.Encode(tiles); err != nil {
				log.Println(err)
			} else if err = os.WriteFile("save.dat", buf.Bytes(), 0755); err != nil {
				log.Println(err)
			}
			log.Printf("data length: %d", buf.Len())
		}

		if ctx.Button("Load") == debugui.ResponseSubmit {
			data, err := os.ReadFile("save.dat")
			if err != nil {
				log.Println(err)
			} else {
				dec := gob.NewDecoder(bytes.NewReader(data))
				var chunks []tile_data
				if err = dec.Decode(&chunks); err != nil {
					log.Println(err)
				} else {
					for _, chunk := range chunks {
						chunk_x := chunk.X * tile_size
						chunk_y := chunk.Y * tile_size
						closed_cells := chunk.ClosedCells
						for y := tile_size - 1; y >= 0; y-- {
							for x := tile_size - 1; x >= 0; x-- {
								g.cell_at(x+chunk_x, y+chunk_y).closed = closed_cells&1 == 1
								closed_cells >>= 1
							}
						}
					}
				}
				g.update_all_cells()
			}
		}

		if ctx.Button("Clear") == debugui.ResponseSubmit {
			for i := range g.cells {
				g.cells[i].closed = false
			}
			g.update_all_cells()
		}
		if ctx.Button("Fill") == debugui.ResponseSubmit {
			for i := range g.cells {
				g.cells[i].closed = true
			}
			g.update_all_cells()
		}
	})
	ctx.LayoutColumn(func() {
		ctx.SetLayoutRow([]int{-1}, 14)
		if ctx.Checkbox("Draw Grids", &g.draw_grids) == debugui.ResponseChange {
			g.grid_dirty = true
		}
		if ctx.Checkbox("Draw Distance Field", &g.draw_distance_field) == debugui.ResponseChange {
			g.grid_dirty = true
		}
		ctx.Label("")
		ctx.Label("Left-click and drag to toggle the cells")
		ctx.Label("open/closed state.")
		ctx.Label("")
		ctx.Label("The number on each cell represents the")
		ctx.Label("available capacity for that cell.")
		ctx.Label("")
		ctx.Label("The capacity is compared to the player")
		ctx.Label("size and determines whether they can")
		ctx.Label("move into that cell.")
		ctx.Label("")
		ctx.Label("Hold shift and left-click to set the")
		ctx.Label("goal.")
		ctx.Label("")

		ctx.Label(fmt.Sprintf("TPS: %.3f", ebiten.ActualTPS()))
		ctx.Label(fmt.Sprintf("FPS: %.3f", ebiten.ActualFPS()))
		close_button(ctx)
	})
}

func (g *distance_field) update_path() {
	g.path, g.path_ok = g.bfs(path_args{
		start:        vec2i{g.player_x, g.player_y},
		goal:         g.goal,
		min_space:    int(g.player_size),
		max_distance: 0,
		max_reach:    int(g.max_reach),
	})
}

func (g *distance_field) Draw(screen *ebiten.Image) {
	if g.grid_dirty {
		g.grid_dirty = false
		log.Println("repaint grid")
		if g.grid_image == nil {
			g.grid_image = ebiten.NewImage(grid_size_px, grid_size_px)
			// g.grid_image = ebiten.NewImageWithOptions(image.Rect(0, 0, grid_size_px, grid_size_px), &ebiten.NewImageOptions{
			// 	Unmanaged: true,
			// })
		}
		g.grid_image.Clear()
		for grid_y := 0; grid_y < grid_size; grid_y++ {
			for grid_x := 0; grid_x < grid_size; grid_x++ {
				cell_x := float32(grid_x*cell_size) + .5
				cell_y := float32(grid_y*cell_size) + .5
				cell := g.cell_at(grid_x, grid_y)

				var clr color.Color = color.RGBA{127, 127, 127, 255}

				if g.draw_distance_field {
					grey := uint8((cell.space * 255) / max_distance)
					clr = color.RGBA{
						grey, grey, grey, 255,
					}
				} else {
					if cell.closed {
						clr = color.Black
					}
				}

				vector.DrawFilledRect(g.grid_image, cell_x, cell_y, cell_size, cell_size, clr, false)

				if g.draw_grids {
					vector.StrokeRect(g.grid_image, cell_x, cell_y, cell_size, cell_size, 1, color.RGBA{64, 64, 64, 64}, false)
				}
			}
		}

		if g.draw_grids {
			for chunk_y := 0; chunk_y < grid_tile_divisions; chunk_y++ {
				for chunk_x := 0; chunk_x < grid_tile_divisions; chunk_x++ {
					x := float32(chunk_x*tile_size_px) + .5
					y := float32(chunk_y*tile_size_px) + .5
					vector.StrokeRect(g.grid_image, x, y, tile_size_px-1, tile_size_px-1, 1, color.RGBA{16, 48, 98, 128}, false)
				}
			}
		}
	}

	screen.DrawImage(g.grid_image, nil)

	clr := color.RGBA{64, 255, 128, 255}

	if !g.path_ok {
		clr = color.RGBA{255, 64, 128, 255}
		cell_x := float32(g.goal.x*cell_size) + .5
		cell_y := float32(g.goal.y*cell_size) + .5
		vector.DrawFilledRect(screen, cell_x+1, cell_y+1, cell_size-2, cell_size-2, clr, false)
	}

	for _, pos := range g.path {
		cell_x := float32(pos.x*cell_size) + .5
		cell_y := float32(pos.y*cell_size) + .5
		vector.DrawFilledRect(screen, cell_x+3, cell_y+3, cell_size-5, cell_size-5, clr, false)
	}

	player_x := float32(g.player_x*cell_size) + (cell_size * 0.5)
	player_y := float32(g.player_y*cell_size) + (cell_size * 0.5)
	vector.DrawFilledCircle(screen, player_x+1, player_y+1, 2, color.RGBA{0, 0, 0, 64}, false)
	vector.DrawFilledCircle(screen, player_x, player_y, 2, color.RGBA{0, 255, 0, 255}, false)
	vector.StrokeCircle(screen, player_x+1, player_y+1, float32((1+2*(g.player_size-1))*cell_size*0.5), 1, color.RGBA{0, 0, 0, 64}, false)
	vector.StrokeCircle(screen, player_x, player_y, float32((1+2*(g.player_size-1))*cell_size*0.5), 1, color.RGBA{0, 255, 0, 255}, false)
}
