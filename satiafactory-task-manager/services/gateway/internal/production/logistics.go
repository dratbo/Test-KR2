package production

import "strconv"

// Belt rates (items/min) for Satisfactory v1.0.
var beltRates = map[int]float64{
	1: 60,
	2: 120,
	3: 270,
	4: 480,
	5: 780,
	6: 1200,
}

// Pipe rates (m³/min) for fluids.
var pipeRates = map[int]float64{
	1: 300,
	2: 600,
}

// LogisticsParams holds player belt/pipe infrastructure.
type LogisticsParams struct {
	ConveyorMk int
	PipeMk     int
}

func NormalizeConveyorMk(mk int) int {
	if mk < 1 {
		return 0
	}
	if mk > 6 {
		return 6
	}
	return mk
}

func NormalizePipeMk(mk int) int {
	if mk < 1 {
		return 0
	}
	if mk > 2 {
		return 2
	}
	return mk
}

func (p LogisticsParams) BeltRate() float64 {
	if p.ConveyorMk <= 0 {
		return 0
	}
	return beltRates[p.ConveyorMk]
}

func (p LogisticsParams) PipeRate() float64 {
	if p.PipeMk <= 0 {
		return 0
	}
	return pipeRates[p.PipeMk]
}

func (p LogisticsParams) Configured() bool {
	return p.ConveyorMk > 0 && p.PipeMk > 0
}

// ConveyorLabel returns a Russian label for UI.
func ConveyorLabel(mk int) string {
	if mk <= 0 {
		return ""
	}
	return "Конвейер Mk." + strconv.Itoa(mk) + " (" + formatRate(beltRates[mk]) + " предм./мин)"
}

// PipeLabel returns a Russian label for UI.
func PipeLabel(mk int) string {
	if mk <= 0 {
		return ""
	}
	return "Труба Mk." + strconv.Itoa(mk) + " (" + formatRate(pipeRates[mk]) + " м³/мин)"
}

func formatRate(v float64) string {
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}

// ApplyBeltCap splits machine slots so each belt line does not exceed belt capacity.
func ApplyBeltCap(slots []MachineSlot, beltCapacity float64) []MachineSlot {
	if beltCapacity <= 0 || len(slots) == 0 {
		return slots
	}
	var out []MachineSlot
	for _, s := range slots {
		if s.RatePerMachine <= 0 {
			out = append(out, s)
			continue
		}
		maxPerLine := int(beltCapacity / s.RatePerMachine)
		if maxPerLine < 1 {
			maxPerLine = 1
		}
		remaining := s.Count
		for remaining > 0 {
			n := remaining
			if n > maxPerLine {
				n = maxPerLine
			}
			out = append(out, MachineSlot{
				Count:          n,
				Shards:         s.Shards,
				ClockPercent:   s.ClockPercent,
				RatePerMachine: s.RatePerMachine,
			})
			remaining -= n
		}
	}
	return out
}

// BeltLinesNeeded counts conveyor lines for slots at the given belt capacity.
func BeltLinesNeeded(slots []MachineSlot, beltCapacity float64) int {
	if beltCapacity <= 0 {
		return 0
	}
	capped := ApplyBeltCap(slots, beltCapacity)
	return len(capped)
}
