// Package ai is documented in doc.go.
package ai

import (
	"math/rand"
	"time"

	"squabble/dictionary"
	"squabble/engine"
)

// aiRequest carries everything the AI goroutine needs to compute one move.
// state is a deep clone of the live GameState — SECURITY-AI-2 / BR-AI-12.
// gen tags the request so a result for an abandoned request (one cancelled via
// Reset) can be recognised and discarded rather than mistaken for a later one.
type aiRequest struct {
	state *engine.GameState
	dict  *dictionary.Dictionary
	level int
	rng   *rand.Rand // freshly seeded per request to prevent repetitive play patterns
	gen   int
}

// aiResult pairs a computed move with the generation of the request that produced
// it, so Poll can drop results belonging to a request that Reset abandoned.
type aiResult struct {
	gen  int
	move engine.Move
}

// AIWorker runs the AI move computation on a dedicated goroutine so a polling UI
// loop never blocks — Pattern 2 (NFR-AI-T1, NFR-AI-R4).
//
// The two buffered channels (capacity 1) ensure:
//   - Request never blocks: reqCh always has room (only one request in flight at a time).
//   - Poll is non-blocking: resCh holds the result until Poll drains it.
//   - The AI goroutine never blocks indefinitely: before each send it drains any
//     stale result the UI abandoned via Reset, so the capacity-1 resCh always has
//     room and the send completes regardless of how often the UI polls.
//
// busy and gen are written and read exclusively on the UI goroutine; the AI
// goroutine never touches them, so no mutex is needed (NFR-AI-T1).
type AIWorker struct {
	reqCh chan aiRequest
	resCh chan aiResult
	busy  bool
	gen   int // generation of the most recent request
}

// NewAIWorker creates an AIWorker that uses choose as its move computation function.
// Inject ai.ChooseMove for production use; inject a stub for testing (NFR-AI-TEST-3).
func NewAIWorker(choose func(*engine.GameState, *dictionary.Dictionary, int, *rand.Rand) engine.Move) *AIWorker {
	w := &AIWorker{
		reqCh: make(chan aiRequest, 1),
		resCh: make(chan aiResult, 1),
	}
	go w.run(choose)
	return w
}

// Request dispatches a new AI move computation for the given game state.
// It clones state before sending so the AI goroutine and the UI goroutine never
// share a pointer — SECURITY-AI-2.
//
// Panics if called while the previous request's result has not yet been consumed
// by Poll (NFR-AI-R3 / BR-AI-09). The UI is responsible for checking busy before calling.
func (w *AIWorker) Request(state *engine.GameState, dict *dictionary.Dictionary, level int) {
	if w.busy {
		panic("ai.AIWorker.Request: called while a previous request is still in flight")
	}
	w.busy = true
	w.gen++
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	w.reqCh <- aiRequest{
		state: state.Clone(),
		dict:  dict,
		level: level,
		rng:   rng,
		gen:   w.gen,
	}
}

// Reset clears the busy flag and discards any result already sitting in resCh.
// Call this when a move is applied by means other than Poll (e.g. the 10-second
// timeout), so that the next Request call does not panic. A result for the
// abandoned request that has not yet been produced is recognised by its stale
// generation and discarded by a later Poll, so the goroutine never wedges.
func (w *AIWorker) Reset() {
	select {
	case <-w.resCh: // drain an in-flight result if the goroutine already finished
	default:
	}
	w.busy = false
}

// Poll checks whether the AI has finished computing a move without blocking.
// Returns (move, true) if the result for the current request is ready;
// (nil, false) otherwise. A result whose generation does not match the latest
// request is from a request abandoned via Reset: it is drained and discarded, and
// Poll reports not-ready so the goroutine's buffered send can proceed.
// Clears the busy flag when a current result is returned.
//
// The UI should call Poll once per Update() frame.
func (w *AIWorker) Poll() (engine.Move, bool) {
	select {
	case r := <-w.resCh:
		if r.gen != w.gen {
			return nil, false // stale result from an abandoned request — discard
		}
		w.busy = false
		return r.move, true
	default:
		return nil, false
	}
}

// run is the AI goroutine body. It loops forever, receiving requests and sending
// tagged results. The goroutine exits when reqCh is closed (not done in normal gameplay).
func (w *AIWorker) run(choose func(*engine.GameState, *dictionary.Dictionary, int, *rand.Rand) engine.Move) {
	for req := range w.reqCh {
		move := choose(req.state, req.dict, req.level, req.rng)
		// Discard any stale result still buffered from a request the UI abandoned
		// via Reset, so this send always has room and never blocks (resCh has
		// capacity 1). That stale result has an older generation; Poll would have
		// discarded it too, but draining here guarantees the goroutine cannot wedge
		// even if the UI stops polling.
		select {
		case <-w.resCh:
		default:
		}
		w.resCh <- aiResult{gen: req.gen, move: move}
	}
}
