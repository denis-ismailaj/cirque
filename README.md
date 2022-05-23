# Cirque

## Description

`Cirque` is a FIFO queue that internally uses a fixed-size circular list (`Ring` from `container/ring`),
and it holds two separate pointers (_heads_) to that list: one for reading and one for writing.

`Cirque` does not discard any items. When it gets full it simply blocks until it can finish writing.

Note: The same behavior (and more safety) can be achieved with a buffered channel.

## Implementation

`Cirque` has the following rules:
- Both heads are initialized at the same point.
- The _writer head_ does not write if it is next to the _reader head_.
- The _reader head_ does not read if it's on the same sector as the _writer head_.
  
The procedure for writing is:
  1. The _writer head_ checks if _reader head_ is next to it. If it is, wait for it to move.
  2. Data is written in the current position.
  3. The _writer head_ moves forward by 1 position.

The procedure for reading is:
  5. The _reader head_ checks if it's on the same position with the _writer head_.
  If it is, no data is available to read, so it returns.
  6. Data is read from the current position.
  7. The _reader head_ moves forward by 1 position.

`Cirque` exports a simple API with two operations: `WriteOrWait` and `Read`.
`WriteOrWait` gets a slice of items as input, and it returns when all items have been written.
`Read` gets a maximum number of items as input, and it returns a slice of items.

`Cirque` uses generics for its items (upper bound is `any`).
Internally, `ring.Ring`'s item type is the generic item type of `Cirque` wrapped in an `*atomic.Value`.

## Concurrency

`Cirque` reads and writes can happen separately because of its rules.
However, concurrent reads, or concurrent writes are not supported without locks
because there is only one head for each type of operation.
`Cirque` is intended to be used with a single writer and many readers.
Writes happen entirely lock-free while reads use mutex lock internally.

## Potential changes

A `Cirque` writer can fill the buffer in real-time as the readers move,
but if at certain times it can't match up and readers start getting empty results,
it could be implemented that in these cases the `Cirque` is resized such that it
reduces the probability of the readers getting empty results in the future.
This of course would be an expensive operation, but every time it happened
it would become less likely for it to happen again.
