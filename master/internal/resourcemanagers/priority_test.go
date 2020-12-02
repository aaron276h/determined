package resourcemanagers

import (
	"fmt"
	"testing"

	"gotest.tools/assert"

	"github.com/determined-ai/determined/master/pkg/actor"
)

func TestSortTasksByPriorityAndTimestamps(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := make([]*mockAgent, 0)
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0]},
		{id: "task2", slotsNeeded: 1, group: groups[0]},
		{id: "task3", slotsNeeded: 0, group: groups[1]},
		{id: "task4", slotsNeeded: 0, group: groups[1]},
		{id: "task5", slotsNeeded: 4, group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, mockGroups, _ := setupSchedulerStates(t, system, tasks, groups, agents)

	zeroSlotPendingTasksByPriority, _ := sortTasksByPriorityAndTimestamp(
		taskList, mockGroups, "", zeroSlotTaskFilter)

	tasksInLowerPriority := zeroSlotPendingTasksByPriority[lowerPriority]
	expectedTasksInLowerPriority := []*mockTask{}
	assertEqualToAllocate(t, tasksInLowerPriority, expectedTasksInLowerPriority)

	tasksInHigherPriority := zeroSlotPendingTasksByPriority[higherPriority]
	expectedTasksInHigherPriority := []*mockTask{tasks[2], tasks[3]}
	assertEqualToAllocate(t, tasksInHigherPriority, expectedTasksInHigherPriority)

	nonZeroSlotPendingTasksByPriority, _ := sortTasksByPriorityAndTimestamp(
		taskList, mockGroups, "", nonZeroSlotTaskFilter)

	tasksInLowerPriority = nonZeroSlotPendingTasksByPriority[lowerPriority]
	expectedTasksInLowerPriority = []*mockTask{tasks[0], tasks[1]}
	assertEqualToAllocate(t, tasksInLowerPriority, expectedTasksInLowerPriority)

	tasksInHigherPriority = nonZeroSlotPendingTasksByPriority[higherPriority]
	expectedTasksInHigherPriority = []*mockTask{tasks[4]}
	assertEqualToAllocate(t, tasksInHigherPriority, expectedTasksInHigherPriority)

	forceSetTaskAllocations(t, taskList, "task5", 1)
	_, scheduledTasksByPriority := sortTasksByPriorityAndTimestamp(
		taskList, mockGroups, "", nonZeroSlotTaskFilter)

	tasksInLowerPriority = scheduledTasksByPriority[lowerPriority]
	expectedTasksInLowerPriority = make([]*mockTask, 0)
	assertEqualToAllocate(t, tasksInLowerPriority, expectedTasksInLowerPriority)

	tasksInHigherPriority = scheduledTasksByPriority[higherPriority]
	expectedTasksInHigherPriority = []*mockTask{tasks[4]}
	assertEqualToAllocate(t, tasksInHigherPriority, expectedTasksInHigherPriority)
}

func TestPrioritySchedulingPreemptionDisabled(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4, maxZeroSlotContainers: 100},
		{id: "agent2", slots: 4, maxZeroSlotContainers: 100},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0]},
		{id: "task2", slotsNeeded: 1, group: groups[0]},
		{id: "task3", slotsNeeded: 1, group: groups[1]},
		{id: "task4", slotsNeeded: 0, group: groups[1]},
		{id: "task5", slotsNeeded: 4, group: groups[1]},
		{id: "task6", slotsNeeded: 0, group: groups[0]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[1], tasks[2], tasks[3], tasks[4], tasks[5]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	// Check that agent stat has not changed.
	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}
}

func TestPrioritySchedulingPreemptionDisabledHigherPriorityBlocksLowerPriority(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4},
		{id: "agent2", slots: 4},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0]},
		{id: "task2", slotsNeeded: 1, group: groups[0]},
		{id: "task3", slotsNeeded: 12, group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	// Check that agent stat has not changed.
	for _, agent := range agentMap {
		assert.Equal(t, agent.numEmptySlots(), 4)
	}
}

func TestPrioritySchedulingPreemptionDisabledWithLabels(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4, label: "label1"},
		{id: "agent2", slots: 4, label: "label1"},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0], label: "label1"},
		{id: "task2", slotsNeeded: 1, group: groups[0], label: "label1"},
		{id: "task3", slotsNeeded: 4, group: groups[1], label: "label2"},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[0], tasks[1]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}

func TestPrioritySchedulingPreemptionAaron(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4, maxZeroSlotContainers: 100},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0],
			allocatedAgent: agents[0], containerStarted: true},
		{id: "task2", slotsNeeded: 0, group: groups[0]},
		{id: "task3", slotsNeeded: 4, group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := []*mockTask{tasks[1]}
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	expectedToRelease := []*mockTask{tasks[0]}
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}

func TestPrioritySchedulingLowPriorityTasksAreBlockedByHigherPriority(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4, maxZeroSlotContainers: 100},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 4, group: groups[0]},
		{id: "task2", slotsNeeded: 8, group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, _ := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	expectedToAllocate := make([]*mockTask, 0)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)
}

func TestPrioritySchedulerPreemptZeroSlotTask(t *testing.T) {
	lowerPriority := 50
	higherPriority := 40

	agents := []*mockAgent{
		{id: "agent1", slots: 4, maxZeroSlotContainers: 1, zeroSlotContainers: 1},
	}
	groups := []*mockGroup{
		{id: "group1", priority: &lowerPriority},
		{id: "group2", priority: &higherPriority},
	}
	tasks := []*mockTask{
		{id: "task1", slotsNeeded: 0, group: groups[0],
			allocatedAgent: agents[0], containerStarted: true},
		{id: "task2", slotsNeeded: 0, group: groups[1]},
	}

	system := actor.NewSystem(t.Name())
	taskList, groupMap, agentMap := setupSchedulerStates(t, system, tasks, groups, agents)

	p := &priorityScheduler{preemptionEnabled: true}
	toAllocate, toRelease := p.prioritySchedule(taskList, groupMap, agentMap, BestFit)

	fmt.Println("1")
	expectedToAllocate := make([]*mockTask, 0)
	assertEqualToAllocate(t, toAllocate, expectedToAllocate)

	fmt.Println("2", toRelease)
	expectedToRelease := []*mockTask{tasks[0]}
	assertEqualToRelease(t, taskList, toRelease, expectedToRelease)
}
