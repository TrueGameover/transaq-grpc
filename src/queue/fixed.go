package queue

import (
	"container/list"
	"context"
	"sync"
	"time"
)

type FixedQueue[T interface{}] struct {
	maxSize      int
	elementsList *list.List
	mutex        *sync.Mutex
	channelsBag  *list.List
}

type channelBag[T interface{}] struct {
	ch  chan T
	ctx context.Context
}

func NewFixedQueue[T interface{}](ctx context.Context, size int) *FixedQueue[T] {
	l := list.New()
	l.Init()

	channelsBagList := list.New()
	channelsBagList.Init()

	m := sync.Mutex{}

	obj := FixedQueue[T]{
		maxSize:      size,
		elementsList: l,
		mutex:        &m,
		channelsBag:  channelsBagList,
	}

	go obj.dispatchToChannels(ctx)

	return &obj
}

func (q *FixedQueue[T]) Push(element T) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	actualSize := q.elementsList.Len()

	if actualSize >= q.maxSize {
		diff := actualSize - q.maxSize + 1
		for i := 0; i < diff; i++ {
			_ = q.catchHead()
		}
	}

	q.elementsList.PushBack(element)
}

func (q *FixedQueue[T]) Pop() *T {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	element := q.catchHead()

	return element
}

func (q *FixedQueue[T]) catchHead() *T {
	head := q.elementsList.Front()

	if head == nil {
		return nil
	}

	q.elementsList.Remove(head)

	element, ok := head.Value.(T)

	if !ok {
		return nil
	}

	return &element
}

func (q *FixedQueue[T]) removeBagAndGoNext(element *list.Element, value *channelBag[T]) *list.Element {
	if value != nil {
		close(value.ch)
	}

	nextElement := element.Next()
	q.channelsBag.Remove(element)
	return nextElement
}

func (q *FixedQueue[T]) dispatchToChannels(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if q.channelsBag.Len() == 0 {
			time.Sleep(time.Second * 1)
			continue
		}

		head := q.Pop()

		if head == nil {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		element := q.channelsBag.Front()
		hasReceivers := false
		for element != nil {
			bag, ok := element.Value.(channelBag[T])

			if !ok {
				element = q.removeBagAndGoNext(element, nil)
				continue
			}

			select {
			case <-bag.ctx.Done():
				element = q.removeBagAndGoNext(element, &bag)
				continue
			default:
			}

			// если не смогли записать в канал, пропускаем его
			select {
			case bag.ch <- *head:
				hasReceivers = true
			default:
			}

			element = element.Next()
		}

		if !hasReceivers {
			// message not delivered, push it again
			q.Push(*head)
		}
	}
}

func (q *FixedQueue[T]) Fetch(ctx context.Context) <-chan T {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	elementsChannel := make(chan T, q.maxSize)
	q.channelsBag.PushBack(channelBag[T]{
		ch:  elementsChannel,
		ctx: ctx,
	})

	return elementsChannel
}

func (q *FixedQueue[T]) GetMaxSize() int {
	return q.maxSize
}
