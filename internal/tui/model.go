package tui

import (
	"context"
	"sort"
	"strings"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/pscyk/instagram-scraper/internal/snapshot"
)

// Model owns a bounded set of snapshots; no command performs network or scraper work.
type Model struct {
	items                                 []snapshot.Snapshot
	visible                               []int
	input                                 textinput.Model
	viewport                              viewport.Model
	width, height, selected, filter, sort int
	detail, help                          bool
	query                                 string
	ctx                                   context.Context
	reel                                  int
	reelOffsets                           []int
	previewNote                           string
	previewGeneration                     uint64
}

// New prepares the profile catalog.
func New(items []snapshot.Snapshot) *Model {
	input := textinput.New()
	input.Prompt = "/ "
	input.Placeholder = "Search handles, names, or category"
	input.CharLimit = 100
	input.SetVirtualCursor(true)
	model := &Model{items: items, input: input, viewport: viewport.New(),
		width: 100, height: 32, ctx: context.Background()}
	model.applyFilters()
	return model
}

func (m *Model) Init() tea.Cmd { return nil }

// WithContext connects native image previews to the command's shutdown signal.
func (m *Model) WithContext(ctx context.Context) *Model {
	m.ctx = ctx
	return m
}

func (m *Model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	switch message := message.(type) {
	case imageReadyMsg:
		return m, m.imageReady(message)
	case previewDoneMsg:
		m.previewNote = ""
		if message.err != nil {
			m.previewNote = message.err.Error()
		}
	case tea.WindowSizeMsg:
		m.width, m.height = max(1, min(message.Width, 300)), max(1, min(message.Height, 120))
		m.input.SetWidth(max(1, m.width-12))
		m.refreshDetail()
	case tea.KeyPressMsg:
		if message.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.input.Focused() {
			return m, m.searchKey(message)
		}
		return m, m.key(message)
	default:
		if m.input.Focused() {
			var command tea.Cmd
			m.input, command = m.input.Update(message)
			return m, command
		}
	}
	return m, nil
}

func (m *Model) searchKey(message tea.KeyPressMsg) tea.Cmd {
	switch message.String() {
	case "enter":
		m.query = SafeText(m.input.Value(), 100)
		m.input.Blur()
		m.applyFilters()
	case "esc":
		m.input.SetValue(m.query)
		m.input.Blur()
	default:
		var command tea.Cmd
		m.input, command = m.input.Update(message)
		return command
	}
	return nil
}

func (m *Model) key(message tea.KeyPressMsg) tea.Cmd {
	switch message.String() {
	case "q":
		return tea.Quit
	case "?":
		m.previewGeneration++
		m.previewNote = ""
		m.help = !m.help
	case "esc", "backspace":
		m.previewGeneration++
		m.previewNote = ""
		m.detail, m.help = false, false
	case "/":
		m.previewGeneration++
		m.previewNote = ""
		m.detail, m.help = false, false
		return m.input.Focus()
	case "f":
		m.filter = (m.filter + 1) % 4
		m.applyFilters()
	case "s":
		m.sort = (m.sort + 1) % 3
		m.applyFilters()
	case "c":
		m.query, m.filter, m.sort = "", 0, 0
		m.input.Reset()
		m.applyFilters()
	case "enter", "right":
		if len(m.visible) > 0 {
			m.detail = true
			m.refreshDetail()
			m.viewport.GotoTop()
		}
	case "[":
		m.move(-1)
	case "]":
		m.move(1)
	case "i":
		return m.previewImage(false)
	case "t":
		return m.previewImage(true)
	case "n":
		m.selectReel(1)
	case "p":
		m.selectReel(-1)
	default:
		if m.detail {
			m.viewport, _ = m.viewport.Update(message)
		} else {
			switch message.String() {
			case "j", "down":
				m.move(1)
			case "k", "up":
				m.move(-1)
			case "g", "home":
				m.move(-len(m.visible))
			case "G", "end":
				m.move(len(m.visible))
			}
		}
	}
	return nil
}

func (m *Model) move(delta int) {
	m.selected = max(0, min(m.selected+delta, len(m.visible)-1))
	m.reel, m.previewNote = 0, ""
	m.previewGeneration++
	if m.detail {
		m.refreshDetail()
		m.viewport.GotoTop()
	}
}

func (m *Model) current() *snapshot.Snapshot {
	if len(m.visible) == 0 {
		return nil
	}
	return &m.items[m.visible[m.selected]]
}

func (m *Model) applyFilters() {
	m.visible = make([]int, 0, len(m.items))
	query := strings.ToLower(m.query)
	for index := range m.items {
		item := &m.items[index]
		text := item.Profile.Username + " " + item.Profile.FullName + " " + item.Profile.Category
		if !strings.Contains(strings.ToLower(text), query) || !m.matches(item) {
			continue
		}
		m.visible = append(m.visible, index)
	}
	sort.SliceStable(m.visible, func(i, j int) bool {
		a, b := &m.items[m.visible[i]], &m.items[m.visible[j]]
		if m.sort == 1 && nullableCount(a.Profile.Followers) != nullableCount(b.Profile.Followers) {
			return nullableCount(a.Profile.Followers) > nullableCount(b.Profile.Followers)
		}
		if m.sort == 2 && !a.Measurement().Equal(b.Measurement()) {
			return a.Measurement().After(b.Measurement())
		}
		return a.Profile.Username < b.Profile.Username
	})
	m.selected = 0
	m.reel, m.previewNote = 0, ""
	m.previewGeneration++
	if len(m.visible) == 0 {
		m.detail = false
	}
	m.refreshDetail()
}

func (m *Model) selectReel(delta int) {
	item := m.current()
	if !m.detail || item == nil || len(item.Reels) == 0 {
		return
	}
	m.previewGeneration++
	m.previewNote = ""
	m.reel = max(0, min(m.reel+delta, len(item.Reels)-1))
	m.refreshDetail()
	m.viewport.SetYOffset(m.reelOffsets[m.reel])
}

func (m *Model) matches(item *snapshot.Snapshot) bool {
	switch m.filter {
	case 1:
		return !item.Measurement().IsZero()
	case 2:
		return item.Measurement().IsZero()
	case 3:
		return item.Profile.IsPrivate != nil && !*item.Profile.IsPrivate
	default:
		return true
	}
}

func nullableCount(value *int64) int64 {
	if value == nil {
		return -1
	}
	return *value
}

func (m *Model) refreshDetail() {
	m.viewport.SetWidth(max(1, m.width-8))
	m.viewport.SetHeight(max(1, m.height-10))
	if item := m.current(); item != nil {
		m.viewport.SetContent(m.detailContent(item, max(1, m.width-8)))
	}
}
