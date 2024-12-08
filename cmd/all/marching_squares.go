package main

import (
	"fmt"
	"image/color"

	"github.com/ebitengine/debugui"
	"github.com/go-gl/mathgl/mgl32"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	ms_grid_size    = 8
	ms_grid_size_sq = ms_grid_size * ms_grid_size
	ms_cell_size_px = 48.0
	ms_grid_size_px = ms_grid_size * ms_cell_size_px
)

type ms_voxel struct {
	active bool
	scale  float32
}

type marching_squares struct {
	voxels [ms_grid_size_sq]ms_voxel
	white  *ebiten.Image

	mid_x, mid_y     int
	scaling          bool
	scale_pos        int
	press_x, press_y int
}

func (m *marching_squares) Load() error {
	m.white = ebiten.NewImage(3, 3)
	m.white.Fill(color.White)
	for i := range m.voxels {
		m.voxels[i].scale = 1.0
	}
	return nil
}

func (m *marching_squares) screen_to_grid(x, y int) (int, int, bool) {
	x -= m.mid_x - ms_grid_size_px/2 - ms_cell_size_px/2
	y -= m.mid_y - ms_grid_size_px/2 - ms_cell_size_px/2
	x /= ms_cell_size_px
	y /= ms_cell_size_px
	if x >= 0 && y >= 0 && x < ms_grid_size && y < ms_grid_size {
		return x, y, true
	}
	return 0, 0, false
}

func (m *marching_squares) Update() error {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		if x, y, ok := m.screen_to_grid(ebiten.CursorPosition()); ok {
			pos := x + (y * ms_grid_size)
			m.voxels[pos].active = !m.voxels[pos].active
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		m.press_x, m.press_y = ebiten.CursorPosition()
		x, y, ok := m.screen_to_grid(m.press_x, m.press_y)
		if ok {
			m.scaling = true
			m.scale_pos = x + (y * ms_grid_size)
		}
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		m.scaling = false
	}
	return nil
}

func (m *marching_squares) Draw(screen *ebiten.Image) {
	mid_x, mid_y := center(screen.Bounds())
	m.mid_x, m.mid_y = mid_x, mid_y

	var vertices []ebiten.Vertex
	var indices []uint16

	push_triangle := func(p0, p1, p2 mgl32.Vec2, clr color.RGBA) {
		r := float32(clr.R) / 255.0
		g := float32(clr.G) / 255.0
		b := float32(clr.B) / 255.0
		vertices = append(vertices,
			ebiten.Vertex{
				DstX:   p0.X(),
				DstY:   p0.Y(),
				ColorR: r,
				ColorG: g,
				ColorB: b,
				ColorA: 0.5,
			},
			ebiten.Vertex{
				DstX:   p1.X(),
				DstY:   p1.Y(),
				ColorR: r,
				ColorG: g,
				ColorB: b,
				ColorA: 0.5,
			},
			ebiten.Vertex{
				DstX:   p2.X(),
				DstY:   p2.Y(),
				ColorR: r,
				ColorG: g,
				ColorB: b,
				ColorA: 0.5,
			},
		)

		index := uint16(len(indices))
		indices = append(indices, index, index+1, index+2)
	}

	for y := 0; y < ms_grid_size; y++ {
		sy := float32(mid_y - (ms_grid_size_px / 2) + (y * ms_cell_size_px))
		sy += 0.5

		for x := 0; x < ms_grid_size; x++ {
			sx := float32(mid_x - (ms_grid_size_px / 2) + (x * ms_cell_size_px))
			sx += 0.5

			vector.StrokeRect(screen, float32(sx), float32(sy), ms_cell_size_px-1, ms_cell_size_px-1, 1, color.RGBA{128, 128, 128, 255}, false)

			if value := m.sample(x, y); value > 0 {
				for _, t := range ms_triangles[value] {
					x0, y0 := t.x0, t.y0
					x1, y1 := t.x1, t.y1
					x2, y2 := t.x2, t.y2
					push_triangle(
						mgl32.Vec2{sx + (x0 * ms_cell_size_px), sy + (y0 * ms_cell_size_px)},
						mgl32.Vec2{sx + (x1 * ms_cell_size_px), sy + (y1 * ms_cell_size_px)},
						mgl32.Vec2{sx + (x2 * ms_cell_size_px), sy + (y2 * ms_cell_size_px)},
						color.RGBA{255, 255, 255, 255},
					)
				}
			}

			clr := color.Black
			if v := m.voxel(x, y); v.active {
				clr = color.White
			}
			vector.DrawFilledRect(screen, float32(sx-1), float32(sy-1), 3, 3, clr, false)
		}
	}

	screen.DrawTriangles(vertices, indices, m.white, nil)

	if m.scaling {
		_, y := ebiten.CursorPosition()
		f := float64(m.press_y-y) / 32.0
		f = min(1, max(-1, f))
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%.3f", f), m.press_x, m.press_y+12)
	}

}

func (m *marching_squares) voxel(x, y int) (marked *ms_voxel) {
	if x >= 0 && y >= 0 && x < ms_grid_size && y < ms_grid_size {
		return &m.voxels[x+(y*ms_grid_size)]
	}
	return nil
}

const (
	TL = 0b1000
	TR = 0b0100
	BR = 0b0010
	BL = 0b0001
)

type tri struct {
	x0, y0, x1, y1, x2, y2 float32
}

var ms_triangles = map[int][]tri{
	BL: {
		{0, 1, 0, 0.5, 0.5, 1},
	},
	BR: {
		{1, 1, 0.5, 1, 1, 0.5},
	},
	TR: {
		{1, 0, 1, 0.5, 0.5, 0},
	},
	TL: {
		{0, 0, 0.5, 0, 0, 0.5},
	},
	BL | BR: {
		{0, 1, 0, 0.5, 1, 0.5},
		{0, 1, 1, 0.5, 1, 1},
	},
	TR | BL: {
		{0, 1, 0, 0.5, 0.5, 1},
		{0, 0.5, 0.5, 0, 0.5, 1},
		{0.5, 0, 1, 0.5, 0.5, 1},
		{0.5, 0, 1, 0, 1, 0.5},
	},
	TL | BL: {
		{0, 0, 0.5, 0, 0.5, 1},
		{0, 0, 0.5, 1, 0, 1},
	},
	TR | BR: {
		{0.5, 0, 1, 0, 1, 1},
		{0.5, 0, 1, 1, 0.5, 1},
	},
	TL | BR: {
		{0, 0, 0.5, 0, 0, 0.5},
		{0.5, 0, 0, 0.5, 1, 0.5},
		{0.5, 1, 0, 0.5, 1, 0.5},
		{0.5, 1, 1, 0.5, 1, 1},
	},
	TL | TR: {
		{0, 0, 1, 0, 0, 0.5},
		{1, 0, 1, 0.5, 0, 0.5},
	},
	TR | BR | BL: {
		{0.5, 0, 1, 0, 1, 1},
		{0.5, 0, 1, 1, 0, 0.5},
		{0, 0.5, 1, 1, 0, 1},
	},
	TL | BL | BR: {
		{0, 0, 0.5, 0, 0, 1},
		{0, 1, 0.5, 0, 1, 0.5},
		{0, 1, 1, 0.5, 1, 1},
	},
	TL | TR | BL: {
		{0, 0, 1, 0, 1, 0.5},
		{0, 0, 1, 0.5, 0.5, 1},
		{0, 0, 0.5, 1, 0, 1},
	},
	TL | TR | BR: {
		{0, 0, 1, 0, 0, 0.5},
		{1, 0, 0.5, 1, 0, 0.5},
		{1, 0, 1, 1, 0.5, 1},
	},
	TL | TR | BR | BL: {
		{0, 0, 1, 0, 0, 1},
		{1, 0, 0, 1, 1, 1},
	},
}

func (m *marching_squares) sample(x, y int) (value int) {
	if v := m.voxel(x, y); v != nil && v.active {
		value |= TL
	}
	if v := m.voxel(x+1, y); v != nil && v.active {
		value |= TR
	}
	if v := m.voxel(x+1, y+1); v != nil && v.active {
		value |= BR
	}
	if v := m.voxel(x, y+1); v != nil && v.active {
		value |= BL
	}
	return
}

func (m *marching_squares) Menu(ctx *debugui.Context) {
	ctx.SetLayoutRow([]int{-1}, 14)
	if ctx.Button("Print") == debugui.ResponseSubmit {
		for i, value := range m.voxels {
			if value.active {
				fmt.Printf("m.points[%d] = %v\n", i, value)
			}
		}
	}
	close_button(ctx)
}
