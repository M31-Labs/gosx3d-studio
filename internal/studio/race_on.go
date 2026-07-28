//go:build race

package studio

// raceEnabled reports whether this binary was built with -race. Race
// instrumentation multiplies memory-access cost, so a wall-clock budget gate
// measures the instrumentation rather than the code under it.
const raceEnabled = true
