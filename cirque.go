package cirque

import (
	"container/ring"
	"log"
	"sync"
	"sync/atomic"
	"unsafe"
)

type Cirque[T any] struct {
	writeHead *ring.Ring // Writer head position pointer
	readHead  *ring.Ring // Reader head position pointer
	readMu    sync.Mutex // Mutex lock for reads only
	len       int        // Number of items in queue
	cap       int        // Capacity of queue
}

// New creates a Cirque of initial size n with items of type T.
func New[T any](n int) *Cirque[T] {
	if n <= 0 {
		return nil
	}
	cq := new(Cirque[T])

	// Saving capacity in the struct itself.
	// This can be calculated by calling Ring.Len(), but that has O(n) complexity.
	// By saving the capacity from the start we can lower that to O(1).
	cq.cap = n

	// Create heads
	cq.readHead = ring.New(n)
	cq.writeHead = cq.readHead

	return cq
}

// Len returns the number of items currently in the queue.
// Because this is updated on every operation, this method offers O(1) complexity.
func (cq *Cirque[T]) Len() int {
	return cq.len
}

func (cq *Cirque[T]) loadHead(head **ring.Ring) *ring.Ring {
	return (*ring.Ring)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(head))))
}

func (cq *Cirque[T]) getReaderHead() *ring.Ring {
	return cq.loadHead(&cq.readHead)
}

func (cq *Cirque[T]) getWriterHead() *ring.Ring {
	return cq.loadHead(&cq.writeHead)
}

func (cq *Cirque[T]) moveHeadForward(head **ring.Ring) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(head)), unsafe.Pointer((*head).Next()))
}

func (cq *Cirque[T]) moveWriterHeadForward() {
	cq.moveHeadForward(&cq.writeHead)
}

func (cq *Cirque[T]) moveReaderHeadForward() {
	cq.moveHeadForward(&cq.readHead)
}

// Write to the current position
func (cq *Cirque[T]) write(item T) {
	h := cq.getWriterHead()
	h.Value = item
}

// Read from current position.
func (cq *Cirque[T]) read() T {
	return cq.getReaderHead().Value.(T)
}

func (cq *Cirque[T]) grow(min int) {
	if min < 0 {
		log.Printf("Tried to call grow on Cirque with min of %d.\n", min)
		return
	}
	cq.readMu.Lock()
	defer cq.readMu.Unlock()

	// Create new ring to (more than) double current capacity
	newRing := ring.New(min)

	// Join rings together
	cq.writeHead.Link(newRing)

	// Update capacity
	cq.cap += min
}

// Enqueue adds the input elements to the queue
func (cq *Cirque[T]) Enqueue(elements ...T) {
	for _, item := range elements {
		// If the writer head is next to the reader head the queue is full.
		if cq.getWriterHead().Next() == cq.getReaderHead() {
			// grow is a blocking call here, and since we assume a single writer
			// this is safe to do without a lock for writes.
			minSize := cq.cap + len(elements)
			cq.grow(minSize)
		}

		// Write data in the current position.
		cq.write(item)

		// Update length
		cq.len++

		// Move writer head to the next position.
		cq.moveWriterHeadForward()
	}
}

// Dequeue returns a maximum of n items from the queue.
func (cq *Cirque[T]) Dequeue(n int) []T {
	if n <= 0 {
		return nil
	}

	// Temporary slice to populate with results
	var result []T

	cq.readMu.Lock()
	defer cq.readMu.Unlock()

	for i := 0; i < n; i++ {
		// If reader head is in the same place as writer head no data is available to read.
		if cq.getReaderHead() == cq.getWriterHead() {
			return result
		}

		// Dequeue from current position.
		result = append(result, cq.read())
		
		// Update length
		cq.len--		

		// Move reader head to the next position.
		cq.moveReaderHeadForward()
	}

	return result
}
