package shared

import (
	"os"
	"runtime"
	"strconv"
	"sync"
)

// Workers returns the analysis worker count. GIRL_JOBS overrides the default
// of one worker per CPU.
func Workers() int {
	if v, err := strconv.Atoi(os.Getenv("GIRL_JOBS")); err == nil && v > 0 {
		return v
	}
	return runtime.GOMAXPROCS(0)
}

// ParallelMap applies fn to every item using Workers() goroutines and returns
// the kept results in input order, so output stays deterministic. newState is
// called once per worker for per-goroutine state that is not safe to share.
func ParallelMap[In, Out, State any](items []In, newState func() State, fn func(State, In) (Out, bool)) []Out {
	workers := Workers()
	if workers > len(items) {
		workers = len(items)
	}
	type slot struct {
		val Out
		ok  bool
	}
	slots := make([]slot, len(items))
	next := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		state := newState()
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range next {
				v, ok := fn(state, items[i])
				slots[i] = slot{v, ok}
			}
		}()
	}
	for i := range items {
		next <- i
	}
	close(next)
	wg.Wait()

	out := make([]Out, 0, len(items))
	for _, s := range slots {
		if s.ok {
			out = append(out, s.val)
		}
	}
	return out
}
