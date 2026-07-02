package vector2d

import (
	"math"
	"testing"
)

// --- Helper ---

func almostEqual(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

// --- Min / Max / Clamp ---

func TestMin(t *testing.T) {
	if Min(3, 5) != 3 {
		t.Error("Min(3,5) should be 3")
	}
	if Min(5, 3) != 3 {
		t.Error("Min(5,3) should be 3")
	}
	if Min(-1, 0) != -1 {
		t.Error("Min(-1,0) should be -1")
	}
}

func TestMax(t *testing.T) {
	if Max(3, 5) != 5 {
		t.Error("Max(3,5) should be 5")
	}
	if Max(5, 3) != 5 {
		t.Error("Max(5,3) should be 5")
	}
	if Max(-1, 0) != 0 {
		t.Error("Max(-1,0) should be 0")
	}
}

func TestClamp(t *testing.T) {
	if Clamp(5, 0, 10) != 5 {
		t.Error("Clamp(5,0,10) should be 5")
	}
	if Clamp(-5, 0, 10) != 0 {
		t.Error("Clamp(-5,0,10) should be 0")
	}
	if Clamp(15, 0, 10) != 10 {
		t.Error("Clamp(15,0,10) should be 10")
	}
}

// --- Constructor ---

func TestNew(t *testing.T) {
	v := New(3, 4)
	if v.X != 3 || v.Y != 4 {
		t.Errorf("New(3,4): got (%v,%v)", v.X, v.Y)
	}
}

// --- Negate ---

func TestNegate(t *testing.T) {
	v := New(3, -4)
	n := v.Negate()
	if n.X != -3 || n.Y != 4 {
		t.Errorf("Negate: got (%v,%v)", n.X, n.Y)
	}
}

// --- Add / Sub / Mul / Div ---

func TestAdd(t *testing.T) {
	a := New(1, 2)
	b := New(3, 4)
	r := a.Add(b)
	if r.X != 4 || r.Y != 6 {
		t.Errorf("Add: got (%v,%v)", r.X, r.Y)
	}
}

func TestAddInPlace(t *testing.T) {
	a := New(1, 2)
	a.AddInPlace(New(3, 4))
	if a.X != 4 || a.Y != 6 {
		t.Errorf("AddInPlace: got (%v,%v)", a.X, a.Y)
	}
}

func TestAddScalar(t *testing.T) {
	v := New(1, 2)
	r := v.AddScalar(10)
	if r.X != 11 || r.Y != 12 {
		t.Errorf("AddScalar: got (%v,%v)", r.X, r.Y)
	}
}

func TestSub(t *testing.T) {
	a := New(5, 7)
	b := New(3, 2)
	r := a.Sub(b)
	if r.X != 2 || r.Y != 5 {
		t.Errorf("Sub: got (%v,%v)", r.X, r.Y)
	}
}

func TestSubScalar(t *testing.T) {
	v := New(10, 7)
	r := v.SubScalar(3)
	if r.X != 7 || r.Y != 4 {
		t.Errorf("SubScalar: got (%v,%v)", r.X, r.Y)
	}
}

func TestMul(t *testing.T) {
	a := New(2, 3)
	b := New(4, 5)
	r := a.Mul(b)
	if r.X != 8 || r.Y != 15 {
		t.Errorf("Mul: got (%v,%v)", r.X, r.Y)
	}
}

func TestScale(t *testing.T) {
	v := New(3, 4)
	r := v.Scale(2)
	if r.X != 6 || r.Y != 8 {
		t.Errorf("Scale: got (%v,%v)", r.X, r.Y)
	}
}

func TestDiv(t *testing.T) {
	a := New(8, 9)
	b := New(4, 3)
	r := a.Div(b)
	if r.X != 2 || r.Y != 3 {
		t.Errorf("Div: got (%v,%v)", r.X, r.Y)
	}
}

func TestDivScalar(t *testing.T) {
	v := New(8, 6)
	r := v.DivScalar(2)
	if r.X != 4 || r.Y != 3 {
		t.Errorf("DivScalar: got (%v,%v)", r.X, r.Y)
	}
}

// --- In-place scalar variants ---

func TestAddScalarInPlace(t *testing.T) {
	v := New(1, 2)
	v.AddScalarInPlace(5)
	if v.X != 6 || v.Y != 7 {
		t.Errorf("AddScalarInPlace: got (%v,%v)", v.X, v.Y)
	}
}

func TestSubInPlace(t *testing.T) {
	v := New(5, 7)
	v.SubInPlace(New(2, 3))
	if v.X != 3 || v.Y != 4 {
		t.Errorf("SubInPlace: got (%v,%v)", v.X, v.Y)
	}
}

func TestSubScalarInPlace(t *testing.T) {
	v := New(10, 7)
	v.SubScalarInPlace(3)
	if v.X != 7 || v.Y != 4 {
		t.Errorf("SubScalarInPlace: got (%v,%v)", v.X, v.Y)
	}
}

func TestMulInPlace(t *testing.T) {
	v := New(2, 3)
	v.MulInPlace(New(4, 5))
	if v.X != 8 || v.Y != 15 {
		t.Errorf("MulInPlace: got (%v,%v)", v.X, v.Y)
	}
}

func TestScaleInPlace(t *testing.T) {
	v := New(3, 4)
	v.ScaleInPlace(2)
	if v.X != 6 || v.Y != 8 {
		t.Errorf("ScaleInPlace: got (%v,%v)", v.X, v.Y)
	}
}

func TestDivInPlace(t *testing.T) {
	v := New(8, 9)
	v.DivInPlace(New(4, 3))
	if v.X != 2 || v.Y != 3 {
		t.Errorf("DivInPlace: got (%v,%v)", v.X, v.Y)
	}
}

func TestDivScalarInPlace(t *testing.T) {
	v := New(8, 6)
	v.DivScalarInPlace(2)
	if v.X != 4 || v.Y != 3 {
		t.Errorf("DivScalarInPlace: got (%v,%v)", v.X, v.Y)
	}
}

// --- Equals / NotEquals ---

func TestEquals(t *testing.T) {
	a := New(1.0, 2.0)
	b := New(1.0, 2.0)
	if !a.Equals(b) {
		t.Error("Equals: identical vectors should be equal")
	}
	c := New(1.0, 3.0)
	if a.Equals(c) {
		t.Error("Equals: different vectors should not be equal")
	}
}

func TestNotEquals(t *testing.T) {
	a := New(1.0, 2.0)
	b := New(1.0, 3.0)
	if !a.NotEquals(b) {
		t.Error("NotEquals: different vectors should not be equal")
	}
	c := New(1.0, 2.0)
	if a.NotEquals(c) {
		t.Error("NotEquals: same vectors should be equal")
	}
}

// --- Length / LengthSQ / DistanceFrom ---

func TestLength(t *testing.T) {
	v := New(3.0, 4.0)
	if !almostEqual(v.Length(), 5.0, 1e-9) {
		t.Errorf("Length: expected 5, got %v", v.Length())
	}
}

func TestLengthZero(t *testing.T) {
	v := New(0.0, 0.0)
	if v.Length() != 0 {
		t.Errorf("Length of zero vector should be 0, got %v", v.Length())
	}
}

func TestLengthSQ(t *testing.T) {
	v := New(3, 4)
	if v.LengthSQ() != 25 {
		t.Errorf("LengthSQ: expected 25, got %v", v.LengthSQ())
	}
}

func TestDistanceFrom(t *testing.T) {
	a := New(0.0, 0.0)
	b := New(3.0, 4.0)
	if !almostEqual(a.DistanceFrom(b), 5.0, 1e-9) {
		t.Errorf("DistanceFrom: expected 5, got %v", a.DistanceFrom(b))
	}
}

// --- Normalize ---

func TestNormalize(t *testing.T) {
	v := New(3.0, 4.0)
	v.Normalize()
	if !almostEqual(v.Length(), 1.0, 1e-9) {
		t.Errorf("Normalize: length should be 1, got %v", v.Length())
	}
}

func TestNormalizeZeroVector(t *testing.T) {
	v := New(0.0, 0.0)
	v.Normalize() // should not panic or modify
	if v.X != 0 || v.Y != 0 {
		t.Error("Normalizing zero vector should leave it unchanged")
	}
}

// --- Set ---

func TestSet(t *testing.T) {
	v := New(1, 2)
	v.Set(10, 20)
	if v.X != 10 || v.Y != 20 {
		t.Errorf("Set: got (%v,%v)", v.X, v.Y)
	}
}

// --- MidPoint ---

func TestMidPoint(t *testing.T) {
	a := New(0.0, 0.0)
	b := New(4.0, 6.0)
	m := a.MidPoint(b)
	if !almostEqual(m.X, 2.0, 1e-9) || !almostEqual(m.Y, 3.0, 1e-9) {
		t.Errorf("MidPoint: expected (2,3), got (%v,%v)", m.X, m.Y)
	}
}

// --- Nor (perpendicular) ---

func TestNor(t *testing.T) {
	v := New(3.0, 4.0)
	n := v.Nor()
	// Nor returns (-Y, X), dot product with original should be 0
	dot := v.X*n.X + v.Y*n.Y
	if !almostEqual(dot, 0.0, 1e-9) {
		t.Errorf("Nor: should be perpendicular, dot=%v", dot)
	}
	if n.X != -v.Y || n.Y != v.X {
		t.Errorf("Nor: expected (%v,%v), got (%v,%v)", -v.Y, v.X, n.X, n.Y)
	}
}

// --- RotateBy ---

func TestRotateBy90(t *testing.T) {
	v := New(1.0, 0.0)
	center := New(0.0, 0.0)
	v.RotateBy(90, center)
	// After 90° rotation: (0, 1)
	if !almostEqual(v.X, 0.0, 1e-6) || !almostEqual(v.Y, 1.0, 1e-6) {
		t.Errorf("RotateBy 90°: expected (0,1), got (%v,%v)", v.X, v.Y)
	}
}

func TestRotateBy180(t *testing.T) {
	v := New(1.0, 0.0)
	center := New(0.0, 0.0)
	v.RotateBy(180, center)
	if !almostEqual(v.X, -1.0, 1e-6) || !almostEqual(v.Y, 0.0, 1e-6) {
		t.Errorf("RotateBy 180°: expected (-1,0), got (%v,%v)", v.X, v.Y)
	}
}

func TestRotateByAroundCenter(t *testing.T) {
	v := New(2.0, 0.0)
	center := New(1.0, 0.0)
	v.RotateBy(90, center)
	// Rotating (2,0) around (1,0) by 90° → (1,1)
	if !almostEqual(v.X, 1.0, 1e-6) || !almostEqual(v.Y, 1.0, 1e-6) {
		t.Errorf("RotateBy around center: expected (1,1), got (%v,%v)", v.X, v.Y)
	}
}

// --- GetAngle ---

func TestGetAngleRight(t *testing.T) {
	v := New(1.0, 0.0) // positive X → 0°
	if !almostEqual(v.GetAngle(), 0, 1e-6) {
		t.Errorf("GetAngle right: expected 0, got %v", v.GetAngle())
	}
}

func TestGetAngleLeft(t *testing.T) {
	v := New(-1.0, 0.0) // negative X → 180°
	if !almostEqual(v.GetAngle(), 180, 1e-6) {
		t.Errorf("GetAngle left: expected 180, got %v", v.GetAngle())
	}
}

func TestGetAngleUp(t *testing.T) {
	v := New(0.0, -1.0) // negative Y → 90°
	if !almostEqual(v.GetAngle(), 90, 1e-6) {
		t.Errorf("GetAngle up: expected 90, got %v", v.GetAngle())
	}
}

func TestGetAngleDown(t *testing.T) {
	v := New(0.0, 1.0) // positive Y → 270°
	if !almostEqual(v.GetAngle(), 270, 1e-6) {
		t.Errorf("GetAngle down: expected 270, got %v", v.GetAngle())
	}
}

// --- Type aliases smoke test ---

func TestTypeAliases(t *testing.T) {
	var vi Vector2di = New[int32](1, 2)
	var vf Vector2df = New[float32](1.5, 2.5)
	var vd Vector2dd = New[float64](1.5, 2.5)

	if vi.X != 1 || vi.Y != 2 {
		t.Errorf("Vector2di: got (%v,%v)", vi.X, vi.Y)
	}
	if !almostEqual(float64(vf.X), 1.5, 1e-5) {
		t.Errorf("Vector2df: got %v", vf.X)
	}
	if !almostEqual(vd.X, 1.5, 1e-9) {
		t.Errorf("Vector2dd: got %v", vd.X)
	}
}
