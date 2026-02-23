package main

import (
	"fmt"
	"math"
)

// Shape is an interface that any shape must implement.
type Shape interface {
	Area() float64
	Perimeter() float64
}

// Stringer is a simple interface for types that can describe themselves.
type Stringer interface {
	String() string
}

// Describer composes Shape and Stringer into a single interface.
type Describer interface {
	Shape
	Stringer
}

// Circle implements Shape and Stringer.
type Circle struct {
	Radius float64
}

func (c Circle) Area() float64      { return math.Pi * c.Radius * c.Radius }
func (c Circle) Perimeter() float64 { return 2 * math.Pi * c.Radius }
func (c Circle) String() string     { return fmt.Sprintf("Circle(r=%.2f)", c.Radius) }

// Rectangle implements Shape and Stringer.
type Rectangle struct {
	Width, Height float64
}

func (r Rectangle) Area() float64      { return r.Width * r.Height }
func (r Rectangle) Perimeter() float64 { return 2 * (r.Width + r.Height) }
func (r Rectangle) String() string {
	return fmt.Sprintf("Rectangle(w=%.2f, h=%.2f)", r.Width, r.Height)
}

// Triangle implements Shape only (no String method).
type Triangle struct {
	A, B, C float64 // side lengths
}

func (t Triangle) Area() float64 {
	// Heron's formula
	s := (t.A + t.B + t.C) / 2
	return math.Sqrt(s * (s - t.A) * (s - t.B) * (s - t.C))
}
func (t Triangle) Perimeter() float64 { return t.A + t.B + t.C }

// printShape accepts any Shape — polymorphism via interface.
func printShape(s Shape) {
	fmt.Printf("  Area: %.4f  Perimeter: %.4f\n", s.Area(), s.Perimeter())
}

// printDescriber only accepts types that implement both Shape and Stringer.
func printDescriber(d Describer) {
	fmt.Printf("  %s → Area: %.4f  Perimeter: %.4f\n", d.String(), d.Area(), d.Perimeter())
}

// totalArea works over a slice of any Shape.
func totalArea(shapes []Shape) float64 {
	total := 0.0
	for _, s := range shapes {
		total += s.Area()
	}
	return total
}

func main() {
	c := Circle{Radius: 5}
	r := Rectangle{Width: 4, Height: 6}
	t := Triangle{A: 3, B: 4, C: 5}

	// --- Basic interface usage ---
	fmt.Println("=== Basic interface usage ===")
	shapes := []Shape{c, r, t}
	for _, s := range shapes {
		printShape(s)
	}

	// --- Composed interface ---
	fmt.Println("\n=== Composed interface (Shape + Stringer) ===")
	describers := []Describer{c, r}
	for _, d := range describers {
		printDescriber(d)
	}

	// --- Type assertion ---
	fmt.Println("\n=== Type assertion ===")
	var s Shape = c
	if circle, ok := s.(Circle); ok {
		fmt.Printf("  Asserted to Circle with radius %.2f\n", circle.Radius)
	}

	// --- Type switch ---
	fmt.Println("\n=== Type switch ===")
	for _, shape := range shapes {
		switch v := shape.(type) {
		case Circle:
			fmt.Printf("  Circle  — radius: %.2f\n", v.Radius)
		case Rectangle:
			fmt.Printf("  Rectangle — %dx%d\n", int(v.Width), int(v.Height))
		case Triangle:
			fmt.Printf("  Triangle  — sides: %.0f, %.0f, %.0f\n", v.A, v.B, v.C)
		}
	}

	// --- Interface slice aggregation ---
	fmt.Println("\n=== Aggregation over interface slice ===")
	fmt.Printf("  Total area of all shapes: %.4f\n", totalArea(shapes))

	// --- nil interface ---
	fmt.Println("\n=== nil interface ===")
	var nilShape Shape // zero value of an interface is nil
	fmt.Printf("  nilShape == nil: %v\n", nilShape == nil)
}
