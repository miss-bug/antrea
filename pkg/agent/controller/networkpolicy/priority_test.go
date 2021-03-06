// Copyright 2020 Antrea Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package networkpolicy

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/antrea/pkg/agent/types"
)

var (
	p110  = types.Priority{TierPriority: 1, PolicyPriority: 1, RulePriority: 0}
	p1120 = types.Priority{TierPriority: 1, PolicyPriority: 1.2, RulePriority: 0}
	p1121 = types.Priority{TierPriority: 1, PolicyPriority: 1.2, RulePriority: 1}
	p1130 = types.Priority{TierPriority: 1, PolicyPriority: 1.3, RulePriority: 0}
	p1131 = types.Priority{TierPriority: 1, PolicyPriority: 1.3, RulePriority: 1}
	p1140 = types.Priority{TierPriority: 1, PolicyPriority: 1.4, RulePriority: 0}
	p1141 = types.Priority{TierPriority: 1, PolicyPriority: 1.4, RulePriority: 1}
	p1160 = types.Priority{TierPriority: 1, PolicyPriority: 1.6, RulePriority: 0}
	p1161 = types.Priority{TierPriority: 1, PolicyPriority: 1.6, RulePriority: 1}
	p190  = types.Priority{TierPriority: 1, PolicyPriority: 9, RulePriority: 0}
	p191  = types.Priority{TierPriority: 1, PolicyPriority: 9, RulePriority: 1}
	p192  = types.Priority{TierPriority: 1, PolicyPriority: 9, RulePriority: 2}
	p193  = types.Priority{TierPriority: 1, PolicyPriority: 9, RulePriority: 3}
)

func TestUpdatePriorityAssignment(t *testing.T) {
	tests := []struct {
		name                string
		argsPriorities      []types.Priority
		argsOFPriorities    []uint16
		expectedPriorityMap map[types.Priority]uint16
		expectedOFMap       map[uint16]types.Priority
		expectedSorted      []uint16
	}{
		{
			"in-order",
			[]types.Priority{p110, p1120, p1121},
			[]uint16{10000, 9999, 9998},
			map[types.Priority]uint16{p110: 10000, p1120: 9999, p1121: 9998},
			map[uint16]types.Priority{10000: p110, 9999: p1120, 9998: p1121},
			[]uint16{9998, 9999, 10000},
		},
		{
			"reverse-order",
			[]types.Priority{p1121, p1120, p110},
			[]uint16{9998, 9999, 10000},
			map[types.Priority]uint16{p110: 10000, p1120: 9999, p1121: 9998},
			map[uint16]types.Priority{10000: p110, 9999: p1120, 9998: p1121},
			[]uint16{9998, 9999, 10000},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pa := newPriorityAssigner(InitialOFPriority, true)
			for i := 0; i < len(tt.argsPriorities); i++ {
				pa.updatePriorityAssignment(tt.argsOFPriorities[i], tt.argsPriorities[i])
			}
			assert.Equalf(t, tt.expectedPriorityMap, pa.priorityMap, "Got unexpected priorityMap")
			assert.Equalf(t, tt.expectedOFMap, pa.ofPriorityMap, "Got unexpected ofPriorityMap")
			assert.Equalf(t, tt.expectedSorted, pa.sortedOFPriorities, "Got unexpected sortedOFPriorities")
		})
	}
}

func TestGetInsertionPoint(t *testing.T) {
	tests := []struct {
		name                 string
		argsPriorities       []types.Priority
		argsOFPriorities     []uint16
		insertingPriority    types.Priority
		initialOFPriority    uint16
		expectInsertionPoint uint16
		expectOccupied       bool
	}{
		{
			"spot-on",
			[]types.Priority{},
			[]uint16{},
			p110,
			10000,
			10000,
			false,
		},
		{
			"stepped-on-toes-lower",
			[]types.Priority{p110},
			[]uint16{10000},
			p1120,
			10000,
			9999,
			false,
		},
		{
			"stepped-on-toes-higher",
			[]types.Priority{p1120},
			[]uint16{10000},
			p110,
			10000,
			10001,
			false,
		},
		{
			"search-up",
			[]types.Priority{p1120, p1121, p1130, p1131},
			[]uint16{10000, 9999, 9998, 9997},
			p110,
			9998,
			10001,
			false,
		},
		{
			"search-down",
			[]types.Priority{p1120, p1121, p1130},
			[]uint16{10000, 9999, 9998},
			p1131,
			10000,
			9997,
			false,
		},
		{
			"find-insertion-up",
			[]types.Priority{p110, p1120, p1130, p1131},
			[]uint16{10000, 9999, 9998, 9997},
			p1121,
			9997,
			9999,
			true,
		},
		{
			"find-insertion-down",
			[]types.Priority{p110, p1120, p1130, p1131},
			[]uint16{10000, 9999, 9998, 9997},
			p1121,
			10000,
			9999,
			true,
		},
		{
			"upper-bound",
			[]types.Priority{p1120, p1121, p1130},
			[]uint16{PolicyTopPriority, PolicyTopPriority - 1, PolicyTopPriority - 2},
			p110,
			PolicyTopPriority - 2,
			PolicyTopPriority + 1,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pa := newPriorityAssigner(func(p types.Priority, isSingleTier bool) uint16 {
				return tt.initialOFPriority
			}, true)
			for i := 0; i < len(tt.argsPriorities); i++ {
				pa.updatePriorityAssignment(tt.argsOFPriorities[i], tt.argsPriorities[i])
			}
			got, occupied := pa.getInsertionPoint(tt.insertingPriority)
			assert.Equalf(t, tt.expectInsertionPoint, got, "Got unexpected insertion point")
			assert.Equalf(t, tt.expectOccupied, occupied, "Insertion point occupied status is unexpected")
		})
	}
}

func TestReassignPriorities(t *testing.T) {

	tests := []struct {
		name                string
		argsPriorities      []types.Priority
		argsOFPriorities    []uint16
		insertingPriorities []types.Priority
		insertionPoints     []uint16
		expectedAssigned    []uint16
		expectedUpdates     []map[uint16]uint16
	}{
		{
			"sift-down-at-upper-bound",
			[]types.Priority{p191, p193},
			[]uint16{PolicyTopPriority, PolicyTopPriority - 1},
			[]types.Priority{p190, p192},
			[]uint16{PolicyTopPriority + 1, PolicyTopPriority - 1},
			[]uint16{PolicyTopPriority, PolicyTopPriority - 2},
			[]map[uint16]uint16{
				{
					PolicyTopPriority:     PolicyTopPriority - 1,
					PolicyTopPriority - 1: PolicyTopPriority - 2,
				},
				{
					PolicyTopPriority - 2: PolicyTopPriority - 3,
				},
			},
		},
		{
			"sift-up-at-lower-bound",
			[]types.Priority{p1130, p1120},
			[]uint16{PolicyBottomPriority, PolicyBottomPriority + 1},
			[]types.Priority{p1121, p1131},
			[]uint16{PolicyBottomPriority + 1, PolicyBottomPriority},
			[]uint16{PolicyBottomPriority + 1, PolicyBottomPriority},
			[]map[uint16]uint16{
				{
					PolicyBottomPriority + 1: PolicyBottomPriority + 2,
				},
				{
					PolicyBottomPriority:     PolicyBottomPriority + 1,
					PolicyBottomPriority + 1: PolicyBottomPriority + 2,
					PolicyBottomPriority + 2: PolicyBottomPriority + 3,
				},
			},
		},
		{
			"sift-based-on-cost",
			[]types.Priority{p110, p1121, p1131},
			[]uint16{10000, 9999, 9998},
			[]types.Priority{p1130, p1120},
			[]uint16{9999, 10000},
			[]uint16{9998, 10000},
			[]map[uint16]uint16{
				{9998: 9997}, {10000: 10001},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pa := newPriorityAssigner(InitialOFPriority, true)
			for i := 0; i < len(tt.argsPriorities); i++ {
				pa.updatePriorityAssignment(tt.argsOFPriorities[i], tt.argsPriorities[i])
			}
			for i := 0; i < len(tt.insertingPriorities); i++ {
				got, updates, _, err := pa.reassignPriorities(tt.insertionPoints[i], tt.insertingPriorities[i])
				assert.Equalf(t, err, nil, "Error occurred in reassigning priorities")
				assert.Equalf(t, tt.expectedAssigned[i], *got, "Got unexpected assigned priority")
				assert.Equalf(t, tt.expectedUpdates[i], updates, "Got unexpected priority updates")
			}
		})
	}
}

func TestRegisterPrioritiesAndRelease(t *testing.T) {
	pa := newPriorityAssigner(InitialOFPriority, true)
	priorites := []types.Priority{p1160, p1141, p1140, p1130, p1121, p1120, p110}
	err := pa.RegisterPriorities(priorites)
	assert.Equalf(t, err, nil, "Error occurred in registering priorities")
	expectedPriorityMap := map[types.Priority]uint16{}
	ofPriorities := make([]uint16, len(priorites))
	for i, p := range priorites {
		ofPriority := pa.initialOFPriorityFunc(p, true)
		expectedPriorityMap[p] = ofPriority
		ofPriorities[i] = ofPriority
	}
	assert.Equalf(t, expectedPriorityMap, pa.priorityMap, "Got unexpected priorityMap")

	pa.Release(ofPriorities[0])
	pa.Release(ofPriorities[2])
	pa.Release(ofPriorities[5])
	expectedOFMap := map[uint16]types.Priority{
		ofPriorities[1]: p1141, ofPriorities[3]: p1130, ofPriorities[4]: p1121, ofPriorities[6]: p110,
	}
	expectedPriorityMap = map[types.Priority]uint16{
		p110: ofPriorities[6], p1121: ofPriorities[4], p1130: ofPriorities[3], p1141: ofPriorities[1],
	}
	expectedSorted := []uint16{ofPriorities[1], ofPriorities[3], ofPriorities[4], ofPriorities[6]}
	assert.Equalf(t, expectedOFMap, pa.ofPriorityMap, "Got unexpected priorityMap")
	assert.Equalf(t, expectedPriorityMap, pa.priorityMap, "Got unexpected ofPriorityMap")
	assert.Equalf(t, expectedSorted, pa.sortedOFPriorities, "Got unexpected sortedOFPriorities")
}

func TestRevertUpdates(t *testing.T) {
	tests := []struct {
		name                string
		insertionPoint      uint16
		extraPriority       types.Priority
		originalPriorityMap map[types.Priority]uint16
		originalOFMap       map[uint16]types.Priority
		originalSorted      []uint16
	}{
		{
			"single-update-up",
			9999,
			p1121,
			map[types.Priority]uint16{p1120: 9999, p1130: 9998},
			map[uint16]types.Priority{9999: p1120, 9998: p1130},
			[]uint16{9998, 9999},
		},
		{
			"multiple-updates-up",
			9997,
			p1131,
			map[types.Priority]uint16{
				p1120: 9999, p1121: 9998, p1130: 9997, p1140: 9996, p1141: 9995, p1160: 9994, p1161: 9993},
			map[uint16]types.Priority{
				9999: p1120, 9998: p1121, 9997: p1130, 9996: p1140, 9995: p1141, 9994: p1160, 9993: p1161},
			[]uint16{9993, 9994, 9995, 9996, 9997, 9998, 9999},
		},
		{
			"single-update-down",
			9999,
			p1121,
			map[types.Priority]uint16{p1120: 10000, p1130: 9999},
			map[uint16]types.Priority{10000: p1120, 9999: p1130},
			[]uint16{9999, 10000},
		},
		{
			"multiple-updates-down",
			9998,
			p1131,
			map[types.Priority]uint16{
				p1120: 10000, p1121: 9999, p1130: 9998, p1140: 9997, p1141: 9996, p1160: 9995},
			map[uint16]types.Priority{
				10000: p1120, 9999: p1121, 9998: p1130, 9997: p1140, 9996: p1141, 9995: p1160},
			[]uint16{9995, 9996, 9997, 9998, 9999, 10000},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pa := newPriorityAssigner(func(p types.Priority, isSingleTier bool) uint16 {
				return tt.insertionPoint
			}, true)
			for ofPriority, p := range tt.originalOFMap {
				pa.updatePriorityAssignment(ofPriority, p)
			}
			_, _, revertFunc, _ := pa.GetOFPriority(tt.extraPriority)
			revertFunc()
			assert.Equalf(t, tt.originalPriorityMap, pa.priorityMap, "Got unexpected priorityMap")
			assert.Equalf(t, tt.originalOFMap, pa.ofPriorityMap, "Got unexpected ofPriorityMap")
			assert.Equalf(t, tt.originalSorted, pa.sortedOFPriorities, "Got unexpected sortedOFPriorities")
		})
	}
}

func generatePriorities(start, end int32) []types.Priority {
	priorities := make([]types.Priority, end-start+1)
	for i := start; i <= end; i++ {
		priorities[i-start] = types.Priority{TierPriority: 1, PolicyPriority: 5, RulePriority: i - start}
	}
	return priorities
}

func TestRegisterAllOFPriorities(t *testing.T) {
	pa := newPriorityAssigner(InitialOFPriority, true)
	maxPriorities := generatePriorities(int32(PolicyBottomPriority), int32(PolicyTopPriority))
	err := pa.RegisterPriorities(maxPriorities)
	assert.Equalf(t, nil, err, "Error occurred in registering max number of allowed priorities")

	extraPriority := types.Priority{TierPriority: 1, PolicyPriority: 5, RulePriority: int32(PolicyTopPriority) - int32(PolicyBottomPriority) + 1}
	_, _, _, err = pa.GetOFPriority(extraPriority)
	assert.Errorf(t, err, "Error should be raised after max number of priorities are registered")
}
