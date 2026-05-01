package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yinheli/sshw"
)

const (
	chromeLines  = 4  // header + sep + body + sep + help
	maxBodyLines = 28 // cap list height on very tall terminals
)

type listState struct {
	items  []list.Item
	cursor int
	title  string
}

type model struct {
	list       list.Model
	cols       *columnWidths
	health     *healthState
	batch      *batchState
	help       help.Model
	mode       uiMode
	marks      map[*sshw.Node]struct{}
	stack      []listState
	selected   *sshw.Node
	quitting   bool
	width      int
	height     int
	roots      []*sshw.Node
	globalMode bool
	preGlobal  *listState
}

func newModel(nodes []*sshw.Node) *model {
	items := nodesToListItems(nodesToItems(nodes))
	cols := computeColumns(items)
	health := newHealthState()
	batch := newBatchState()
	marks := make(map[*sshw.Node]struct{})

	delegate := compactDelegate{cols: &cols, health: health, marks: marks}
	l := list.New(items, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Filter = multiKeywordFilter
	l.SetShowHelp(false)
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(colorPrimary)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(colorPrimary)

	l.KeyMap.GoToStart.SetKeys("left", "home", "g")
	l.KeyMap.GoToEnd.SetKeys("right", "end", "G")
	l.KeyMap.PrevPage.SetKeys("pgup")
	l.KeyMap.NextPage.SetKeys("pgdown")

	h := help.New()
	// Adopt the existing palette so help blends with the rest of the UI.
	h.Styles.ShortKey = helpKeyStyle
	h.Styles.FullKey = helpKeyStyle
	h.Styles.ShortDesc = helpDescStyle
	h.Styles.FullDesc = helpDescStyle
	h.Styles.ShortSeparator = helpDescStyle
	h.Styles.FullSeparator = helpDescStyle

	return &model{
		list:   l,
		cols:   &cols,
		health: health,
		batch:  batch,
		help:   h,
		marks:  marks,
		roots:  nodes,
	}
}

// activeKeys returns the help.KeyMap for the current mode. ShortHelp is
// the one-line footer; FullHelp shows when ? is pressed.
func (m *model) activeKeys() modeKeys {
	switch m.mode {
	case modeBatchPrompt:
		return modeKeys{
			short: []key.Binding{
				key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
				key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
			},
		}
	case modeBatchConfirm:
		if m.batch.dangerous != "" {
			return modeKeys{
				short: []key.Binding{
					key.NewBinding(key.WithKeys("enter"), key.WithHelp("type "+dangerConfirmPhrase, "confirm")),
					key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "edit cmd")),
				},
			}
		}
		return modeKeys{
			short: []key.Binding{
				key.NewBinding(key.WithKeys("y", "enter"), key.WithHelp("y/enter", "run")),
				key.NewBinding(key.WithKeys("n"), key.WithHelp("n/esc", "edit cmd")),
			},
		}
	case modeBatchRunning:
		return modeKeys{
			short: []key.Binding{
				key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc/ctrl+c", "cancel & show partial")),
			},
		}
	case modeBatchResults:
		return modeKeys{
			short: []key.Binding{
				keys.Up, keys.Down,
				keys.Enter,
				keys.BatchGroup,
				keys.BatchFilterFail,
				keys.BatchRerun,
				keys.BatchRerunFailed,
				keys.Back,
				keys.Help,
			},
			full: [][]key.Binding{
				{keys.Up, keys.Down, keys.Enter},
				{keys.BatchGroup, keys.BatchFilterFail},
				{keys.BatchRerun, keys.BatchRerunFailed},
				{keys.Back, keys.Help},
			},
		}
	case modeBatchDetail:
		return modeKeys{
			short: []key.Binding{
				key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch tab")),
				key.NewBinding(key.WithKeys("up", "down", "pgup", "pgdown"), key.WithHelp("↑↓/pgup/pgdn", "scroll")),
				keys.Back,
			},
		}
	}
	// modeList default. Enter's meaning is context-sensitive: marks > 0
	// flips it from "ssh into selected host" to "start batch flow on
	// marked hosts" — surface that in help so the change isn't silent.
	enterBinding := keys.Enter
	if len(m.marks) > 0 {
		enterBinding = key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", fmt.Sprintf("batch (%d marked)", len(m.marks))),
		)
	}
	return modeKeys{
		short: []key.Binding{
			keys.Up, keys.Down,
			enterBinding,
			keys.Select,
			keys.BatchRun,
			keys.HealthCheck,
			keys.GlobalPalette,
			keys.Back,
			keys.Help,
		},
		full: [][]key.Binding{
			{keys.Up, keys.Down, enterBinding},
			{keys.Select, keys.BatchRun},
			{keys.HealthCheck, keys.GlobalPalette},
			{keys.Back, keys.Quit, keys.Help},
		},
	}
}

// nodeForListItem extracts the underlying *sshw.Node from a list item if any.
func nodeForListItem(li list.Item) *sshw.Node {
	switch v := li.(type) {
	case item:
		return v.node
	case indexedLeafItem:
		return v.idx.Node
	}
	return nil
}

// visibleConnectable returns connectable nodes from the current list view, in display order.
func (m *model) visibleConnectable() []*sshw.Node {
	var nodes []*sshw.Node
	for _, li := range m.list.Items() {
		switch v := li.(type) {
		case item:
			if v.node.Connectable() {
				nodes = append(nodes, v.node)
			}
		case indexedLeafItem:
			nodes = append(nodes, v.idx.Node)
		}
	}
	return nodes
}

// batchTargets returns the nodes the next batch run should hit:
// the marked subset if any, otherwise every visible connectable host.
func (m *model) batchTargets() []*sshw.Node {
	if len(m.marks) > 0 {
		// Preserve display order from the current list view.
		visible := m.visibleConnectable()
		out := make([]*sshw.Node, 0, len(m.marks))
		for _, n := range visible {
			if _, ok := m.marks[n]; ok {
				out = append(out, n)
			}
		}
		// Fallback (e.g. marks made before navigating): include any leftover.
		if len(out) < len(m.marks) {
			seen := make(map[*sshw.Node]struct{}, len(out))
			for _, n := range out {
				seen[n] = struct{}{}
			}
			extras := make([]*sshw.Node, 0)
			for n := range m.marks {
				if _, ok := seen[n]; !ok {
					extras = append(extras, n)
				}
			}
			sort.Slice(extras, func(i, j int) bool { return extras[i].Name < extras[j].Name })
			out = append(out, extras...)
		}
		return out
	}
	return m.visibleConnectable()
}

// clearMarks resets the selection set.
func (m *model) clearMarks() {
	for k := range m.marks {
		delete(m.marks, k)
	}
}

// toggleMark flips the selection state for the host under the cursor (if connectable).
func (m *model) toggleMark() {
	n := nodeForListItem(m.list.SelectedItem())
	if n == nil || !n.Connectable() {
		return
	}
	if _, on := m.marks[n]; on {
		delete(m.marks, n)
	} else {
		m.marks[n] = struct{}{}
	}
}

func globalPaletteFilter(term string, targets []string) []list.Rank {
	trimmed := strings.TrimSpace(term)
	if trimmed == "" {
		return multiKeywordFilter("", targets)
	}
	if strings.Contains(trimmed, " ") {
		return multiKeywordFilter(trimmed, targets)
	}
	return list.DefaultFilter(trimmed, targets)
}

func (m *model) syncLayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}
	avail := m.height - chromeLines
	if avail < 1 {
		avail = 1
	}
	listH := max(3, min(maxBodyLines, avail))
	m.list.SetSize(m.width, listH)
}

func (m *model) setItems(items []list.Item, title string) tea.Cmd {
	cols := computeColumns(items)
	*m.cols = cols
	cmd := m.list.SetItems(items)
	m.syncLayout()
	return cmd
}

func (m *model) toggleGlobalPalette() tea.Cmd {
	if m.globalMode {
		return m.exitGlobalPalette()
	}
	m.preGlobal = &listState{
		items:  m.list.Items(),
		cursor: m.list.Index(),
		title:  m.list.Title,
	}
	hosts := FlattenLeaves(m.roots)
	items := make([]list.Item, len(hosts))
	for i := range hosts {
		items[i] = indexedLeafItem{idx: hosts[i]}
	}
	m.list.Filter = globalPaletteFilter
	m.globalMode = true
	cmd := m.setItems(items, "__global__")
	m.list.Select(0)
	m.list.ResetFilter()
	return cmd
}

func (m *model) exitGlobalPalette() tea.Cmd {
	if !m.globalMode || m.preGlobal == nil {
		return nil
	}
	prev := *m.preGlobal
	m.preGlobal = nil
	m.globalMode = false
	m.list.Filter = multiKeywordFilter
	cmd := m.setItems(prev.items, prev.title)
	m.list.Select(prev.cursor)
	m.list.ResetFilter()
	return cmd
}

func multiKeywordFilter(term string, targets []string) []list.Rank {
	var ranks []list.Rank
	for i, t := range targets {
		if matchMultiKeyword(term, t) {
			ranks = append(ranks, list.Rank{Index: i})
		}
	}
	return ranks
}

func matchMultiKeyword(input, content string) bool {
	input = strings.ToLower(input)
	content = strings.ToLower(content)
	if strings.Contains(input, " ") {
		for _, k := range strings.Split(input, " ") {
			k = strings.TrimSpace(k)
			if k != "" && !strings.Contains(content, k) {
				return false
			}
		}
		return true
	}
	return strings.Contains(content, input)
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncLayout()
		m.syncBatchLayout()
		return m, nil

	case spinner.TickMsg:
		var cmds []tea.Cmd
		if m.health.active {
			var cmd tea.Cmd
			m.health.spinner, cmd = m.health.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
		if m.batch.active {
			var cmd tea.Cmd
			m.batch.spinner, cmd = m.batch.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case healthCheckResultMsg:
		if msg.generation != m.health.generation {
			return m, nil
		}
		m.health.results[msg.node] = &healthResult{
			done: true, ok: msg.err == nil, latency: msg.latency, err: msg.err,
		}
		allDone := true
		for _, r := range m.health.results {
			if !r.done {
				allDone = false
				break
			}
		}
		if allDone {
			m.health.active = false
		}
		return m, nil

	case batchResultMsg:
		if msg.generation != m.batch.generation {
			return m, nil
		}
		m.batch.results[msg.node] = &batchTargetResult{done: true, res: msg.res}
		_, done, _, _ := m.batch.counts()
		if done >= len(m.batch.targets) {
			m.batch.active = false
			m.mode = modeBatchResults
			m.batch.resultIdx = 0
			m.persistRun()
		}
		return m, nil
	}

	// Top-level ctrl+c: in modeBatchRunning we soft-cancel and let the
	// user see partial results; everywhere else we quit (matches the
	// pre-WithoutSignalHandler default behavior so users with ctrl+c
	// muscle memory aren't surprised).
	if km, ok := msg.(tea.KeyMsg); ok && km.Type == tea.KeyCtrlC {
		if m.mode == modeBatchRunning {
			m.cancelRunningBatch()
			return m, nil
		}
		m.quitting = true
		return m, tea.Quit
	}

	// Per-mode input routing.
	switch m.mode {
	case modeBatchPrompt:
		return m.updateBatchPrompt(msg)
	case modeBatchConfirm:
		return m.updateBatchConfirm(msg)
	case modeBatchRunning:
		return m.updateBatchRunning(msg)
	case modeBatchResults:
		return m.updateBatchResults(msg)
	case modeBatchDetail:
		return m.updateBatchDetail(msg)
	}

	// modeList (default).
	if km, ok := msg.(tea.KeyMsg); ok {
		if m.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}
		return m.updateListKey(km)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m *model) updateListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil

	case key.Matches(msg, keys.HealthCheck):
		if !m.health.active {
			cmd := m.startHealthCheck()
			return m, cmd
		}

	case key.Matches(msg, keys.GlobalPalette):
		m.health.reset()
		m.clearMarks()
		cmd := m.toggleGlobalPalette()
		return m, cmd

	case key.Matches(msg, keys.Select):
		m.toggleMark()
		return m, nil

	case key.Matches(msg, keys.BatchRun):
		targets := m.batchTargets()
		if len(targets) == 0 {
			return m, nil
		}
		m.batch.reset()
		m.batch.targets = targets
		m.health.reset()
		m.mode = modeBatchPrompt
		m.batch.input.Reset()
		m.syncBatchLayout()
		return m, m.batch.input.Focus()

	case key.Matches(msg, keys.Enter):
		// If the user has explicitly marked some hosts, Enter operates on
		// that working set (start batch flow) rather than logging into the
		// host under the cursor. Matches the "I marked these for a reason"
		// mental model and lines up with ctrl+x; only difference is that
		// ctrl+x also fires on no-marks (uses visible set) while Enter
		// keeps its original SSH-login meaning when nothing is marked.
		if len(m.marks) > 0 {
			targets := m.batchTargets()
			if len(targets) == 0 {
				break
			}
			m.batch.reset()
			m.batch.targets = targets
			m.health.reset()
			m.mode = modeBatchPrompt
			m.batch.input.Reset()
			m.syncBatchLayout()
			return m, m.batch.input.Focus()
		}
		sel := m.list.SelectedItem()
		if sel == nil {
			break
		}
		switch v := sel.(type) {
		case indexedLeafItem:
			m.selected = v.idx.Node
			m.quitting = true
			return m, tea.Quit
		case item:
			node := v.node
			if len(node.Children) > 0 {
				m.health.reset()
				m.clearMarks()
				m.stack = append(m.stack, listState{
					items:  m.list.Items(),
					cursor: m.list.Index(),
					title:  m.list.Title,
				})
				childItems := nodesToListItems(nodesToItems(node.Children))
				cmd := m.setItems(childItems, node.Name)
				m.list.Select(0)
				m.list.Title = node.Name
				return m, cmd
			}
			m.selected = node
			m.quitting = true
			return m, tea.Quit
		}

	case key.Matches(msg, keys.Back):
		m.health.reset()
		m.clearMarks()
		if m.globalMode {
			cmd := m.exitGlobalPalette()
			return m, cmd
		}
		if len(m.stack) == 0 {
			m.quitting = true
			return m, tea.Quit
		}
		prev := m.stack[len(m.stack)-1]
		m.stack = m.stack[:len(m.stack)-1]
		cmd := m.setItems(prev.items, prev.title)
		m.list.Select(prev.cursor)
		m.list.Title = prev.title
		return m, cmd

	case key.Matches(msg, keys.Quit):
		m.quitting = true
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// updateBatchPrompt routes input while the user types the command.
func (m *model) updateBatchPrompt(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			m.batch.input.Blur()
			m.batch.reset()
			m.mode = modeList
			return m, nil
		case "enter":
			cmd := strings.TrimSpace(m.batch.input.Value())
			if cmd == "" {
				return m, nil
			}
			m.batch.cmdLine = cmd
			m.batch.input.Blur()
			// Decide danger gate up front so render & key handling agree.
			if matched, ok := dangerousMatch(cmd); ok {
				m.batch.dangerous = matched
				m.batch.confirmInput.Reset()
				m.batch.confirmFailed = false
				m.mode = modeBatchConfirm
				return m, m.batch.confirmInput.Focus()
			}
			m.batch.dangerous = ""
			m.mode = modeBatchConfirm
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.batch.input, cmd = m.batch.input.Update(msg)
	return m, cmd
}

// updateBatchConfirm handles either the simple y/n flow or the
// typed-override flow when the command is flagged as dangerous.
// In the safe (non-danger) path, both `y` and `enter` accept — Enter as
// the affirmative is what most users expect after pressing Enter to
// leave the prompt. The visible label still leads with `y` to keep the
// danger gate's typed-confirm flow visually distinct.
func (m *model) updateBatchConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.batch.dangerous != "" {
		return m.updateBatchConfirmDanger(msg)
	}
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch strings.ToLower(km.String()) {
	case "y", "enter":
		return m, m.startBatchRun()
	case "n", "esc":
		// Return to the prompt so the user can edit their command rather
		// than retyping it from scratch. Esc from prompt itself exits.
		m.batch.input.SetValue(m.batch.cmdLine)
		m.batch.input.CursorEnd()
		m.mode = modeBatchPrompt
		return m, m.batch.input.Focus()
	}
	return m, nil
}

// updateBatchConfirmDanger requires the literal phrase to proceed; on esc
// it returns to the prompt with the original command preserved so the
// user can edit out the dangerous part.
func (m *model) updateBatchConfirmDanger(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			m.batch.confirmInput.Blur()
			m.batch.dangerous = ""
			m.batch.confirmFailed = false
			m.batch.input.SetValue(m.batch.cmdLine)
			m.batch.input.CursorEnd()
			m.mode = modeBatchPrompt
			return m, m.batch.input.Focus()
		case "enter":
			if strings.TrimSpace(m.batch.confirmInput.Value()) == dangerConfirmPhrase {
				m.batch.confirmInput.Blur()
				return m, m.startBatchRun()
			}
			m.batch.confirmFailed = true
			m.batch.confirmInput.Reset()
			return m, nil
		}
	}
	// Clear the previous-mismatch banner as soon as the user types anything
	// new — keep the feedback honest to current state.
	m.batch.confirmFailed = false
	var cmd tea.Cmd
	m.batch.confirmInput, cmd = m.batch.confirmInput.Update(msg)
	return m, cmd
}

// updateBatchRunning handles input while commands are in flight.
// esc / ctrl+c soft-cancel: stop accepting new results, fill any
// unfinished hosts with a synthetic "cancelled" RunResult, transition
// to the results view so the user keeps whatever already came back.
// In-flight goroutines hold their SSH sessions until their per-host
// timeout fires; their result messages are dropped via generation gate.
func (m *model) updateBatchRunning(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		if km.String() == "esc" || km.Type == tea.KeyCtrlC {
			m.cancelRunningBatch()
			return m, nil
		}
	}
	return m, nil
}

// cancelRunningBatch is the soft-cancel transition used by both esc and
// ctrl+c during modeBatchRunning. It seals the in-flight run, marks any
// pending hosts as cancelled, and flips to modeBatchResults so the user
// can see what already came back instead of losing all output.
func (m *model) cancelRunningBatch() {
	m.batch.generation++
	m.batch.active = false
	for _, n := range m.batch.targets {
		r := m.batch.results[n]
		if r == nil || !r.done {
			m.batch.results[n] = &batchTargetResult{
				done: true,
				res: sshw.RunResult{
					ExitCode: -1,
					Err:      errBatchCancelled,
				},
			}
		}
	}
	m.batch.flash = "run cancelled — in-flight connections will close on timeout"
	m.mode = modeBatchResults
	m.batch.resultIdx = 0
	m.persistRun()
}

// updateBatchResults handles cursor movement, drill-down, rerun, and back-out.
func (m *model) updateBatchResults(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	visible := m.visibleResults()
	buckets := m.visibleBuckets()

	switch {
	case key.Matches(km, keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil

	case key.Matches(km, keys.BatchGroup):
		// Toggle grouped view; cursors stay independent so the user
		// doesn't lose their place when toggling back.
		m.batch.groupView = !m.batch.groupView
		if m.batch.groupView {
			if m.batch.bucketIdx >= len(buckets) {
				m.batch.bucketIdx = 0
			}
		}
		return m, nil

	case km.String() == "up" || km.String() == "k":
		if m.batch.groupView {
			if m.batch.bucketIdx > 0 {
				m.batch.bucketIdx--
			}
		} else if m.batch.resultIdx > 0 {
			m.batch.resultIdx--
		}
	case km.String() == "down" || km.String() == "j":
		if m.batch.groupView {
			if m.batch.bucketIdx < len(buckets)-1 {
				m.batch.bucketIdx++
			}
		} else if m.batch.resultIdx < len(visible)-1 {
			m.batch.resultIdx++
		}
	case km.String() == "home":
		if m.batch.groupView {
			m.batch.bucketIdx = 0
		} else {
			m.batch.resultIdx = 0
		}
	case km.String() == "end" || km.String() == "G":
		if m.batch.groupView {
			if len(buckets) > 0 {
				m.batch.bucketIdx = len(buckets) - 1
			}
		} else if len(visible) > 0 {
			m.batch.resultIdx = len(visible) - 1
		}
	case key.Matches(km, keys.BatchFilterFail):
		// Toggle failed-only. Clamp cursor into the new visible range.
		if m.batch.resultsFilter == 0 {
			m.batch.resultsFilter = 1
		} else {
			m.batch.resultsFilter = 0
		}
		newVisible := m.visibleResults()
		if len(newVisible) == 0 {
			m.batch.flash = "no failed hosts — press f to clear filter"
			m.batch.resultIdx = 0
		} else {
			if m.batch.resultIdx >= len(newVisible) {
				m.batch.resultIdx = len(newVisible) - 1
			}
			m.batch.flash = ""
		}
		return m, nil
	case key.Matches(km, keys.BatchRerunFailed):
		failed := m.failedTargets()
		if len(failed) == 0 {
			m.batch.flash = "no failed hosts to rerun"
			return m, nil
		}
		m.batch.targets = failed
		m.batch.resultsFilter = 0
		m.batch.flash = ""
		return m, m.startBatchRun()
	case key.Matches(km, keys.Enter):
		if m.batch.groupView {
			if len(buckets) == 0 {
				return m, nil
			}
			if m.batch.bucketIdx >= len(buckets) {
				m.batch.bucketIdx = len(buckets) - 1
			}
			bk := buckets[m.batch.bucketIdx]
			m.openDetail(bk.hosts[0], bk.hosts, bk.exemplar)
			return m, nil
		}
		if len(visible) == 0 {
			return m, nil
		}
		if m.batch.resultIdx >= len(visible) {
			m.batch.resultIdx = len(visible) - 1
		}
		n := visible[m.batch.resultIdx]
		r := m.batch.results[n]
		if r == nil {
			return m, nil
		}
		m.openDetail(n, nil, r.res)
		return m, nil
	case key.Matches(km, keys.BatchRerun):
		// Always rerun on the original target set; if the filter shrunk
		// what's visible, that's purely a view concern.
		m.batch.resultsFilter = 0
		return m, m.startBatchRun()
	case km.String() == "esc":
		m.batch.reset()
		m.clearMarks()
		m.mode = modeList
		return m, nil
	}
	return m, nil
}

// visibleResults returns the subset of targets shown in the results view,
// honoring the failed-only filter. Order is preserved from m.batch.targets.
func (m *model) visibleResults() []*sshw.Node {
	if m.batch.resultsFilter == 0 {
		return m.batch.targets
	}
	out := make([]*sshw.Node, 0, len(m.batch.targets))
	for _, n := range m.batch.targets {
		if isFailed(m.batch.results[n]) {
			out = append(out, n)
		}
	}
	return out
}

// visibleBuckets returns the buckets for the currently visible result subset.
// Composes with the failed-only filter — grouping inside a filtered view
// only buckets the failed hosts.
func (m *model) visibleBuckets() []bucket {
	return computeBuckets(m.visibleResults(), m.batch.results)
}

// failedTargets returns just the failed nodes from the last completed run.
func (m *model) failedTargets() []*sshw.Node {
	out := make([]*sshw.Node, 0)
	for _, n := range m.batch.targets {
		if isFailed(m.batch.results[n]) {
			out = append(out, n)
		}
	}
	return out
}

// openDetail seeds detail-view state and switches mode. bucketHosts is
// non-nil when drilling in from a grouped bucket so the header can list
// every host that shares this output.
func (m *model) openDetail(node *sshw.Node, bucketHosts []*sshw.Node, r sshw.RunResult) {
	m.batch.detailNode = node
	m.batch.bucketHosts = bucketHosts
	m.batch.detailRes = r
	m.batch.detailTab = 0
	m.batch.detail.SetContent(detailTabContent(0, r))
	m.batch.detail.GotoTop()
	m.mode = modeBatchDetail
}

// setDetailTab switches the active tab and rewrites the viewport content.
// Scroll position resets to top so a small stderr doesn't inherit a stdout
// scroll offset.
func (m *model) setDetailTab(tab int) {
	if tab < 0 || tab >= len(detailTabs) {
		return
	}
	m.batch.detailTab = tab
	m.batch.detail.SetContent(detailTabContent(tab, m.batch.detailRes))
	m.batch.detail.GotoTop()
}

// updateBatchDetail handles tab switching, viewport scrolling, and esc.
func (m *model) updateBatchDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "esc":
			m.batch.detailNode = nil
			m.batch.bucketHosts = nil
			m.batch.detailRes = sshw.RunResult{}
			m.batch.detailTab = 0
			m.mode = modeBatchResults
			return m, nil
		case "tab":
			m.setDetailTab((m.batch.detailTab + 1) % len(detailTabs))
			return m, nil
		case "shift+tab":
			m.setDetailTab((m.batch.detailTab - 1 + len(detailTabs)) % len(detailTabs))
			return m, nil
		case "1":
			m.setDetailTab(0)
			return m, nil
		case "2":
			m.setDetailTab(1)
			return m, nil
		case "3":
			m.setDetailTab(2)
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.batch.detail, cmd = m.batch.detail.Update(msg)
	return m, cmd
}

// startBatchRun kicks off concurrent SSH command execution against m.batch.targets.
// Returns a tea.Cmd that includes the spinner tick plus one runCommandCmd per host.
func (m *model) startBatchRun() tea.Cmd {
	if len(m.batch.targets) == 0 || m.batch.cmdLine == "" {
		return nil
	}
	m.batch.results = make(map[*sshw.Node]*batchTargetResult, len(m.batch.targets))
	for _, n := range m.batch.targets {
		m.batch.results[n] = &batchTargetResult{}
	}
	m.batch.generation++
	m.batch.active = true
	m.batch.runStarted = time.Now()
	m.batch.runID = ""
	m.batch.logDir = ""
	m.batch.logErr = nil
	m.mode = modeBatchRunning
	gen := m.batch.generation

	cmds := []tea.Cmd{m.batch.spinner.Tick}
	for _, n := range m.batch.targets {
		cmds = append(cmds, runCommandCmd(n, m.batch.cmdLine, gen, m.batch.sem))
	}
	return tea.Batch(cmds...)
}

// persistRun writes a RunRecord to disk via sshw.WriteRun. Best-effort:
// failure is stashed in batch.logErr and surfaced in the results footer
// but never aborts the flow.
func (m *model) persistRun() {
	if len(m.batch.targets) == 0 {
		return
	}
	results := make(map[*sshw.Node]sshw.RunResult, len(m.batch.results))
	for n, r := range m.batch.results {
		if r != nil {
			results[n] = r.res
		}
	}
	rec := sshw.RunRecord{
		Cmd:      m.batch.cmdLine,
		Targets:  m.batch.targets,
		Results:  results,
		Started:  m.batch.runStarted,
		Finished: time.Now(),
	}
	runID, logDir, err := sshw.WriteRun(rec)
	m.batch.runID = runID
	m.batch.logDir = logDir
	m.batch.logErr = err
}

// syncBatchLayout sizes the textinput, progress bar, and viewport to the current window.
func (m *model) syncBatchLayout() {
	if m.width <= 0 {
		return
	}
	m.batch.input.Width = max(20, m.width-6)
	m.batch.progress.Width = max(20, m.width-4)

	bodyH := max(5, m.height-chromeLines-3)
	if m.batch.detail.Width == 0 || m.batch.detail.Height == 0 {
		m.batch.detail = viewport.New(m.width-2, bodyH)
	} else {
		m.batch.detail.Width = m.width - 2
		m.batch.detail.Height = bodyH
	}
}

func (m *model) View() string {
	if m.quitting {
		return ""
	}
	if m.width == 0 {
		return "Loading..."
	}

	switch m.mode {
	case modeBatchPrompt:
		return m.renderBatchPrompt()
	case modeBatchConfirm:
		return m.renderBatchConfirm()
	case modeBatchRunning:
		return m.renderBatchRunning()
	case modeBatchResults:
		return m.renderBatchResults()
	case modeBatchDetail:
		return m.renderBatchDetail()
	}

	header := m.renderHeader()
	topSep := separatorStyle.Render(strings.Repeat("─", m.width))
	body := m.list.View()
	botSep := separatorStyle.Render(strings.Repeat("─", m.width))
	help := m.renderHelp()

	return lipgloss.JoinVertical(lipgloss.Left, header, topSep, body, botSep, help)
}

func (m *model) startHealthCheck() tea.Cmd {
	var nodes []*sshw.Node
	for _, li := range m.list.Items() {
		switch v := li.(type) {
		case item:
			if v.node.Connectable() {
				nodes = append(nodes, v.node)
			}
		case indexedLeafItem:
			nodes = append(nodes, v.idx.Node)
		}
	}
	if len(nodes) == 0 {
		return nil
	}

	m.health.reset()
	m.health.generation++
	m.health.active = true
	gen := m.health.generation

	cmds := []tea.Cmd{m.health.spinner.Tick}
	for _, n := range nodes {
		m.health.results[n] = &healthResult{}
		cmds = append(cmds, checkHostCmd(n, gen))
	}
	return tea.Batch(cmds...)
}

func (m *model) renderHealthSummary() string {
	total, ok, fail := m.health.counts()
	if total == 0 {
		return ""
	}
	if m.health.active {
		return m.health.spinner.View() + " " +
			healthCheckingStyle.Render(fmt.Sprintf("checking %d/%d", ok+fail, total))
	}
	var parts []string
	if ok > 0 {
		parts = append(parts, healthOKStyle.Render(fmt.Sprintf("✓ %d", ok)))
	}
	if fail > 0 {
		parts = append(parts, healthFailStyle.Render(fmt.Sprintf("✗ %d", fail)))
	}
	return strings.Join(parts, "  ")
}

func (m *model) renderHeader() string {
	parts := make([]string, 0, len(m.stack)+3)
	parts = append(parts, headerTitleStyle.Render("sshw"))
	if m.globalMode {
		parts = append(parts, headerPathStyle.Render("GLOBAL"))
	} else {
		for _, s := range m.stack {
			parts = append(parts, headerPathStyle.Render(s.title))
		}
		if m.list.Title != "" && m.list.Title != "sshw" && m.list.Title != "__global__" {
			parts = append(parts, headerPathStyle.Render(m.list.Title))
		}
	}

	path := strings.Join(parts, headerSepStyle.Render(" ❯ "))
	rightInfo := m.renderRightInfo()
	rightWidth := lipgloss.Width(rightInfo)
	pathWidth := lipgloss.Width(path)
	gap := max(1, m.width-pathWidth-rightWidth-2)

	return " " + path + strings.Repeat(" ", gap) + rightInfo
}

// renderRightInfo composes the right-aligned status: marks count, batch progress, healthcheck progress.
func (m *model) renderRightInfo() string {
	var parts []string
	if n := len(m.marks); n > 0 && m.mode == modeList {
		parts = append(parts, batchMarkOnStyle.Render(fmt.Sprintf("[%d marked]", n)))
	}
	if m.batch.active {
		total, done, _, _ := m.batch.counts()
		parts = append(parts,
			m.batch.spinner.View()+" "+batchHintStyle.Render(fmt.Sprintf("running %d/%d", done, total)))
	}
	if h := m.renderHealthSummary(); h != "" {
		parts = append(parts, h)
	}
	return strings.Join(parts, "  ")
}

// renderHelp renders the bottom-bar help via bubbles/help. The active
// keymap is mode-aware; pressing ? swaps short ↔ full.
func (m *model) renderHelp() string {
	if m.width > 0 {
		m.help.Width = m.width - 2
	}
	return " " + m.help.View(m.activeKeys())
}

// Run starts the TUI and returns the selected node, or nil if the user quit.
func Run(nodes []*sshw.Node) (*sshw.Node, error) {
	m := newModel(nodes)
	// WithoutSignalHandler so ctrl+c arrives as a tea.KeyMsg{Type: KeyCtrlC}
	// and we can route it per-mode (soft-cancel during batch run vs. quit
	// elsewhere) instead of being unconditionally killed by the default
	// SIGINT handler.
	p := tea.NewProgram(m, tea.WithoutSignalHandler())
	result, err := p.Run()
	if err != nil {
		return nil, err
	}
	finalModel := result.(*model)
	return finalModel.selected, nil
}
