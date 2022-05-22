package ringbuffer

import (
	"container/ring"
	"sync"
	"sync/atomic"
	"unsafe"
)

type RingBuffer[T any] struct {
	writeHead *ring.Ring    // Writer head position pointer
	readHead  *ring.Ring    // Reader head position pointer
	mu        sync.Mutex    // Mutex lock for reads only
	updateCh  chan struct{} // Channel used as a condition variable when writer head is waiting for reader head to move.
}

// New creates a RingBuffer of fixed size n with items of type T.
func New[T any](n int) *RingBuffer[T] {
	if n <= 0 {
		return nil
	}
	or := new(RingBuffer[T])
	or.readHead = ring.New(n)
	or.writeHead = or.readHead
	or.updateCh = make(chan struct{})
	return or
}

func (rb *RingBuffer[T]) loadHead(head **ring.Ring) *ring.Ring {
	return (*ring.Ring)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(head))))
}

func (rb *RingBuffer[T]) getReaderHead() *ring.Ring {
	return rb.loadHead(&rb.readHead)
}

func (rb *RingBuffer[T]) getWriterHead() *ring.Ring {
	return rb.loadHead(&rb.writeHead)
}

func (rb *RingBuffer[T]) moveHeadForward(head **ring.Ring) {
	atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(head)), unsafe.Pointer((*head).Next()))
}

func (rb *RingBuffer[T]) moveWriterHeadForward() {
	rb.moveHeadForward(&rb.writeHead)
}

func (rb *RingBuffer[T]) moveReaderHeadForward() {
	rb.moveHeadForward(&rb.readHead)

	// Notify writer head if it is blocked, otherwise ignore if it's not listening.
	select {
	case rb.updateCh <- struct{}{}:
	default:
	}
}

// A reader does not read if its head is in the same place as the writer head.
// The writer head does not move to another position until after it has finished modifying the value where
// its head currently is. Writer head moves are atomic.
// So, writing a value does not really present a race condition as far as I can tell.
// However, to be sure I am saving the value atomically anyway.
func (rb *RingBuffer[T]) write(item T) {
	h := rb.getWriterHead()

	if h.Value == nil {
		h.Value = &atomic.Value{}
	}

	h.Value.(*atomic.Value).Store(item)
}

// Read from current position.
func (rb *RingBuffer[T]) read() T {
	return rb.getReaderHead().Value.(*atomic.Value).Load().(T)
}

// WriteOrWait gets a slice of items as input, and it returns when all items have been written.
func (rb *RingBuffer[T]) WriteOrWait(newItems []T) {
	for _, item := range newItems {
		// If the writer head is next to the reader head the buffer does not write anymore.
		// Wait for the reader head to signal a move by writing to the channel.
		if rb.getWriterHead().Next() == rb.getReaderHead() {
			<-rb.updateCh
		}

		// Write data in the current position.
		rb.write(item)

		// Move writer head to the next position.
		rb.moveWriterHeadForward()
	}
}

// Read gets a maximum number of items as input, and it returns a slice of items.
func (rb *RingBuffer[T]) Read(n int) []T {
	if n <= 0 {
		return nil
	}

	// Temporary slice to populate with results
	var result []T

	rb.mu.Lock()
	defer rb.mu.Unlock()

	for i := 0; i < n; i++ {
		// If reader head is in the same place as writer head no data is available to read.
		if rb.getReaderHead() == rb.getWriterHead() {
			return result
		}

		// Read from current position.
		result = append(result, rb.read())

		// Move reader head to the next position.
		rb.moveReaderHeadForward()
	}

	return result
}
