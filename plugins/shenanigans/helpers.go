//nolint:mnd // Visual effects plugin: magic numbers are by design.
package main

// Shared math helpers used across multiple effects.

// sinApprox is a fast sine approximation (Bhaskara I's formula) avoiding math import.
func sinApprox(x float64) float64 {
	// Normalize to [0, 2π)
	const twoPi = 6.283185307
	const pi = 3.141592654

	for x < 0 {
		x += twoPi
	}
	for x >= twoPi {
		x -= twoPi
	}

	// Map to [0, π] with sign
	sign := 1.0
	if x > pi {
		x -= pi
		sign = -1.0
	}

	// Bhaskara I's approximation: sin(x) ≈ 16x(π-x) / (5π²-4x(π-x))
	num := 16 * x * (pi - x)
	den := 5*pi*pi - 4*x*(pi-x)
	return sign * num / den
}

// absF returns the absolute value of a float64.
func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// sinNorm returns sinApprox mapped to 0.0-1.0 range.
func sinNorm(x float64) float64 {
	return (sinApprox(x) + 1.0) / 2.0
}

// cosApprox uses sinApprox shifted by pi/2.
func cosApprox(x float64) float64 {
	return sinApprox(x + 1.5707963)
}

// atan2Approx is a rough atan2 approximation.
func atan2Approx(y, x float64) float64 {
	// Simple approximation using the identity and sinApprox
	const pi = 3.14159265
	if x == 0 {
		if y > 0 {
			return pi / 2
		}
		return -pi / 2
	}
	a := y / x
	// Clamp for stability
	if a > 10 {
		a = 10
	} else if a < -10 {
		a = -10
	}
	// Polynomial approximation of atan for small values
	result := a / (1 + 0.28*a*a)
	if x < 0 {
		if y >= 0 {
			result += pi
		} else {
			result -= pi
		}
	}
	return result
}
