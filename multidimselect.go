package promptui

import (
	"bytes"
	"fmt"
	"io"
	"text/tabwriter"
	"text/template"

	"github.com/chzyer/readline"
	"github.com/lemotw/promptui/multidimlist"
	"github.com/lemotw/promptui/screenbuf"
)

// EnterCallback is a function that is called when the user presses enter
// The function should return true if the select should exit
type EnterCallback func(item interface{}, cursor []int) (bool, error)

// MultidimSelect is a select list that allows the user to navigate through a list of items
type MultidimSelect struct {
	// Label is the text displayed on top of the list
	Label interface{}
	// Items are the items to display inside the list
	Items interface{}

	// Templates can be used to customize the select output
	Templates *MultidimSelectTemplates
	// Keys is the set of keys used to control the interface
	Keys *MultidimSelectKeys
	// Internal list implementation
	list *multidimlist.List
	// Input/Output streams
	Stdin  io.ReadCloser
	Stdout io.WriteCloser
	// A function that determines how to render the cursor
	Pointer Pointer
	// EnterCallback is a function that is called when the user presses enter
	EnterCallback EnterCallback
	// Searcher is a function for filtering items
	Searcher multidimlist.Searcher

	// Size is the number of items that should appear
	Size int
	// CursorPos is the initial position of the cursor
	CursorPos int

	// IsVimMode sets whether to use vim mode
	IsVimMode bool
	// HideHelp sets whether to hide help information
	HideHelp bool
	// HideSelected sets whether to hide the text displayed after selection
	HideSelected bool
	// StartInSearchMode sets whether to start in search mode
	StartInSearchMode bool
}

// MultidimSelectKeys defines the available keys
type MultidimSelectKeys struct {
	// Navigation keys
	Next     Key
	Prev     Key
	PageUp   Key
	PageDown Key

	// Dimension navigation keys
	DiveIn  Key
	DiveOut Key

	// Search key
	Search Key
}

// MultidimSelectTemplates allows customizing the display
// You can use the FuncMap to add custom functions to the templates.
// joinSlice, isSlice, sliceLen, sliceItem and joinMap are available by default.
type MultidimSelectTemplates struct {
	// Compiled templates
	label    *template.Template
	active   *template.Template
	inactive *template.Template
	selected *template.Template
	details  *template.Template
	help     *template.Template
	// Function map for template execution
	FuncMap template.FuncMap

	// Label is the text displayed on top of the list
	Label string
	// Active is the template for the currently selected item
	Active string
	// Inactive is the template for non-selected items
	Inactive string
	// Selected is the template for when an item is chosen
	Selected string
	// Details is the template for additional item information
	Details string
	// Help is the template for help text
	Help string
}

// Run executes the select list
func (s *MultidimSelect) Run() ([]int, interface{}, error) {
	return s.RunCursorAt(s.CursorPos, 0)
}

// RunCursorAt executes the select list at a specific cursor position
func (s *MultidimSelect) RunCursorAt(cursorPos, scroll int) ([]int, interface{}, error) {
	if s.Size == 0 {
		s.Size = 5
	}

	l, err := multidimlist.New(s.Items, s.Size)
	if err != nil {
		return nil, "", err
	}
	l.Searcher = s.Searcher
	s.list = l

	s.setKeys()

	err = s.prepareTemplates()
	if err != nil {
		return nil, "", err
	}

	return s.innerRun(cursorPos, scroll, ' ')
}

func (s *MultidimSelect) innerRun(cursorPos, scroll int, top rune) ([]int, interface{}, error) {
	// Wrap stdin to intercept input
	stdinWrapper := newStdinWrapper(s.Stdin, func(p []byte) (bool, error) {
		switch (rune)(p[0]) {
		case KeyEnter:
			items, idx := s.list.Items()
			item := items[idx]

			// If the enter callback returns true, exit the select
			if s.EnterCallback != nil {
				return s.EnterCallback(item, s.list.GetCursor())
			}
		}
		return true, nil
	})

	c := &readline.Config{
		Stdin:  stdinWrapper,
		Stdout: s.Stdout,
	}
	err := c.Init()
	if err != nil {
		return nil, nil, err
	}

	c.Stdin = readline.NewCancelableStdin(c.Stdin)

	if s.IsVimMode {
		c.VimMode = true
	}

	c.HistoryLimit = -1
	c.UniqueEditLine = true

	rl, err := readline.NewEx(c)
	if err != nil {
		return nil, nil, err
	}

	rl.Write([]byte(hideCursor))
	sb := screenbuf.New(rl)

	cur := NewCursor("", s.Pointer, false)

	canSearch := s.Searcher != nil
	searchMode := s.StartInSearchMode
	s.list.SetCursor(cursorPos)
	s.list.SetStart(scroll)

	c.SetListener(func(line []rune, pos int, key rune) ([]rune, int, bool) {
		switch {
		case key == KeyEnter:
			return nil, 0, false
		case key == s.Keys.Next.Code || (key == 'j' && !searchMode):
			s.list.Next()
		case key == s.Keys.Prev.Code || (key == 'k' && !searchMode):
			s.list.Prev()
		case key == s.Keys.DiveIn.Code:
			s.list.DiveIn()
		case key == s.Keys.DiveOut.Code:
			s.list.DiveOut()
		case key == s.Keys.Search.Code:
			if !canSearch {
				break
			}

			if searchMode {
				searchMode = false
				cur.Replace("")
				s.list.CancelSearch()
			} else {
				searchMode = true
			}
		case key == KeyBackspace || key == KeyCtrlH:
			if !canSearch || !searchMode {
				break
			}

			cur.Backspace()
			if len(cur.Get()) > 0 {
				s.list.Search(cur.Get())
			} else {
				s.list.CancelSearch()
			}
		case key == s.Keys.PageUp.Code || (key == 'h' && !searchMode):
			s.list.DiveOut()
		case key == s.Keys.PageDown.Code || (key == 'l' && !searchMode):
			s.list.DiveIn()
		default:
			if canSearch && searchMode {
				cur.Update(string(line))
				s.list.Search(cur.Get())
			}
		}

		if searchMode {
			header := SearchPrompt + cur.Format()
			sb.WriteString(header)
		} else if !s.HideHelp {
			help := s.renderHelp(canSearch)
			sb.Write(help)
		}

		label := render(s.Templates.label, s.Label)
		sb.Write(label)

		items, idx := s.list.Items()
		last := len(items) - 1

		for i, item := range items {
			page := " "
			switch i {
			case 0:
				if s.list.CanPageUp() {
					page = "↑"
				} else {
					page = string(top)
				}
			case last:
				if s.list.CanPageDown() {
					page = "↓"
				}
			}

			output := []byte(page + " ")

			if i == idx {
				output = append(output, render(s.Templates.active, item)...)
			} else {
				output = append(output, render(s.Templates.inactive, item)...)
			}

			sb.Write(output)
		}

		if idx == multidimlist.NotFound {
			sb.WriteString("")
			sb.WriteString("No results")
		} else {
			active := items[idx]
			details := s.renderDetails(active)
			for _, d := range details {
				sb.Write(d)
			}
		}

		sb.Flush()

		return nil, 0, true
	})

	for {
		_, err = rl.Readline()

		if err != nil {
			switch {
			case err == readline.ErrInterrupt, err.Error() == "Interrupt":
				err = ErrInterrupt
			case err == io.EOF:
				err = ErrEOF
			}
			break
		}

		_, idx := s.list.Items()
		if idx != multidimlist.NotFound {
			break
		}
	}

	if err != nil {
		if err.Error() == "Interrupt" {
			err = ErrInterrupt
		}
		sb.Reset()
		sb.WriteString("")
		sb.Flush()
		rl.Write([]byte(showCursor))
		rl.Close()
		return nil, nil, err
	}

	items, idx := s.list.Items()
	item := items[idx]

	if s.HideSelected {
		clearScreen(sb)
	} else {
		sb.Reset()
		sb.Write(render(s.Templates.selected, item))
		sb.Flush()
	}

	rl.Write([]byte(showCursor))
	rl.Close()

	return s.list.Index(), item, err
}

func (s *MultidimSelect) setKeys() {
	if s.Keys != nil {
		return
	}
	s.Keys = &MultidimSelectKeys{
		Prev:    Key{Code: KeyPrev, Display: KeyPrevDisplay},
		Next:    Key{Code: KeyNext, Display: KeyNextDisplay},
		DiveIn:  Key{Code: KeyForward, Display: "→"},
		DiveOut: Key{Code: KeyBackward, Display: "←"},
		Search:  Key{Code: '/', Display: "/"},
	}
}

func (s *MultidimSelect) prepareTemplates() error {
	tpls := s.Templates
	if tpls == nil {
		tpls = &MultidimSelectTemplates{}
	}

	if tpls.FuncMap == nil {
		tpls.FuncMap = FuncMap
	}

	if tpls.Label == "" {
		tpls.Label = fmt.Sprintf("%s {{.}}: ", IconInitial)
	}

	tpl, err := template.New("").Funcs(tpls.FuncMap).Parse(tpls.Label)
	if err != nil {
		return err
	}
	tpls.label = tpl

	if tpls.Active == "" {
		tpls.Active = fmt.Sprintf(`
            {{- if isSlice . -}}
                %s {{ joinSlice " & " . | underline }}
            {{- else -}}
                %s {{ . | underline }}
            {{- end -}}
        `, IconSelect, IconSelect)
	}

	tpl, err = template.New("").Funcs(tpls.FuncMap).Parse(tpls.Active)
	if err != nil {
		return err
	}
	tpls.active = tpl

	if tpls.Inactive == "" {
		tpls.Inactive = `
            {{- if isSlice . -}}
                    {{ joinSlice " & " . }}
            {{- else -}}
                    {{ . }}
            {{- end -}}
        `
	}

	tpl, err = template.New("").Funcs(tpls.FuncMap).Parse(tpls.Inactive)
	if err != nil {
		return err
	}
	tpls.inactive = tpl

	if tpls.Selected == "" {
		tpls.Selected = fmt.Sprintf(`
			{{ if isSlice . }}
				{{ "%s" | green}} {{ joinSlice " & " . }}
			{{ else }}
			 	{{ "%s" | green}} {{ . }}
			{{ end }}
		`, IconGood, IconGood)
	}

	tpl, err = template.New("").Funcs(tpls.FuncMap).Parse(tpls.Selected)
	if err != nil {
		return err
	}
	tpls.selected = tpl

	if tpls.Details != "" {
		tpl, err = template.New("").Funcs(tpls.FuncMap).Parse(tpls.Details)
		if err != nil {
			return err
		}
		tpls.details = tpl
	}

	if tpls.Help == "" {
		tpls.Help = fmt.Sprintf(`{{ "Use the arrow keys to navigate:" | faint }} {{ .NextKey | faint }} ` +
			`{{ .PrevKey | faint }} {{ .PageDownKey | faint }} {{ .PageUpKey | faint }} ` +
			`{{ "Dimensions:" | faint }} {{ .DiveInKey | faint }} {{ .DiveOutKey | faint }} ` +
			`{{ if .Search }} {{ "and" | faint }} {{ .SearchKey | faint }} {{ "toggles search" | faint }}{{ end }}`)
	}

	tpl, err = template.New("").Funcs(tpls.FuncMap).Parse(tpls.Help)
	if err != nil {
		return err
	}
	tpls.help = tpl

	s.Templates = tpls

	return nil
}

func (s *MultidimSelect) renderDetails(item interface{}) [][]byte {
	if s.Templates.details == nil {
		return nil
	}

	var buf bytes.Buffer
	w := tabwriter.NewWriter(&buf, 0, 0, 8, ' ', 0)

	err := s.Templates.details.Execute(w, item)
	if err != nil {
		fmt.Fprintf(w, "%v", item)
	}

	w.Flush()
	output := buf.Bytes()
	return bytes.Split(output, []byte("\n"))
}

func (s *MultidimSelect) renderHelp(search bool) []byte {
	keys := struct {
		NextKey     string
		PrevKey     string
		PageDownKey string
		PageUpKey   string
		DiveInKey   string
		DiveOutKey  string
		SearchKey   string
		Search      bool
	}{
		NextKey:     s.Keys.Next.Display,
		PrevKey:     s.Keys.Prev.Display,
		PageDownKey: s.Keys.PageDown.Display,
		PageUpKey:   s.Keys.PageUp.Display,
		DiveInKey:   s.Keys.DiveIn.Display,
		DiveOutKey:  s.Keys.DiveOut.Display,
		SearchKey:   s.Keys.Search.Display,
		Search:      search,
	}

	return render(s.Templates.help, keys)
}

// for hijacking stdin
type stdinWrapper struct {
	original io.Reader
	onRead   func([]byte) (bool, error) // Callback for intercepting reads
}

func newStdinWrapper(original io.Reader, onRead func([]byte) (bool, error)) *stdinWrapper {
	if original == nil {
		original = readline.Stdin
	}

	return &stdinWrapper{
		original: original,
		onRead:   onRead,
	}
}

func (w *stdinWrapper) Read(p []byte) (n int, err error) {
	// First read from original stdin
	n, err = w.original.Read(p)
	if err != nil {
		return n, err
	}

	// If callback is set, process the input
	if w.onRead != nil {
		// Call callback to check if input is allowed
		if shouldPass, callbackErr := w.onRead(p[:n]); !shouldPass {
			if callbackErr != nil {
				return 0, callbackErr
			}
			// If input is not allowed, return 0 length
			return 0, nil
		}
	}

	return n, err
}

func (w *stdinWrapper) Close() error {
	if closer, ok := w.original.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
