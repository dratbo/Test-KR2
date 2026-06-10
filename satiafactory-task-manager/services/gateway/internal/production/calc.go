package production

import (
	"math"
	"sort"
)

// ClockPercent returns production clock speed for 0–3 power modules.
func ClockPercent(shardCount int) int {
	if shardCount < 0 {
		shardCount = 0
	}
	if shardCount > 3 {
		shardCount = 3
	}
	return 100 + shardCount*50
}

// ItemsPerMinute calculates output rate for a recipe at a given clock speed.
func ItemsPerMinute(productPerCycle, durationSec float64, clockPercent int) float64 {
	if durationSec <= 0 || productPerCycle <= 0 {
		return 0
	}
	return (productPerCycle / durationSec) * 60 * (float64(clockPercent) / 100)
}

// MachineSlot describes a group of identical machines with the same clock speed.
type MachineSlot struct {
	Count          int
	Shards         int
	ClockPercent   int
	RatePerMachine float64
}

// TotalMachines returns the sum of machine counts in slots.
func TotalMachines(slots []MachineSlot) int {
	n := 0
	for _, s := range slots {
		n += s.Count
	}
	return n
}

// TotalRate returns combined output rate across all slots.
func TotalRate(slots []MachineSlot) float64 {
	var sum float64
	for _, s := range slots {
		sum += s.RatePerMachine * float64(s.Count)
	}
	return sum
}

// TotalShardsUsed returns total power modules across all machine slots.
func TotalShardsUsed(slots []MachineSlot) int {
	n := 0
	for _, s := range slots {
		n += s.Shards * s.Count
	}
	return n
}

// AllocateWithoutShards distributes required throughput across machines at ≤100% clock.
func AllocateWithoutShards(requiredRate, baseRate float64) []MachineSlot {
	if requiredRate <= 0 || baseRate <= 0 {
		return nil
	}
	full := int(math.Floor(requiredRate / baseRate))
	remainder := requiredRate - float64(full)*baseRate

	var slots []MachineSlot
	if full > 0 {
		slots = append(slots, MachineSlot{
			Count:          full,
			Shards:         0,
			ClockPercent:   100,
			RatePerMachine: baseRate,
		})
	}
	if remainder > 0.01 {
		clock := int(math.Ceil(remainder / baseRate * 100))
		if clock < 1 {
			clock = 1
		}
		if clock > 100 {
			clock = 100
		}
		slots = append(slots, MachineSlot{
			Count:          1,
			Shards:         0,
			ClockPercent:   clock,
			RatePerMachine: baseRate * float64(clock) / 100,
		})
	}
	return slots
}

// AllocateWithShards minimizes machine count using up to 3 power modules per building.
func AllocateWithShards(requiredRate, baseRate float64) []MachineSlot {
	return allocateWithShardBudget(requiredRate, baseRate, math.MaxInt32)
}

// AllocateWithShardBudget minimizes machines using at most shardBudget power modules total.
func AllocateWithShardBudget(requiredRate, baseRate float64, shardBudget int) []MachineSlot {
	if shardBudget < 0 {
		shardBudget = 0
	}
	return allocateWithShardBudget(requiredRate, baseRate, shardBudget)
}

func allocateWithShardBudget(requiredRate, baseRate float64, shardBudget int) []MachineSlot {
	if requiredRate <= 0 || baseRate <= 0 {
		return nil
	}

	counts := map[int]int{}
	remaining := requiredRate
	budget := shardBudget

	for _, shards := range []int{3, 2, 1} {
		rate := baseRate * float64(ClockPercent(shards)) / 100
		for remaining >= rate-0.001 && budget >= shards {
			counts[shards]++
			remaining -= rate
			budget -= shards
		}
	}

	var slots []MachineSlot
	for shards := 3; shards >= 1; shards-- {
		if counts[shards] == 0 {
			continue
		}
		clock := ClockPercent(shards)
		slots = append(slots, MachineSlot{
			Count:          counts[shards],
			Shards:         shards,
			ClockPercent:   clock,
			RatePerMachine: baseRate * float64(clock) / 100,
		})
	}

	if remaining > 0.01 {
		zeroSlots := AllocateWithoutShards(remaining, baseRate)
		slots = append(slots, zeroSlots...)
	}

	return slots
}

// RateTableEntry is one row in the per-clock-speed rate table.
type RateTableEntry struct {
	Shards       int
	ClockPercent int
	ItemsPerMin  float64
}

// BuildRateTable returns production rates for 0–3 power modules.
func BuildRateTable(productPerCycle, durationSec, baseRate float64) []RateTableEntry {
	var rows []RateTableEntry
	for shards := 0; shards <= 3; shards++ {
		clock := ClockPercent(shards)
		rate := baseRate
		if productPerCycle > 0 && durationSec > 0 {
			rate = ItemsPerMinute(productPerCycle, durationSec, clock)
		} else {
			rate = ItemsPerMinute(1, 1, clock) * baseRate
		}
		rows = append(rows, RateTableEntry{
			Shards:       shards,
			ClockPercent: clock,
			ItemsPerMin:  round2(rate),
		})
	}
	return rows
}

// Scenario describes one machine allocation strategy.
type Scenario struct {
	Name          string
	Slots         []MachineSlot
	TotalMachines int
	TotalRate     float64
	ShardsUsed    int
}

// BuildScenarios creates standard allocation scenarios for a production step.
func BuildScenarios(requiredRate, baseRate float64, allowShards bool, shardBudget int) []Scenario {
	var scenarios []Scenario

	scenarios = append(scenarios, Scenario{
		Name:          "Одна постройка (100%)",
		Slots:         []MachineSlot{{Count: 1, Shards: 0, ClockPercent: 100, RatePerMachine: baseRate}},
		TotalMachines: 1,
		TotalRate:     baseRate,
		ShardsUsed:    0,
	})

	noShardSlots := AllocateWithoutShards(requiredRate, baseRate)
	if len(noShardSlots) > 0 {
		scenarios = append(scenarios, Scenario{
			Name:          "Без энергомодулей",
			Slots:         noShardSlots,
			TotalMachines: TotalMachines(noShardSlots),
			TotalRate:     round2(TotalRate(noShardSlots)),
			ShardsUsed:    0,
		})
	}

	if allowShards {
		if shardBudget > 0 {
			budgetSlots := AllocateWithShardBudget(requiredRate, baseRate, shardBudget)
			if len(budgetSlots) > 0 {
				scenarios = append(scenarios, Scenario{
					Name:          "С вашими энергомодулями",
					Slots:         budgetSlots,
					TotalMachines: TotalMachines(budgetSlots),
					TotalRate:     round2(TotalRate(budgetSlots)),
					ShardsUsed:    TotalShardsUsed(budgetSlots),
				})
			}
		} else {
			shardSlots := AllocateWithShards(requiredRate, baseRate)
			if len(shardSlots) > 0 {
				scenarios = append(scenarios, Scenario{
					Name:          "С энергомодулями (оптимально)",
					Slots:         shardSlots,
					TotalMachines: TotalMachines(shardSlots),
					TotalRate:     round2(TotalRate(shardSlots)),
					ShardsUsed:    TotalShardsUsed(shardSlots),
				})
			}
		}
	}

	return scenarios
}

// PickScenario selects the scenario used for building cost calculation.
func PickScenario(scenarios []Scenario, shardBudget int) *Scenario {
	if len(scenarios) == 0 {
		return nil
	}
	if shardBudget > 0 {
		for i := range scenarios {
			if scenarios[i].Name == "С вашими энергомодулями" {
				return &scenarios[i]
			}
		}
	}
	for i := range scenarios {
		if scenarios[i].Name == "Без энергомодулей" {
			return &scenarios[i]
		}
	}
	return &scenarios[len(scenarios)-1]
}

// DistributeShardBudget splits total shards across overclockable steps.
func DistributeShardBudget(stepCount int, overclockable []bool, totalShards int) []int {
	budgets := make([]int, stepCount)
	var indices []int
	for i, ok := range overclockable {
		if ok {
			indices = append(indices, i)
		}
	}
	if len(indices) == 0 || totalShards <= 0 {
		return budgets
	}
	per := totalShards / len(indices)
	rem := totalShards % len(indices)
	for j, idx := range indices {
		budgets[idx] = per
		if j < rem {
			budgets[idx]++
		}
	}
	return budgets
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// SortSlotsByRate sorts machine slots by descending rate (for stable display).
func SortSlotsByRate(slots []MachineSlot) {
	sort.Slice(slots, func(i, j int) bool {
		return slots[i].RatePerMachine > slots[j].RatePerMachine
	})
}
