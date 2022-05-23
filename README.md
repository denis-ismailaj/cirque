# Cirque

## Description

`Cirque` is a FIFO queue with generics backed by a circular list (`Ring` from `container/ring`) that enables 
independent reads and writes.

## How it works

Independent reads and writes are achieved by holding two separate pointers (_heads_) to that list,
one for reading and one for writing, which follow this set of rules:
- Both heads are initialized at the same point.
- The _writer head_ does not write if it is next to the _reader head_.
- The _reader head_ does not read if it's on the same place as the _writer head_.

The procedure for writing is:
1. The _writer head_ checks if _reader head_ is next to it.
    1. If it isn't, there is free space, so it continues.
    2. If it is, the queue is full, so read lock is requested and the list is resized.
2. Data is written in the current position.
3. The _writer head_ moves forward by 1 position.

The procedure for reading is:
1. Gain read lock.
2. The _reader head_ checks if it's on the same position with the _writer head_. If it is, no data is available to read, so it returns.
3. Data is read from the current position.
4. The _reader head_ moves forward by 1 position.

## Concurrency

Concurrent reads are policed with a mutex lock.
As long as list capacity is sufficient, a single writer can operate lock-free without disturbing reads.
However, no reads can be made while the list is being resized.
