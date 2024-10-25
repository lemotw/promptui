package multidimlist

import (
	"reflect"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		items   interface{}
		size    int
		wantErr bool
	}{
		{"Valid slice", []string{"a", "b", "c"}, 2, false},
		{"Empty slice", []int{}, 1, false},
		{"Invalid size", []float64{1.1, 2.2}, 0, true},
		{"Nil items", nil, 1, true},
		{"Non-slice items", 42, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.items, tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Errorf("New() returned nil, want non-nil")
			}
		})
	}
}

func TestList_Prev(t *testing.T) {
	var testArr interface{} = []string{"a", "b", "c", "d", "e"}
	list, _ := New(testArr, 3)
	list.SetCursor(2)
	list.Prev()
	if list.cursor[0] != 1 {
		t.Errorf("Prev() cursor = %v, want %v", list.cursor[0], 1)
	}
	list.Prev()
	list.Prev()
	if list.cursor[0] != 0 {
		t.Errorf("Prev() cursor = %v, want %v", list.cursor[0], 0)
	}
}

func TestList_Next(t *testing.T) {
	var testArr interface{} = []string{"a", "b", "c", "d", "e"}
	list, _ := New(testArr, 3)
	list.Next()
	if list.cursor[0] != 1 {
		t.Errorf("Next() cursor = %v, want %v", list.cursor[0], 1)
	}
	list.Next()
	list.Next()
	list.Next()
	if list.cursor[0] != 4 {
		t.Errorf("Next() cursor = %v, want %v", list.cursor[0], 4)
	}

	list.Next()
	if list.cursor[0] != 4 {
		t.Errorf("Next() cursor = %v, want %v", list.cursor[0], 4)
	}
}

func TestList_PageUp(t *testing.T) {
	var testArr interface{} = []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	list, _ := New(testArr, 3)
	list.SetCursor(6)
	list.PageUp()
	if list.start != 1 || list.cursor[0] != 1 {
		t.Errorf("PageUp() start = %v, cursor = %v, want start = 3, cursor = 3", list.start, list.cursor[0])
	}
}

func TestList_PageDown(t *testing.T) {
	var testArr interface{} = []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	list, _ := New(testArr, 3)
	list.PageDown()
	if list.start != 3 || list.cursor[0] != 3 {
		t.Errorf("PageDown() start = %v, cursor = %v, want start = 3, cursor = 3", list.start, list.cursor[0])
	}
}

func TestList_SetCursor(t *testing.T) {
	var testArr interface{} = []string{"a", "b", "c", "d", "e"}
	list, _ := New(testArr, 3)
	list.SetCursor(4)
	if list.cursor[0] != 4 || list.start != 2 {
		t.Errorf("SetCursor(4) cursor = %v, start = %v, want cursor = 4, start = 2", list.cursor[0], list.start)
	}
	list.SetCursor(-1)
	if list.cursor[0] != 0 {
		t.Errorf("SetCursor(-1) cursor = %v, want 0", list.cursor[0])
	}
}

func TestList_Search(t *testing.T) {
	var items interface{} = []string{"apple", "banana", "cherry", "date"}
	list, _ := New(items, 3)
	list.Searcher = func(input string, item interface{}, index int) bool {
		itemStr := item.(string)
		return strings.Contains(itemStr, input)
	}
	list.Search("b")
	if len(list.scope) != 1 || *list.scope[0] != "banana" {
		t.Errorf("Search('b') scope = %v, want [banana]", list.scope)
	}
}

func TestList_CancelSearch(t *testing.T) {
	var items interface{} = []string{"apple", "banana", "cherry", "date"}
	list, _ := New(items, 3)
	list.Searcher = func(input string, item interface{}, index int) bool {
		return strings.Contains(item.(string), input)
	}
	list.Search("b")
	list.CancelSearch()
	if len(list.scope) != len([]string{"apple", "banana", "cherry", "date"}) {
		t.Errorf("CancelSearch() scope length = %v, want %v", len(list.scope), len([]string{"apple", "banana", "cherry", "date"}))
	}
}

func TestList_Index(t *testing.T) {
	var testArr interface{} = []string{"a", "b", "c", "d", "e"}
	list, _ := New(testArr, 3)
	list.SetCursor(2)
	index := list.Index()
	if !reflect.DeepEqual(index, []int{2}) {
		t.Errorf("Index() = %v, want [2]", index)
	}
}

func TestList_Items(t *testing.T) {
	var testArr interface{} = []string{"a", "b", "c", "d", "e"}
	list, _ := New(testArr, 3)
	list.SetCursor(1)
	items, active := list.Items()
	expectedItems := []interface{}{"a", "b", "c"}
	if !reflect.DeepEqual(items, expectedItems) || active != 1 {
		t.Errorf("Items() = %v, %v, want %v, 1", items, active, expectedItems)
	}

	list.SetCursor(3)
	items, active = list.Items()
	expectedItems = []interface{}{"b", "c", "d"}
	if !reflect.DeepEqual(items, expectedItems) || active != 2 {
		t.Errorf("Items() = %v, %v, want %v, 1", items, active, expectedItems)
	}
}

func TestList_CanPageUp(t *testing.T) {
	var testArr interface{} = []string{"a", "b", "c", "d", "e"}
	list, _ := New(testArr, 3)
	if list.CanPageUp() {
		t.Errorf("CanPageUp() = true, want false")
	}
	list.SetCursor(4)
	if !list.CanPageUp() {
		t.Errorf("CanPageUp() = false, want true")
	}
}

func TestList_CanPageDown(t *testing.T) {
	var testArr interface{} = []string{"a", "b", "c", "d", "e"}
	list, _ := New(testArr, 3)
	if !list.CanPageDown() {
		t.Errorf("CanPageDown() = false, want true")
	}
	list.SetCursor(4)
	if list.CanPageDown() {
		t.Errorf("CanPageDown() = true, want false")
	}
}

func TestList_DiveIn(t *testing.T) {
	tests := []struct {
		name       string
		input      interface{}
		size       int
		operations func(*List)
		wantCursor []int
		wantErr    bool
	}{
		{
			name: "dive into first level list",
			input: []interface{}{
				[]interface{}{"a", "b", "c"},
				[]interface{}{"d", "e", "f"},
			},
			size:       2,
			wantCursor: []int{0, 0},
			wantErr:    false,
		},
		{
			name: "dive into nested list at specific position",
			input: []interface{}{
				[]interface{}{"a", "b"},
				[]interface{}{"c", "d"},
				[]interface{}{"e", "f"},
			},
			size: 2,
			operations: func(l *List) {
				l.Next()
			},
			wantCursor: []int{1, 0},
			wantErr:    false,
		},
		{
			name: "attempt to dive into non-list item",
			input: []interface{}{
				"not a list",
				[]interface{}{"a", "b"},
			},
			size:       2,
			wantCursor: []int{0},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := New(tt.input, tt.size)
			if err != nil {
				t.Fatalf("Failed to create new List: %v", err)
			}

			if tt.operations != nil {
				tt.operations(l)
			}

			err = l.DiveIn()

			if (err != nil) != tt.wantErr {
				t.Errorf("DiveIn() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(l.cursor) != len(tt.wantCursor) {
					t.Errorf("Cursor length = %v, want %v", len(l.cursor), len(tt.wantCursor))
				}

				for i, v := range tt.wantCursor {
					if l.cursor[i] != v {
						t.Errorf("Cursor[%d] = %v, want %v", i, l.cursor[i], v)
					}
				}
			}
		})
	}
}

func TestList_DiveOut(t *testing.T) {
	tests := []struct {
		name       string
		input      interface{}
		size       int
		operations func(*List)
		wantCursor []int
		wantErr    bool
	}{
		{
			name: "dive out from first level",
			input: []interface{}{
				[]interface{}{"a", "b"},
				[]interface{}{"c", "d"},
			},
			size: 2,
			operations: func(l *List) {
				l.DiveIn()
			},
			wantCursor: []int{0},
			wantErr:    false,
		},
		{
			name: "dive out from second level",
			input: []interface{}{
				[]interface{}{
					[]interface{}{"a", "b"},
					[]interface{}{"c", "d"},
				},
			},
			size: 2,
			operations: func(l *List) {
				l.DiveIn()
				l.DiveIn()
			},
			wantCursor: []int{0, 0},
			wantErr:    false,
		},
		{
			name: "dive out from different position",
			input: []interface{}{
				[]interface{}{"a", "b"},
				[]interface{}{"c", "d"},
			},
			size: 2,
			operations: func(l *List) {
				l.Next()
				l.DiveIn()
			},
			wantCursor: []int{1},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := New(tt.input, tt.size)
			if err != nil {
				t.Fatalf("Failed to create new List: %v", err)
			}

			if tt.operations != nil {
				tt.operations(l)
			}

			err = l.DiveOut()

			if (err != nil) != tt.wantErr {
				t.Errorf("DiveOut() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(l.cursor) != len(tt.wantCursor) {
					t.Errorf("Cursor length = %v, want %v", len(l.cursor), len(tt.wantCursor))
				}

				for i, v := range tt.wantCursor {
					if l.cursor[i] != v {
						t.Errorf("Cursor[%d] = %v, want %v", i, l.cursor[i], v)
					}
				}
			}
		})
	}
}

func TestMultiLayerListNavigation(t *testing.T) {
	testData := []interface{}{
		[]interface{}{
			"1.1",
			[]interface{}{"1.2.1", "1.2.2", "1.2.3"},
			"1.3",
		},
		[]interface{}{
			"2.1",
			[]interface{}{"2.2.1", "2.2.2"},
			"2.3",
		},
		"3",
	}

	list, err := New(testData, 2)
	if err != nil {
		t.Fatalf("Failed to create new list: %v", err)
	}

	items, active := list.Items()
	if len(items) != 2 || active != 0 {
		t.Errorf("Initial state incorrect, got %d items with active %d, want 2 items with active 0", len(items), active)
	}

	list.Next()
	err = list.DiveIn()
	if err != nil {
		t.Fatalf("Failed to dive into first sublayer: %v", err)
	}

	items, active = list.Items()
	if len(items) != 2 || items[0] != "2.1" {
		t.Errorf("First sublayer incorrect, got first item %v, want '2.1'", items[0])
	}

	list.Next()
	cursor := list.Index()
	if cursor[len(cursor)-1] != 1 {
		t.Errorf("Cursor position incorrect in sublayer, got %v, want 1", cursor[len(cursor)-1])
	}

	items, active = list.Items()
	if len(items) != 2 || items[0] != "2.1" {
		t.Errorf("First sublayer incorrect, got first item %v, want '2.1'", items[0])
	}

	err = list.DiveOut()
	if err != nil {
		t.Fatalf("Failed to dive out: %v", err)
	}

	items, active = list.Items()
	index := list.Index()
	if len(index) != 1 || index[0] != 1 {
		t.Errorf("After diving out, index incorrect, got %v, want [1]", index)
	}

	if !list.CanPageDown() {
		t.Error("Should be able to page down")
	}
	list.PageDown()
	items, active = list.Items()
	if len(items) != 2 {
		t.Errorf("After page down, got %d items, want 2", len(items))
	}

	list.Searcher = func(input string, item interface{}, index int) bool {
		return reflect.TypeOf(item).Kind() == reflect.String && strings.Contains(item.(string), input)
	}
	list.Search("3")
	items, active = list.Items()
	if len(items) != 1 || items[0] != "3" {
		t.Errorf("Search result incorrect, got %v, want ['3']", items)
	}

	err = list.CancelSearch()
	if err != nil {
		t.Fatalf("Failed to cancel search: %v", err)
	}
	items, active = list.Items()
	if len(items) != 2 {
		t.Errorf("After cancel search, got %d items, want 2", len(items))
	}
}
