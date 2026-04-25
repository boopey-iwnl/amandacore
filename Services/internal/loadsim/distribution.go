package loadsim

import (
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

type ZoneDistributionPlan struct {
	Mode    string         `json:"mode"`
	Weights map[string]int `json:"weights,omitempty"`
	Single  string         `json:"single,omitempty"`
}

func ParseZoneDistribution(input string, availableZones []string) (ZoneDistributionPlan, error) {
	available := map[string]struct{}{}
	for _, zoneID := range availableZones {
		available[zoneID] = struct{}{}
	}
	raw := strings.TrimSpace(input)
	if raw == "" || raw == "even" {
		return ZoneDistributionPlan{Mode: "even"}, nil
	}
	if raw == "transition-heavy" {
		return ZoneDistributionPlan{Mode: "transition-heavy"}, nil
	}
	if strings.HasPrefix(raw, "single:") {
		zoneID := strings.TrimSpace(strings.TrimPrefix(raw, "single:"))
		if _, ok := available[zoneID]; !ok {
			return ZoneDistributionPlan{}, fmt.Errorf("zone %s is not in content package", zoneID)
		}
		return ZoneDistributionPlan{Mode: "single", Single: zoneID}, nil
	}
	if strings.HasPrefix(raw, "weighted:") {
		weights := map[string]int{}
		for _, part := range strings.Split(strings.TrimPrefix(raw, "weighted:"), ",") {
			pieces := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(pieces) != 2 {
				return ZoneDistributionPlan{}, fmt.Errorf("invalid weighted distribution segment %q", part)
			}
			zoneID := strings.TrimSpace(pieces[0])
			if _, ok := available[zoneID]; !ok {
				return ZoneDistributionPlan{}, fmt.Errorf("zone %s is not in content package", zoneID)
			}
			weight, err := strconv.Atoi(strings.TrimSpace(pieces[1]))
			if err != nil || weight <= 0 {
				return ZoneDistributionPlan{}, fmt.Errorf("zone %s weight must be a positive integer", zoneID)
			}
			weights[zoneID] = weight
		}
		if len(weights) == 0 {
			return ZoneDistributionPlan{}, fmt.Errorf("weighted distribution needs at least one zone")
		}
		return ZoneDistributionPlan{Mode: "weighted", Weights: weights}, nil
	}
	return ZoneDistributionPlan{}, fmt.Errorf("unsupported zone distribution %q", input)
}

func AssignClientZones(plan ZoneDistributionPlan, zoneOrder []string, clients int, rng *rand.Rand) ([]string, error) {
	if clients <= 0 {
		return nil, fmt.Errorf("clients must be greater than zero")
	}
	if len(zoneOrder) == 0 {
		return nil, fmt.Errorf("at least one zone is required")
	}
	assignments := make([]string, clients)
	switch plan.Mode {
	case "", "even", "transition-heavy":
		for index := range assignments {
			assignments[index] = zoneOrder[index%len(zoneOrder)]
		}
	case "single":
		for index := range assignments {
			assignments[index] = plan.Single
		}
	case "weighted":
		total := 0
		for _, weight := range plan.Weights {
			total += weight
		}
		if total <= 0 {
			return nil, fmt.Errorf("weighted distribution total must be positive")
		}
		for index := range assignments {
			roll := rng.Intn(total)
			cursor := 0
			for _, zoneID := range zoneOrder {
				weight := plan.Weights[zoneID]
				if weight <= 0 {
					continue
				}
				cursor += weight
				if roll < cursor {
					assignments[index] = zoneID
					break
				}
			}
		}
	default:
		return nil, fmt.Errorf("unsupported distribution mode %q", plan.Mode)
	}
	return assignments, nil
}
