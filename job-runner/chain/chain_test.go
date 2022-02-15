// Copyright 2017-2019, Square, Inc.

package chain

import (
	"reflect"
	"sort"
	"testing"

	"github.com/square/spincycle/v2/proto"
	testutil "github.com/square/spincycle/v2/test"
)

func TestNewChain(t *testing.T) {
	jc := &proto.JobChain{
		Jobs: map[string]proto.Job{
			"job1": proto.Job{
				Id:    "job1",
				State: proto.STATE_COMPLETE,
			},
			"job2": proto.Job{
				Id:    "job2",
				State: proto.STATE_FAIL,
			},
			"job3": proto.Job{
				Id:    "job3",
				State: proto.STATE_STOPPED,
			},
			"job4": proto.Job{
				Id:    "job4",
				State: proto.STATE_UNKNOWN,
			},
			"job5": proto.Job{
				Id:    "job5",
				State: proto.STATE_RUNNING,
			},
			"job6": proto.Job{
				Id:    "job6",
				State: proto.STATE_PENDING,
			},
		},
		FinishedJobs: 1,
	}

	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))

	expectedJobStates := map[string]byte{
		"job1": proto.STATE_COMPLETE,
		"job2": proto.STATE_FAIL,
		"job3": proto.STATE_STOPPED,
		"job4": proto.STATE_UNKNOWN,
		"job5": proto.STATE_RUNNING,
		"job6": proto.STATE_PENDING,
	}
	for jobId, expectedState := range expectedJobStates {
		if c.JobState(jobId) != expectedState {
			t.Errorf("%s state = %d, expected state %d", jobId, c.JobState(jobId), expectedState)
		}
	}

	// FinishedJobs reports proto.JobChain.FinishedJobs. We haven't ran anything,
	// so the number is straight from the struct. If the job chain ran, a reaper
	// would call Chain.IncrementFinishedJobs.
	gotFinished := c.FinishedJobs()
	if gotFinished != 1 {
		t.Errorf("got %d finished jobs, expected 1", gotFinished)
	}
}

func TestRunnableJobs(t *testing.T) {
	// Job chain:
	//       2 - 5
	//      / \
	// -> 1    4
	//     \  /
	//      3
	// Job 3 and 5 should be runnable

	jobs := map[string]proto.Job{
		"job1": proto.Job{
			Id:            "job1",
			State:         proto.STATE_COMPLETE,
			SequenceId:    "job1",
			SequenceRetry: 1,
		},
		"job2": proto.Job{
			Id:         "job2",
			State:      proto.STATE_COMPLETE,
			SequenceId: "job1",
		},
		"job3": proto.Job{ // cannot be run
			Id:         "job3",
			State:      proto.STATE_STOPPED,
			SequenceId: "job1",
			Retry:      1,
		},
		"job4": proto.Job{
			Id:         "job4",
			State:      proto.STATE_PENDING,
			SequenceId: "job1",
		},
		"job5": proto.Job{ // cTestIsRunnablean be run
			Id:         "job5",
			State:      proto.STATE_PENDING,
			SequenceId: "job1",
		},
	}
	jc := &proto.JobChain{
		RequestId: "resume",
		Jobs:      jobs,
		AdjacencyList: map[string][]string{
			"job1": {"job2", "job3"},
			"job2": {"job4", "job5"},
			"job3": {"job4"},
		},
	}
	sjc := &proto.SuspendedJobChain{
		RequestId: "resume",
		JobChain:  jc,
		TotalJobTries: map[string]uint{
			"job1": 2, // sequence retried once
			"job2": 2,
			"job3": 3,
			"job4": 1,
		},
		LatestRunJobTries: map[string]uint{
			"job1": 1,
			"job2": 1,
			"job3": 2, // job3 should have 1 try left
			"job4": 1,
		},
		SequenceTries: map[string]uint{
			"job1": 1,
		},
	}
	c := NewChain(sjc.JobChain, sjc.SequenceTries, sjc.TotalJobTries, sjc.LatestRunJobTries)

	expectedJobs := proto.Jobs{jc.Jobs["job5"]}
	sort.Sort(expectedJobs)
	runnableJobs := c.RunnableJobs()
	sort.Sort(runnableJobs)

	if !reflect.DeepEqual(runnableJobs, expectedJobs) {
		t.Errorf("runnableJobs = %v, want %v", runnableJobs, expectedJobs)
	}
}

func TestNextJobs(t *testing.T) {
	jc := &proto.JobChain{
		Jobs: testutil.InitJobs(4),
		AdjacencyList: map[string][]string{
			"job1": {"job2", "job3"},
			"job2": {"job4"},
			"job3": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))

	expectedNextJobs := proto.Jobs{jc.Jobs["job2"], jc.Jobs["job3"]}
	sort.Sort(expectedNextJobs)
	nextJobs := c.NextJobs("job1")
	sort.Sort(nextJobs)

	if !reflect.DeepEqual(nextJobs, expectedNextJobs) {
		t.Errorf("nextJobs = %v, want %v", nextJobs, expectedNextJobs)
	}

	nextJobs = c.NextJobs("job4")

	if len(nextJobs) != 0 {
		t.Errorf("nextJobs count = %d, want 0", len(nextJobs))
	}
}

func TestPreviousJobs(t *testing.T) {
	jc := &proto.JobChain{
		Jobs: testutil.InitJobs(4),
		AdjacencyList: map[string][]string{
			"job1": {"job2", "job3"},
			"job2": {"job4"},
			"job3": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))

	expectedPreviousJobs := proto.Jobs{jc.Jobs["job2"], jc.Jobs["job3"]}
	sort.Sort(expectedPreviousJobs)
	previousJobs := c.previousJobs("job4")
	sort.Sort(previousJobs)

	if !reflect.DeepEqual(previousJobs, expectedPreviousJobs) {
		t.Errorf("previousJobs = %v, want %v", previousJobs, expectedPreviousJobs)
	}

	previousJobs = c.previousJobs("job1")

	if len(previousJobs) != 0 {
		t.Errorf("previousJobs count = %d, want 0", len(previousJobs))
	}
}

func TestIsRunnable(t *testing.T) {
	jc := &proto.JobChain{
		Jobs: testutil.InitJobs(6),
		AdjacencyList: map[string][]string{
			"job1": {"job2", "job3", "job5"},
			"job2": {"job4", "job6"},
			"job3": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_COMPLETE)
	c.SetJobState("job3", proto.STATE_PENDING)
	c.SetJobState("job6", proto.STATE_STOPPED)
	c.IncrementJobTries("job6", 1) // tried once before stop

	// Job 1 has already been run
	expectedRunnable := false
	runnable := c.IsRunnable("job1")

	if runnable != expectedRunnable {
		t.Errorf("runnable = %t, want %t", runnable, expectedRunnable)
	}

	// Job 4 can't run until job 3 is complete
	expectedRunnable = false
	runnable = c.IsRunnable("job4")

	if runnable != expectedRunnable {
		t.Errorf("runnable = %t, want %t", runnable, expectedRunnable)
	}

	// Job 5 can run (because Job 1 is done)
	expectedRunnable = true
	runnable = c.IsRunnable("job5")

	if runnable != expectedRunnable {
		t.Errorf("runnable = %t, want %t", runnable, expectedRunnable)
	}

	// Job 6 can run (stopped)
	expectedRunnable = false
	runnable = c.IsRunnable("job6")

	if runnable != expectedRunnable {
		t.Errorf("runnable = %t, want %t", runnable, expectedRunnable)
	}
}

func TestIsDoneRunning(t *testing.T) {
	// A chain is not done (and not complete) if any job is running
	jc := &proto.JobChain{
		Jobs: testutil.InitJobs(4),
		AdjacencyList: map[string][]string{
			"job1": {"job2", "job3"},
			"job2": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))
	c.IncrementSequenceTries("job1", 1)
	c.SetJobState("job1", proto.STATE_RUNNING)

	expectedDone := false
	expectedComplete := false
	done, complete := c.IsDoneRunning()

	if done != expectedDone || complete != expectedComplete {
		t.Errorf("done = %t, complete = %t, want %t and %t", done, complete, expectedDone, expectedComplete)
	}
}

func TestIsDoneCompleteAndPending(t *testing.T) {
	// A chain is not done (and not complete) if any job is pending and runnable
	jc := &proto.JobChain{
		Jobs: testutil.InitJobs(4),
		AdjacencyList: map[string][]string{
			"job1": {"job2", "job3"},
			"job2": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))
	c.IncrementSequenceTries("job1", 1)
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_COMPLETE)
	c.SetJobState("job3", proto.STATE_PENDING)
	// ^ Job 4 can still be run

	expectedDone := false
	expectedComplete := false
	done, complete := c.IsDoneRunning()

	if done != expectedDone || complete != expectedComplete {
		t.Errorf("done = %t, complete = %t, want %t and %t", done, complete, expectedDone, expectedComplete)
	}
}

func TestIsDoneFailAndPending(t *testing.T) {
	// A chain is not done (and not complete) if any job is pending and runnable
	jc := &proto.JobChain{
		Jobs: testutil.InitJobs(4),
		AdjacencyList: map[string][]string{
			"job1": {"job2", "job3"},
			"job2": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))
	c.IncrementSequenceTries("job1", 1)
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_COMPLETE)
	c.SetJobState("job3", proto.STATE_FAIL)
	// ^ Job 4 is pending and runnable because job2 is complete
	// The job3 fail doesn't matter because, currently, we don't fail the chain
	// immediately when a job fails

	expectedDone := false
	expectedComplete := false
	done, complete := c.IsDoneRunning()

	if done != expectedDone || complete != expectedComplete {
		t.Errorf("done = %t, complete = %t, want %t and %t", done, complete, expectedDone, expectedComplete)
	}
}

func TestIsDoneUnknownAndPending(t *testing.T) {
	// A chain is not done (and not complete) if any job is pending and runnable
	jc := &proto.JobChain{
		Jobs: testutil.InitJobs(4),
		AdjacencyList: map[string][]string{
			"job1": {"job2", "job3"},
			"job2": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))
	c.IncrementSequenceTries("job1", 1)
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_COMPLETE)
	c.SetJobState("job3", proto.STATE_UNKNOWN)
	// ^ Job 4 is pending and runnable because job2 is complete
	// The job3 "unknown" doesn't matter because, currently, we don't fail the chain
	// immediately when a job fails

	expectedDone := false
	expectedComplete := false
	done, complete := c.IsDoneRunning()

	if done != expectedDone || complete != expectedComplete {
		t.Errorf("done = %t, complete = %t, want %t and %t", done, complete, expectedDone, expectedComplete)
	}
}

func TestIsDoneFailNoSeqRetry(t *testing.T) {
	// A chain is done (and not complete) if any job is failed and can't retry seq
	jc := &proto.JobChain{
		Jobs: testutil.InitJobs(4),
		AdjacencyList: map[string][]string{
			"job1": {"job2", "job3"},
			"job2": {"job4"},
		},
		FinishedJobs: 2,
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))
	c.IncrementSequenceTries("job1", 1)
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_FAIL) // can't retry seq
	c.SetJobState("job3", proto.STATE_COMPLETE)
	c.SetJobState("job4", proto.STATE_PENDING)

	expectedDone := true
	expectedComplete := false
	done, complete := c.IsDoneRunning()

	if done != expectedDone || complete != expectedComplete {
		t.Errorf("done = %t, complete = %t, want %t and %t", done, complete, expectedDone, expectedComplete)
	}
}

func TestIsDoneUnknownNoSeqRetry(t *testing.T) {
	// A chain is done (and not complete) if any job is failed and can't retry seq
	jc := &proto.JobChain{
		Jobs: testutil.InitJobs(4),
		AdjacencyList: map[string][]string{
			"job1": {"job2", "job3"},
			"job2": {"job4"},
		},
		FinishedJobs: 2,
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))
	c.IncrementSequenceTries("job1", 1)
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_UNKNOWN) // can't retry seq
	c.SetJobState("job3", proto.STATE_COMPLETE)
	c.SetJobState("job4", proto.STATE_PENDING)

	expectedDone := true
	expectedComplete := false
	done, complete := c.IsDoneRunning()

	if done != expectedDone || complete != expectedComplete {
		t.Errorf("done = %t, complete = %t, want %t and %t", done, complete, expectedDone, expectedComplete)
	}
}

func TestIsDoneComplete(t *testing.T) {
	// A chain is done and complete if all jobs are complete
	jc := &proto.JobChain{
		Jobs: testutil.InitJobs(4),
		AdjacencyList: map[string][]string{
			"job1": {"job2", "job3"},
			"job2": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))
	c.IncrementSequenceTries("job1", 1)
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_COMPLETE)
	c.SetJobState("job3", proto.STATE_COMPLETE)
	c.SetJobState("job4", proto.STATE_COMPLETE)

	expectedDone := true
	expectedComplete := true
	done, complete := c.IsDoneRunning()

	if done != expectedDone || complete != expectedComplete {
		t.Errorf("done = %t, complete = %t, want %t and %t", done, complete, expectedDone, expectedComplete)
	}
}

func TestIsDoneStoppedAndComplete(t *testing.T) {
	// A chain is done (and not complete) if all jobs are complete and stopped
	jc := &proto.JobChain{
		Jobs: testutil.InitJobs(4),
		AdjacencyList: map[string][]string{
			"job1": {"job2", "job3"},
			"job2": {"job4"},
		},
		FinishedJobs: 3,
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_STOPPED)
	c.SetJobState("job3", proto.STATE_COMPLETE)
	c.SetJobState("job4", proto.STATE_COMPLETE)

	done, complete := c.IsDoneRunning()
	if done != true { // expect done = true
		t.Errorf("done is false, expected true")
	}
	if complete != false { // expect complete = false
		t.Errorf("complete is true, exepcted false")
	}
}

func TestIsDoneStoppedAndRunning(t *testing.T) {
	// A chain is not done (and not complete) if any job is running
	jc := &proto.JobChain{
		Jobs: testutil.InitJobs(4),
		AdjacencyList: map[string][]string{
			"job1": {"job2", "job3"},
			"job2": {"job4"},
		},
		FinishedJobs: 3,
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_COMPLETE)
	c.SetJobState("job3", proto.STATE_STOPPED)
	c.SetJobState("job4", proto.STATE_RUNNING)

	done, complete := c.IsDoneRunning()
	if done != false { // expect done = false
		t.Errorf("done is true, expected false")
	}
	if complete != false { // expect complete = false
		t.Errorf("complete is true, exepcted false")
	}
}

func TestIsDoneSuspendedJobChain(t *testing.T) {
	//   2-4
	// 1<
	//   3

	// A chain is not done (and not complete) if any job is pending and runnable
	jc := &proto.JobChain{
		Jobs: testutil.InitJobs(4),
		AdjacencyList: map[string][]string{
			"job1": {"job2", "job3"},
			"job2": {"job4"},
		},
		FinishedJobs: 3,
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))

	// This is how a suspended job chain will look: some complete, some stopped,
	// and the one's not ran are still pending. So we expect done = true because
	// nothing is runnable.
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_STOPPED) // block job4 from being runnable
	c.SetJobState("job3", proto.STATE_COMPLETE)
	c.SetJobState("job4", proto.STATE_PENDING) // not runnable because job2 is not complete

	done, complete := c.IsDoneRunning()
	if done != true { // expect done = true
		t.Errorf("done is true, expected false")
	}
	if complete != false { // expect complete = false
		t.Errorf("complete is true, exepcted false")
	}

	// Another variation: the stopped job isn't blocking another job.
	// So now we expect done = false because job4 is runnable.
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_COMPLETE)
	c.SetJobState("job3", proto.STATE_STOPPED) // nothing depends on this job
	c.SetJobState("job4", proto.STATE_PENDING) // runnable because job2 is complete

	done, complete = c.IsDoneRunning()
	if done != false { // expect done = false
		t.Errorf("done is true, expected false")
	}
	if complete != false { // expect complete = false
		t.Errorf("complete is true, exepcted false")
	}
}

func TestIsRunnableSuspendedJobChain(t *testing.T) {
	//   2-4
	// 1<
	//   3
	jc := &proto.JobChain{
		Jobs: testutil.InitJobs(4),
		AdjacencyList: map[string][]string{
			"job1": {"job2", "job3"},
			"job2": {"job4"},
		},
		FinishedJobs: 3,
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))

	// First variation from prev test ^
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_STOPPED) // not runnable
	c.SetJobState("job3", proto.STATE_COMPLETE)
	c.SetJobState("job4", proto.STATE_PENDING) // not runnable because job2 is not complete

	if c.IsRunnable("job1") != false {
		t.Error("job1 runnable, expected false because it's complete")
	}
	if c.IsRunnable("job2") != false {
		t.Error("job2 runnable, expected false because it's stopped")
	}
	if c.IsRunnable("job3") != false {
		t.Error("job3 runnable, expected false because it's complete")
	}
	if c.IsRunnable("job4") != false {
		t.Error("job4 runnable, expected false because job2 isn't complete")
	}

	// Another variation (see prev test)
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_COMPLETE)
	c.SetJobState("job3", proto.STATE_STOPPED) // not runnable
	c.SetJobState("job4", proto.STATE_PENDING) // runnable because job2 is complete

	if c.IsRunnable("job1") != false {
		t.Error("job1 runnable, expected false because it's complete")
	}
	if c.IsRunnable("job2") != false {
		t.Error("job1 runnable, expected false because it's complete")
	}
	if c.IsRunnable("job3") != false {
		t.Error("job3 runnable, expected false because it's stopped")
	}
	if c.IsRunnable("job4") != true {
		t.Error("job4 not runnable, expected true because job2 is complete")
	}
}

func TestSetJobState(t *testing.T) {
	jc := &proto.JobChain{
		Jobs: testutil.InitJobs(1),
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))

	c.SetJobState("job1", proto.STATE_COMPLETE)
	if jc.Jobs["job1"].State != proto.STATE_COMPLETE {
		t.Errorf("State = %d, want %d", jc.Jobs["job1"].State, proto.STATE_COMPLETE)
	}
}

func TestSetState(t *testing.T) {
	jc := &proto.JobChain{}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))

	c.SetState(proto.STATE_RUNNING)
	if c.State() != proto.STATE_RUNNING {
		t.Errorf("State = %d, want %d", c.State(), proto.STATE_RUNNING)
	}
}

func TestSequenceStartJob(t *testing.T) {
	jobs := testutil.InitJobsWithSequenceRetry(4, 2)
	jc := &proto.JobChain{
		Jobs: jobs,
		AdjacencyList: map[string][]string{
			"job1": {"job2"},
			"job2": {"job3"},
			"job3": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))

	expect := jobs["job1"]
	actual := c.SequenceStartJob("job2")

	if !reflect.DeepEqual(actual, expect) {
		t.Errorf("sequence start job= %v, expected %v", actual, expect)
	}
}

func TestIsSequenceStartJobs(t *testing.T) {
	jobs := testutil.InitJobsWithSequenceRetry(4, 2)
	jc := &proto.JobChain{
		Jobs: jobs,
		AdjacencyList: map[string][]string{
			"job1": {"job2"},
			"job2": {"job3"},
			"job3": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))

	if c.IsSequenceStartJob("job2") {
		t.Errorf("got true that job2 is a sequence start job, expected false")
	}
	if !c.IsSequenceStartJob("job1") {
		t.Errorf("got that job1 is not a sequence start job, expected true")
	}
}

func TestCanRetrySequenceTrue(t *testing.T) {
	jobs := testutil.InitJobsWithSequenceRetry(4, 2)
	jc := &proto.JobChain{
		Jobs: jobs,
		AdjacencyList: map[string][]string{
			"job1": {"job2"},
			"job2": {"job3"},
			"job3": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))

	expect := true
	actual := c.CanRetrySequence("job2")

	if actual != expect {
		t.Errorf("can retry sequence = %v, expected %v", actual, expect)
	}
}

func TestCanRetrySequenceFalse(t *testing.T) {
	jobs := testutil.InitJobsWithSequenceRetry(4, 2)
	jc := &proto.JobChain{
		Jobs: jobs,
		AdjacencyList: map[string][]string{
			"job1": {"job2"},
			"job2": {"job3"},
			"job3": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))

	// 2 retries are configured for the sequence job2 is in
	jobId := "job2"
	// Increment sequence tries thrice to exhaust retries
	c.IncrementSequenceTries(jobId, 3)

	expect := false
	actual := c.CanRetrySequence(jobId)

	if actual != expect {
		t.Errorf("can retry sequence = %v, expected %v", actual, expect)
	}
}

func TestIncrementSequenceTries(t *testing.T) {
	jobs := testutil.InitJobsWithSequenceRetry(4, 2)
	jc := &proto.JobChain{
		Jobs: jobs,
		AdjacencyList: map[string][]string{
			"job1": {"job2"},
			"job2": {"job3"},
			"job3": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))

	jobId := "job2"
	c.IncrementSequenceTries(jobId, 1)

	expect := uint(1)
	actual := c.SequenceTries(jobId)

	if actual != expect {
		t.Errorf("sequence tries= %v, expected %v", actual, expect)
	}
}

func TestSequenceTries(t *testing.T) {
	jobs := testutil.InitJobsWithSequenceRetry(4, 2)
	jc := &proto.JobChain{
		Jobs: jobs,
		AdjacencyList: map[string][]string{
			"job1": {"job2"},
			"job2": {"job3"},
			"job3": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))

	jobId := "job2"

	expect := uint(0)
	actual := c.SequenceTries(jobId)

	if actual != expect {
		t.Errorf("sequence tries= %v, expected %v", actual, expect)
	}
}

func TestIsDoneRetryableSequenceFalse(t *testing.T) {
	jobs := testutil.InitJobsWithSequenceRetry(4, 2)
	jc := &proto.JobChain{
		Jobs: jobs,
		AdjacencyList: map[string][]string{
			"job1": {"job2"},
			"job2": {"job3"},
			"job3": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))
	c.IncrementSequenceTries("job1", 1)
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_FAIL)

	expectDone := false
	expectComplete := false
	actualDone, actualComplete := c.IsDoneRunning()

	if actualDone != expectDone || actualComplete != expectComplete {
		t.Errorf("done = %v, expected %v. complete = %v, expected %v.", actualDone, expectDone, actualComplete, expectComplete)
	}
}

func TestIsDoneRetryableSequenceTrue(t *testing.T) {
	jobs := testutil.InitJobsWithSequenceRetry(4, 2)
	jc := &proto.JobChain{
		Jobs: jobs,
		AdjacencyList: map[string][]string{
			"job1": {"job2"},
			"job2": {"job3"},
			"job3": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))
	c.IncrementSequenceTries("job1", 1)
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_FAIL)

	// Simulate exhausting sequence retries
	failedJobId := "job2"
	c.IncrementSequenceTries(failedJobId, 2)

	expectDone := true
	expectComplete := false
	actualDone, actualComplete := c.IsDoneRunning()

	if actualDone != expectDone || actualComplete != expectComplete {
		t.Errorf("done = %v, expected %v. complete = %v, expected %v.", actualDone, expectDone, actualComplete, expectComplete)
	}
}

func TestIsDoneRetryableSequenceFalseUnknown(t *testing.T) {
	jobs := testutil.InitJobsWithSequenceRetry(4, 2)
	jc := &proto.JobChain{
		Jobs: jobs,
		AdjacencyList: map[string][]string{
			"job1": {"job2"},
			"job2": {"job3"},
			"job3": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))
	c.IncrementSequenceTries("job1", 1)
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_UNKNOWN)

	expectDone := false
	expectComplete := false
	actualDone, actualComplete := c.IsDoneRunning()

	if actualDone != expectDone || actualComplete != expectComplete {
		t.Errorf("done = %v, expected %v. complete = %v, expected %v.", actualDone, expectDone, actualComplete, expectComplete)
	}
}

func TestIsDoneRetryableSequenceTrueUnknown(t *testing.T) {
	jobs := testutil.InitJobsWithSequenceRetry(4, 2)
	jc := &proto.JobChain{
		Jobs: jobs,
		AdjacencyList: map[string][]string{
			"job1": {"job2"},
			"job2": {"job3"},
			"job3": {"job4"},
		},
	}
	c := NewChain(jc, make(map[string]uint), make(map[string]uint), make(map[string]uint))
	c.IncrementSequenceTries("job1", 1)
	c.SetJobState("job1", proto.STATE_COMPLETE)
	c.SetJobState("job2", proto.STATE_UNKNOWN)

	// Simulate exhausting sequence retries
	failedJobId := "job2"
	c.IncrementSequenceTries(failedJobId, 2)

	expectDone := true
	expectComplete := false
	actualDone, actualComplete := c.IsDoneRunning()

	if actualDone != expectDone || actualComplete != expectComplete {
		t.Errorf("done = %v, expected %v. complete = %v, expected %v.", actualDone, expectDone, actualComplete, expectComplete)
	}
}
