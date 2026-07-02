package vector2d

import "math"

type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~float32 | ~float64
}

func Min[T Numeric](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func Max[T Numeric](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func Clamp[T Numeric](value, low, high T) T {
	return Min(Max(value, low), high)
}

type Vector2d[T Numeric] struct {
	X, Y T
}

type Vector2di = Vector2d[int32]
type Vector2df = Vector2d[float32]
type Vector2dd = Vector2d[float64]

func New[T Numeric](x, y T) Vector2d[T] {
	return Vector2d[T]{
		X: x,
		Y: y,
	}
}

func (v *Vector2d[T]) Negate() Vector2d[T] {
	return Vector2d[T]{
		X: -v.X,
		Y: -v.Y,
	}
}

func (v *Vector2d[T]) Add(other Vector2d[T]) Vector2d[T] {
	return Vector2d[T]{
		X: v.X + other.X,
		Y: v.Y + other.Y,
	}
}

func (v *Vector2d[T]) AddInPlace(other Vector2d[T]) *Vector2d[T] {
	v.X += other.X
	v.Y += other.Y
	return v
}

func (v *Vector2d[T]) AddScalar(s T) Vector2d[T] {
	return Vector2d[T]{
		X: v.X + s,
		Y: v.Y + s,
	}
}

func (v *Vector2d[T]) AddScalarInPlace(s T) *Vector2d[T] {
	v.X += s
	v.Y += s
	return v
}

func (v *Vector2d[T]) Sub(other Vector2d[T]) Vector2d[T] {
	return Vector2d[T]{
		X: v.X - other.X,
		Y: v.Y - other.Y,
	}
}

func (v *Vector2d[T]) SubInPlace(other Vector2d[T]) *Vector2d[T] {
	v.X -= other.X
	v.Y -= other.Y
	return v
}

func (v *Vector2d[T]) SubScalar(s T) Vector2d[T] {
	return Vector2d[T]{
		X: v.X - s,
		Y: v.Y - s,
	}
}

func (v *Vector2d[T]) SubScalarInPlace(s T) *Vector2d[T] {
	v.X -= s
	v.Y -= s
	return v
}

func (v *Vector2d[T]) Mul(other Vector2d[T]) Vector2d[T] {
	return Vector2d[T]{
		X: v.X * other.X,
		Y: v.Y * other.Y,
	}
}

func (v *Vector2d[T]) MulInPlace(other Vector2d[T]) *Vector2d[T] {
	v.X *= other.X
	v.Y *= other.Y
	return v
}

func (v *Vector2d[T]) Scale(s T) Vector2d[T] {
	return Vector2d[T]{
		X: v.X * s,
		Y: v.Y * s,
	}
}

func (v *Vector2d[T]) ScaleInPlace(s T) *Vector2d[T] {
	v.X *= s
	v.Y *= s
	return v
}

func (v *Vector2d[T]) Div(other Vector2d[T]) Vector2d[T] {
	return Vector2d[T]{
		X: v.X / other.X,
		Y: v.Y / other.Y,
	}
}

func (v *Vector2d[T]) DivInPlace(other Vector2d[T]) *Vector2d[T] {
	v.X /= other.X
	v.Y /= other.Y

	return v
}

func (v *Vector2d[T]) DivScalar(s T) Vector2d[T] {
	return Vector2d[T]{
		X: v.X / s,
		Y: v.Y / s,
	}
}

func (v *Vector2d[T]) DivScalarInPlace(s T) *Vector2d[T] {
	v.X /= s
	v.Y /= s

	return v
}

func (v *Vector2d[T]) Equals(other Vector2d[T]) bool {
	return math.Abs(float64(v.X-other.X)) < 0.000001 && math.Abs(float64(v.Y-other.Y)) < 0.000001
}

func (v *Vector2d[T]) NotEquals(other Vector2d[T]) bool {
	return !(math.Abs(float64(v.X-other.X)) < 0.00001 && math.Abs(float64(v.Y-other.Y)) < 0.00001)
}

func (v *Vector2d[T]) Length() float64 {
	return math.Sqrt(float64(v.X*v.X + v.Y*v.Y))
}

func (v *Vector2d[T]) DistanceFrom(other Vector2d[T]) float64 {
	diff := Vector2d[T]{
		X: v.X - other.X,
		Y: v.Y - other.Y,
	}

	return diff.Length()
}

func (v *Vector2d[T]) Normalize() *Vector2d[T] {
	lengthSq := float64(v.X*v.X + v.Y*v.Y)
	if lengthSq == 0 {
		return v
	}

	inv := 1.0 / math.Sqrt(lengthSq)
	v.X = T(float64(v.X) * inv)
	v.Y = T(float64(v.Y) * inv)

	return v
}

func (v *Vector2d[T]) Set(nx, ny T) *Vector2d[T] {
	v.X = nx
	v.Y = ny

	return v
}

func (v *Vector2d[T]) RotateBy(degrees float64, center Vector2d[T]) *Vector2d[T] {
	rad := degrees * math.Pi / 180.0
	cs := math.Cos(rad)
	sn := math.Sin(rad)

	x := float64(v.X - center.X)
	y := float64(v.Y - center.Y)

	v.X = T(x*cs-y*sn) + center.X
	v.Y = T(x*sn+y*cs) + center.Y

	return v
}

func (v *Vector2d[T]) GetAngle() float64 {
	x, y := float64(v.X), float64(v.Y)
	if y == 0 {
		if x < 0 {
			return 180
		}
		return 0
	} else if x == 0 {
		if y < 0 {
			return 90
		}
		return 270
	}

	tmp := Clamp(y/math.Sqrt(x*x+y*y), -1.0, 1.0)
	angle := math.Atan(math.Sqrt(1-tmp*tmp)/tmp) * (180.0 / math.Pi)

	switch {
	case x > 0 && y > 0:
		return angle + 270
	case x > 0 && y < 0:
		return angle + 90
	case x < 0 && y < 0:
		return 90 - angle
	case x < 0 && y > 0:
		return 270 - angle
	}

	return angle
}

func (v *Vector2d[T]) LengthSQ() T {
	return v.X*v.X + v.Y*v.Y
}

func (v *Vector2d[T]) MidPoint(o Vector2d[T]) Vector2d[T] {
	return Vector2d[T]{
		X: (v.X + o.X) / 2,
		Y: (v.Y + o.Y) / 2,
	}
}

func (v *Vector2d[T]) Nor() Vector2d[T] {
	return Vector2d[T]{
		X: -v.Y,
		Y: v.X,
	}
}
