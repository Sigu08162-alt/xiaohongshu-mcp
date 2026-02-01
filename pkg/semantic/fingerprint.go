package semantic

type Fingerprint struct {
	VisibleCount   int
	ButtonCount    int
	InputCount     int
	ContainerCount int
}

type FingerprintDelta struct {
	VisibleRatio   float64
	ButtonRatio    float64
	InputRatio     float64
	ContainerRatio float64
}

func (f Fingerprint) Delta(other Fingerprint) FingerprintDelta {
	if f.VisibleCount == 0 {
		return FingerprintDelta{}
	}
	return FingerprintDelta{
		VisibleRatio:   float64(other.VisibleCount-f.VisibleCount) / float64(f.VisibleCount),
		ButtonRatio:    ratioDelta(f.ButtonCount, other.ButtonCount),
		InputRatio:     ratioDelta(f.InputCount, other.InputCount),
		ContainerRatio: ratioDelta(f.ContainerCount, other.ContainerCount),
	}
}

func (d FingerprintDelta) Above(threshold float64) bool {
	return d.VisibleRatio >= threshold
}

func (d FingerprintDelta) ChangedMetricCount(ratioThreshold float64) int {
	count := 0
	if abs(d.VisibleRatio) >= ratioThreshold {
		count++
	}
	if abs(d.ButtonRatio) >= ratioThreshold {
		count++
	}
	if abs(d.InputRatio) >= ratioThreshold {
		count++
	}
	if abs(d.ContainerRatio) >= ratioThreshold {
		count++
	}
	return count
}

func ratioDelta(before, after int) float64 {
	if before == 0 {
		return 0
	}
	return float64(after-before) / float64(before)
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
