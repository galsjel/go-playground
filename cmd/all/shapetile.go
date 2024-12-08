package main

import (
	"image/color"

	"github.com/ebitengine/debugui"
	"github.com/go-gl/mathgl/mgl64"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type node struct {
	weight float64
	layer  float64
}

func (n node) connected_to(other node) bool {
	return n.layer == other.layer
}

type shape_tile_demo struct {
	white_texture  *ebiten.Image
	preview_radius float64
	tl             node
	tr             node
	bl             node
	br             node
}

func (d *shape_tile_demo) Load() error {
	d.tl.layer = 0
	d.tr.layer = 1
	d.bl.layer = 2
	d.br.layer = 3
	d.preview_radius = 64
	d.white_texture = ebiten.NewImage(3, 3)
	d.white_texture.Fill(color.White)
	return nil
}

func (d *shape_tile_demo) Update() error {
	return nil
}

func lerp(a, b mgl64.Vec2, f float64) mgl64.Vec2 {
	return a.Add(b.Sub(a).Mul(f))
}

func (d *shape_tile_demo) Draw(screen *ebiten.Image) {
	bounds := screen.Bounds()
	mid_x := float64(bounds.Min.X + bounds.Dx()/2)
	mid_y := float64(bounds.Min.Y + bounds.Dy()/2)

	mid := mgl64.Vec2{mid_x, mid_y}
	tl := mid.Add(mgl64.Vec2{-d.preview_radius, -d.preview_radius})
	tr := mid.Add(mgl64.Vec2{+d.preview_radius, -d.preview_radius})
	bl := mid.Add(mgl64.Vec2{-d.preview_radius, +d.preview_radius})
	br := mid.Add(mgl64.Vec2{+d.preview_radius, +d.preview_radius})

	mid_top := lerp(tl, tr, 0.5)
	mid_right := lerp(tr, br, 0.5)
	mid_left := lerp(tl, bl, 0.5)
	mid_bottom := lerp(bl, br, 0.5)

	tl_mid := lerp(tl, mid, d.tl.weight)
	tl_top := lerp(tl, mid_top, d.tl.weight)
	tl_left := lerp(tl, mid_left, d.tl.weight)

	tr_mid := lerp(tr, mid, d.tr.weight)
	tr_top := lerp(tr, mid_top, d.tr.weight)
	tr_right := lerp(tr, mid_right, d.tr.weight)

	bl_mid := lerp(bl, mid, d.bl.weight)
	bl_left := lerp(bl, mid_left, d.bl.weight)
	bl_bottom := lerp(bl, mid_bottom, d.bl.weight)

	br_mid := lerp(br, mid, d.br.weight)
	br_right := lerp(br, mid_right, d.br.weight)
	br_bottom := lerp(br, mid_bottom, d.br.weight)

	var vertices []ebiten.Vertex
	var indices []uint16

	push_triangle := func(p0, p1, p2 mgl64.Vec2, clr color.RGBA) {
		r := float32(clr.R) / 255.0
		g := float32(clr.G) / 255.0
		b := float32(clr.B) / 255.0
		vertices = append(vertices,
			ebiten.Vertex{
				DstX:   float32(p0.X()),
				DstY:   float32(p0.Y()),
				ColorR: r,
				ColorG: g,
				ColorB: b,
				ColorA: 1,
			},
			ebiten.Vertex{
				DstX:   float32(p1.X()),
				DstY:   float32(p1.Y()),
				ColorR: r,
				ColorG: g,
				ColorB: b,
				ColorA: 1,
			},
			ebiten.Vertex{
				DstX:   float32(p2.X()),
				DstY:   float32(p2.Y()),
				ColorR: r,
				ColorG: g,
				ColorB: b,
				ColorA: 1,
			},
		)

		index := uint16(len(indices))
		indices = append(indices, index, index+1, index+2)
	}

	push_triangle(tl, tl_mid, tl_top, color.RGBA{255, 140, 0, 255})
	push_triangle(tl, tl_mid, tl_left, color.RGBA{255, 140, 0, 255})

	push_triangle(tr, tr_mid, tr_top, color.RGBA{255, 140, 0, 255})
	push_triangle(tr, tr_mid, tr_right, color.RGBA{255, 140, 0, 255})

	push_triangle(bl, bl_mid, bl_left, color.RGBA{255, 140, 0, 255})
	push_triangle(bl, bl_mid, bl_bottom, color.RGBA{255, 140, 0, 255})

	push_triangle(br, br_mid, br_right, color.RGBA{255, 140, 0, 255})
	push_triangle(br, br_mid, br_bottom, color.RGBA{255, 140, 0, 255})

	if d.tl.connected_to(d.tr) {
		push_triangle(tl_mid, tl_top, tr_top, color.RGBA{159, 86, 255, 255})
		push_triangle(tl_mid, tr_top, tr_mid, color.RGBA{159, 86, 255, 255})
	}

	if d.tl.connected_to(d.bl) {
		push_triangle(tl_mid, tl_left, bl_left, color.RGBA{159, 86, 255, 255})
		push_triangle(tl_mid, bl_left, bl_mid, color.RGBA{159, 86, 255, 255})
	}

	if d.tr.connected_to(d.br) {
		push_triangle(tr_mid, tr_right, br_right, color.RGBA{159, 86, 255, 255})
		push_triangle(tr_mid, br_right, br_mid, color.RGBA{159, 86, 255, 255})
	}

	if d.bl.connected_to(d.br) {
		push_triangle(bl_mid, bl_bottom, br_bottom, color.RGBA{159, 86, 255, 255})
		push_triangle(bl_mid, br_bottom, br_mid, color.RGBA{159, 86, 255, 255})
	}

	screen.DrawTriangles(vertices, indices, d.white_texture, nil)

	for _, point := range []mgl64.Vec2{tl_top, tr_top, tr_right, br_right, tl_left, bl_left, bl_bottom, br_bottom} {
		vector.DrawFilledRect(screen, float32(point.X()-1), float32(point.Y()-1), 3, 3, color.RGBA{0, 255, 255, 255}, false)
	}
	for _, point := range []mgl64.Vec2{mid_top, mid_right, mid_left, mid_bottom} {
		vector.DrawFilledRect(screen, float32(point.X()-1), float32(point.Y()-1), 3, 3, color.RGBA{0, 0, 255, 255}, false)
	}
	for _, point := range []mgl64.Vec2{tl_mid, tr_mid, bl_mid, br_mid} {
		vector.DrawFilledRect(screen, float32(point.X()-1), float32(point.Y()-1), 3, 3, color.RGBA{0, 255, 0, 255}, false)
	}
	for _, point := range []mgl64.Vec2{mid, tl, tr, bl, br} {
		vector.DrawFilledRect(screen, float32(point.X()-1), float32(point.Y()-1), 3, 3, color.RGBA{255, 0, 0, 255}, false)
	}
}

func (d *shape_tile_demo) Menu(ctx *debugui.Context) {
	ctx.SetLayoutRow([]int{48, -1}, 14)
	ctx.Label("Radius")
	ctx.Slider(&d.preview_radius, 8, 256, 1, 1)
	ctx.SetLayoutRow([]int{64, 64, 32}, 14)
	ctx.Label("TL")
	ctx.Slider(&d.tl.weight, 0, 1, 1.0/3.0, 2)
	ctx.Slider(&d.tl.layer, 0, 3, 1, 1)
	ctx.Label("TR")
	ctx.Slider(&d.tr.weight, 0, 1, 1.0/3.0, 2)
	ctx.Slider(&d.tr.layer, 0, 3, 1, 1)
	ctx.Label("BL")
	ctx.Slider(&d.bl.weight, 0, 1, 1.0/3.0, 2)
	ctx.Slider(&d.bl.layer, 0, 3, 1, 1)
	ctx.Label("BR")
	ctx.Slider(&d.br.weight, 0, 1, 1.0/3.0, 2)
	ctx.Slider(&d.br.layer, 0, 3, 1, 1)
	ctx.SetLayoutRow([]int{-1}, 14)
	close_button(ctx)
}
