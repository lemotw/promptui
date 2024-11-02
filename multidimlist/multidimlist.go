package multidimlist

import (
	"fmt"
	"reflect"
	"strings"
)

// Searcher is a base function signature that is used inside select when activating the search mode.
// If defined, it is called on each items of the select and should return a boolean for whether or not
// the item fits the searched term.
type Searcher func(input string, item interface{}, index int) bool

// NotFound is an index returned when no item was selected. This could
// happen due to a search without results.
const NotFound = -1

// List holds a collection of items that can be displayed with an N number of
// visible items. The list can be moved up, down by one item of time or an
// entire page (ie: visible size). It keeps track of the current selected item.
type List struct {
	originalItem interface{}
	items        []*interface{}
	scope        []*interface{}
	cursor       []int // cursor holds the index of the current selected item
	size         int   // size is the number of visible options
	start        int
	Searcher     Searcher
}

// New creates and initializes a list of searchable items. The items attribute must be a slice type with a
// size greater than 0. Error will be returned if those two conditions are not met.
func New(items interface{}, size int) (*List, error) {
	if size < 1 {
		return nil, fmt.Errorf("list size %d must be greater than 0", size)
	}

	if items == nil || reflect.TypeOf(items).Kind() != reflect.Slice {
		return nil, fmt.Errorf("items %v is not a slice", items)
	}

	slice := reflect.ValueOf(items)
	values := make([]*interface{}, slice.Len())

	for i := range values {
		item := slice.Index(i).Interface()
		values[i] = &item
	}

	return &List{size: size, originalItem: items, items: values, scope: values, cursor: []int{0}}, nil
}

// Prev moves the visible list back one item. If the selected item is out of
// view, the new select item becomes the last visible item. If the list is
// already at the top, nothing happens.
func (l *List) Prev() {
	if l.cursor[len(l.cursor)-1] > 0 {
		l.cursor[len(l.cursor)-1]--
	}

	if l.start > l.cursor[len(l.cursor)-1] {
		l.start = l.cursor[len(l.cursor)-1]
	}
}

// Search allows the list to be filtered by a given term. The list must
// implement the searcher function signature for this functionality to work.
func (l *List) Search(term string) {
	term = strings.Trim(term, " ")
	l.cursor[len(l.cursor)-1] = 0
	l.start = 0
	l.search(term)
}

// CancelSearch stops the current search and returns the list to its
// original order.
func (l *List) CancelSearch() error {
	l.cursor[len(l.cursor)-1] = 0
	l.start = 0
	l.scope = l.items

	return nil
}

func (l *List) search(term string) {
	var scope []*interface{}
	for i, item := range l.scope {
		if item != nil {
			if l.Searcher(term, *item, i) {
				scope = append(scope, item)
			}
		}
	}

	l.scope = scope
}

// Start returns the current render start position of the list.
func (l *List) Start() int {
	return l.start
}

// SetStart sets the current scroll position. Values out of bounds will be
// clamped.
func (l *List) SetStart(i int) {
	if i < 0 {
		i = 0
	}
	if i > l.cursor[len(l.cursor)-1] {
		l.start = l.cursor[len(l.cursor)-1]
	} else {
		l.start = i
	}
}

// GetCursor returns the current cursor position.
func (l *List) GetCursor() []int {
	return l.cursor
}

// SetCursor sets the position of the cursor in the list. Values out of bounds
// will be clamped.
func (l *List) SetCursor(i int) {
	max := len(l.scope) - 1
	if i >= max {
		i = max
	}
	if i < 0 {
		i = 0
	}
	l.cursor[len(l.cursor)-1] = i

	if l.start > l.cursor[len(l.cursor)-1] {
		l.start = l.cursor[len(l.cursor)-1]
	} else if l.start+l.size <= l.cursor[len(l.cursor)-1] {
		l.start = l.cursor[len(l.cursor)-1] - l.size + 1
	}
}

// Next moves the visible list forward one item. If the selected item is out of
// view, the new select item becomes the first visible item. If the list is
// already at the bottom, nothing happens.
func (l *List) Next() {
	max := len(l.scope) - 1

	if l.cursor[len(l.cursor)-1] < max {
		l.cursor[len(l.cursor)-1]++
	}

	if l.start+l.size <= l.cursor[len(l.cursor)-1] {
		l.start = l.cursor[len(l.cursor)-1] - l.size + 1
	}
}

// DiveIn moves the cursor to the next layer of the list.
func (l *List) DiveIn() error {
	// check is selected item could be dived into
	selected := l.scope[l.cursor[len(l.cursor)-1]]
	selectedValue := reflect.ValueOf(*selected)

	if selectedValue.Kind() != reflect.Slice {
		return fmt.Errorf("selected item is not a list")
	}

	// find actual cursor index
	for i, item := range l.items {
		if item == selected {
			l.cursor[len(l.cursor)-1] = i
			break
		}
	}

	// append 0 to cursor
	l.cursor = append(l.cursor, 0)

	// reset items and scope
	values := make([]*interface{}, selectedValue.Len())
	for i := range values {
		item := selectedValue.Index(i).Interface()
		values[i] = &item
	}
	l.scope = values
	l.items = values

	return nil
}

// DiveOut moves the cursor to the previous layer of the list.
func (l *List) DiveOut() error {
	// check if the cursor is at the root
	if len(l.cursor) == 1 {
		return fmt.Errorf("cursor is at the root")
	}

	// run through the cursor to find the previous items
	if l.originalItem == nil || reflect.TypeOf(l.originalItem).Kind() != reflect.Slice {
		return fmt.Errorf("items %v is not a slice", l.originalItem)
	}

	// get the original slice
	slice := reflect.ValueOf(l.originalItem).Convert(reflect.TypeOf(l.originalItem))

	// find the actual cursor index
	for i, c := range l.cursor {
		if len(l.cursor)-2 <= i {
			break
		}

		slice = slice.Index(c).Elem()
		if reflect.TypeOf(slice.Interface()).Kind() != reflect.Slice {
			return fmt.Errorf("items %v is not a slice", slice)
		}
	}

	// pop cursor index and reset items and scope
	values := make([]*interface{}, slice.Len())
	for i := range values {
		item := slice.Index(i).Interface()
		values[i] = &item
	}

	l.cursor = l.cursor[:len(l.cursor)-1]
	l.items = values
	l.scope = values

	return nil
}

// PageUp moves the visible list backward by x items. Where x is the size of the
// visible items on the list. The selected item becomes the first visible item.
// If the list is already at the bottom, the selected item becomes the last
// visible item.
func (l *List) PageUp() {
	start := l.start - l.size
	if start < 0 {
		l.start = 0
	} else {
		l.start = start
	}

	cursor := l.start

	if cursor < l.cursor[len(l.cursor)-1] {
		l.cursor[len(l.cursor)-1] = cursor
	}
}

// PageDown moves the visible list forward by x items. Where x is the size of
// the visible items on the list. The selected item becomes the first visible
// item.
func (l *List) PageDown() {
	start := l.start + l.size
	max := len(l.scope) - l.size

	switch {
	case len(l.scope) < l.size:
		l.start = 0
	case start > max:
		l.start = max
	default:
		l.start = start
	}

	cursor := l.start

	if cursor == l.cursor[len(l.cursor)-1] {
		l.cursor[len(l.cursor)-1] = len(l.scope) - 1
	} else if cursor > l.cursor[len(l.cursor)-1] {
		l.cursor[len(l.cursor)-1] = cursor
	}
}

// CanPageDown returns whether a list can still PageDown().
func (l *List) CanPageDown() bool {
	max := len(l.scope)
	return l.start+l.size < max
}

// CanPageUp returns whether a list can still PageUp().
func (l *List) CanPageUp() bool {
	return l.start > 0
}

// Index returns the index of the item currently selected inside the searched list. If no item is selected,
// the NotFound (-1) index is returned.
func (l *List) Index() []int {
	selected := l.scope[l.cursor[len(l.cursor)-1]]

	rt := []int{}
	for _, c := range l.cursor {
		rt = append(rt, c)
	}

	for i, item := range l.items {
		if item == selected {
			rt[len(rt)-1] = i
			return rt
		}
	}

	return []int{NotFound}
}

func (l *List) Items() ([]interface{}, int) {
	var result []interface{}
	max := len(l.scope)
	end := l.start + l.size

	if end > max {
		end = max
	}

	active := NotFound

	for i := l.start; i < end; i++ {
		item := *l.scope[i]
		result = append(result, item)

		if i == l.cursor[len(l.cursor)-1] {
			active = i - l.start
		}
	}

	return result, active
}
