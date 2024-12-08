package main

import "math"

type vec2i struct {
	x int
	y int
}

func (p vec2i) add(other vec2i) vec2i {
	return p.add2(other.x, other.y)
}

func (p vec2i) add2(x, y int) vec2i {
	return vec2i{
		p.x + x,
		p.y + y,
	}
}

func (p vec2i) distance_sq(other vec2i) int {
	dx := other.x - p.x
	dy := other.y - p.y
	return dx*dx + dy*dy
}

func (p vec2i) distance(other vec2i) int {
	return sqrt_int(p.distance_sq(other))
}

func sqrt_int(x int) int {
	return int(math.Round(math.Sqrt(float64(x))))
}
