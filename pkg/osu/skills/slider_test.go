package skills

import (
	"fmt"
	"math"
	"testing"

	"github.com/compico/go-osu/pkg/osu"
	"github.com/compico/go-osu/pkg/vector2d"
)

func TestSliderInit_NonPositivePixelLength(t *testing.T) {
	position := vector2d.New[float64](128, 256)
	for _, pixelLength := range []float64{-700, 0, math.NaN(), math.Inf(1)} {
		t.Run(fmt.Sprint(pixelLength), func(t *testing.T) {
			hitObject := &osu.HitObject{
				Pos:         position,
				PixelLength: pixelLength,
			}

			slider := &Slider{}
			slider.Init(nil, hitObject)

			if slider.Ncurve != 0 {
				t.Fatalf("Ncurve = %d, want 0", slider.Ncurve)
			}
			if len(slider.Curve) != 1 || slider.Curve[0] != position {
				t.Fatalf("Curve = %v, want one point at %v", slider.Curve, position)
			}
			if hitObject.EndPoint != position {
				t.Fatalf("EndPoint = %v, want %v", hitObject.EndPoint, position)
			}
		})
	}
}

func TestCircumscribedCircle_NonPositivePixelLength(t *testing.T) {
	position := vector2d.New[float64](128, 256)
	hitObject := &osu.HitObject{
		Pos:         position,
		PixelLength: -700,
	}

	circle := NewCircumscribedCircle(hitObject)
	if circle.Ncurve != 0 {
		t.Fatalf("Ncurve = %d, want 0", circle.Ncurve)
	}
	if len(circle.Curve) != 1 || circle.Curve[0] != position {
		t.Fatalf("Curve = %v, want one point at %v", circle.Curve, position)
	}
	if hitObject.EndPoint != position {
		t.Fatalf("EndPoint = %v, want %v", hitObject.EndPoint, position)
	}
}
