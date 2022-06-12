package cirque

import "testing"

func TestEnqueueDequeue(t *testing.T) {
	initialSize := 50
	cq := New[int](initialSize)

	// Insert enough items to make the ring grow a couple of times.
	n := initialSize * 10

	var input []int

	for i := 0; i < n; i++ {
		cq.Enqueue(i)
		input = append(input, i)
	}

	for i := 0; i < n; i++ {
		if cq.Dequeue(1)[0] != input[i] {
			t.Fatal("Items missing or reordered.")
		}
	}

	// Test for panic when dequeuing when empty.
	cq.Dequeue(50)
}
