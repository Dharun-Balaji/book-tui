package tui

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dharuncs/novel/internal/core"
	"github.com/dharuncs/novel/internal/source"
	"github.com/dharuncs/novel/internal/storage"
	"github.com/dharuncs/novel/internal/tui/chapterlist"
	"github.com/dharuncs/novel/internal/tui/library"
	"github.com/dharuncs/novel/internal/tui/reader"
	"github.com/dharuncs/novel/internal/tui/search"
	"github.com/dharuncs/novel/internal/tui/settings"
)

type AppModel struct {
	state          ViewState
	libraryManager *core.LibraryManager
	progressMgr    *core.ProgressManager
	statsManager   *core.StatsManager
	registry       *source.Registry
	db             *storage.DB

	libraryModel     library.Model
	readerModel      reader.Model
	searchModel      search.Model
	chapterListModel chapterlist.Model
	settingsModel    settings.Model

	userSettings storage.UserSettings
	width        int
	height       int
	statusMsg    string
	err          error

	activeNovelID string
}

func NewAppModel(
	lm *core.LibraryManager,
	pm *core.ProgressManager,
	sm *core.StatsManager,
	reg *source.Registry,
	db *storage.DB,
) AppModel {
	var initialPlugin *source.Plugin
	if reg != nil {
		if plugins := reg.List(); len(plugins) > 0 {
			initialPlugin = plugins[0]
		}
	}

	searchM := search.New()
	if initialPlugin != nil {
		searchM = searchM.SetPlugin(initialPlugin)
	}

	userSet := storage.UserSettings{
		Theme:         "dark",
		LineWidth:     80,
		AutoSaveEvery: 5,
	}
	if db != nil {
		if s, err := db.GetSettings(); err == nil {
			userSet = s
		}
	}

	settingsM := settings.New().SetSettings(userSet)

	return AppModel{
		state:            ViewLibrary,
		libraryManager:   lm,
		progressMgr:      pm,
		statsManager:     sm,
		registry:         reg,
		db:               db,
		libraryModel:     library.New(),
		readerModel:      reader.New(),
		searchModel:      searchM,
		chapterListModel: chapterlist.New(),
		settingsModel:    settingsM,
		userSettings:     userSet,
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.loadLibraryCmd()
}

func (m AppModel) loadLibraryCmd() tea.Cmd {
	return func() tea.Msg {
		novels, err := m.libraryManager.ListLibrary()
		if err != nil {
			return AddNovelErrorMsg{Err: err}
		}
		return LibraryLoadedMsg{Novels: novels}
	}
}

func (m AppModel) saveProgressCmd(novelID, chapterID string, pIdx, sOffset, totalParas int, readSec int64) tea.Cmd {
	return func() tea.Msg {
		_ = m.progressMgr.SaveReadingState(
			novelID,
			chapterID,
			pIdx,
			sOffset,
			totalParas,
			readSec,
		)
		return nil
	}
}

func (m AppModel) saveSettingsCmd(set storage.UserSettings) tea.Cmd {
	return func() tea.Msg {
		if m.db != nil {
			_ = m.db.UpdateSettings(set)
		}
		return nil
	}
}

func (m AppModel) fetchChapterCmd(novelID, chapterID string, pIdx, sOffset int, isComplete bool) tea.Cmd {
	return func() tea.Msg {
		var plugin *source.Plugin
		if m.registry != nil {
			if n, err := m.libraryManager.GetNovel(novelID); err == nil {
				if p, ok := m.registry.Get(n.SourceID); ok {
					plugin = p
				}
			}
		}

		ch, err := m.libraryManager.GetChapterContent(context.Background(), plugin, chapterID)
		if err != nil {
			return ChapterErrorMsg{Err: err}
		}
		return ChapterLoadedMsg{
			Chapter:         ch,
			ParagraphIdx:    pIdx,
			ScrollOffset:    sOffset,
			IsNovelComplete: isComplete,
		}
	}
}

func (m AppModel) openNovelCmd(novelID string) tea.Cmd {
	return func() tea.Msg {
		state, err := m.progressMgr.GetResumeState(novelID)
		if err != nil {
			return ChapterErrorMsg{Err: err}
		}
		pIdx := 0
		sOffset := 0
		if state.Progress != nil {
			pIdx = state.Progress.ParagraphIdx
			sOffset = state.Progress.ScrollOffset
		}
		return m.fetchChapterCmd(novelID, state.Chapter.ID, pIdx, sOffset, state.IsNovelComplete)()
	}
}

func (m AppModel) addNovelCmd(sourceID, sourceURL string) tea.Cmd {
	return func() tea.Msg {
		var plugin *source.Plugin
		if m.registry != nil {
			if p, ok := m.registry.Get(sourceID); ok {
				plugin = p
			}
		}
		if plugin == nil {
			return AddNovelErrorMsg{Err: fmt.Errorf("source plugin %q not found", sourceID)}
		}

		novel, err := m.libraryManager.AddNovel(context.Background(), plugin, sourceURL, func(status string) {
			// Progress logs can be handled via status bar or channel
		})
		if err != nil {
			return AddNovelErrorMsg{Err: err}
		}
		return AddNovelSuccessMsg{Novel: novel}
	}
}

func (m AppModel) openChapterListCmd(novelID string) tea.Cmd {
	return func() tea.Msg {
		n, err := m.libraryManager.GetNovel(novelID)
		if err != nil {
			return ChapterErrorMsg{Err: err}
		}
		chapters, err := m.libraryManager.ListChapters(novelID)
		if err != nil {
			return ChapterErrorMsg{Err: err}
		}
		return struct {
			Novel    core.Novel
			Chapters []core.Chapter
		}{Novel: n, Chapters: chapters}
	}
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.libraryModel = m.libraryModel.SetSize(msg.Width, msg.Height)
		m.readerModel = m.readerModel.SetSize(msg.Width, msg.Height)
		m.searchModel = m.searchModel.SetSize(msg.Width, msg.Height)
		m.chapterListModel = m.chapterListModel.SetSize(msg.Width, msg.Height)
		m.settingsModel = m.settingsModel.SetSize(msg.Width, msg.Height)

	case SwitchViewMsg:
		m.state = msg.State
		if m.state == ViewSearch {
			m.refreshSearchPlugin()
		}

	case settings.SaveSettingsMsg:
		m.userSettings = msg.Settings
		cmds = append(cmds, m.saveSettingsCmd(msg.Settings))

	case search.AddNovelMsg:
		m.statusMsg = "Adding novel to library..."
		m.state = ViewLibrary
		cmds = append(cmds, m.addNovelCmd(msg.SourceID, msg.SourceURL))

	case chapterlist.SelectChapterMsg:
		m.state = ViewReader
		cmds = append(cmds, m.fetchChapterCmd(m.activeNovelID, msg.ChapterID, 0, 0, false))

	case struct {
		Novel    core.Novel
		Chapters []core.Chapter
	}:
		m.chapterListModel = m.chapterListModel.SetChapters(msg.Novel, msg.Chapters)
		m.state = ViewChapterList

	case reader.NextChapterMsg:
		chapters, err := m.libraryManager.ListChapters(msg.NovelID)
		if err == nil {
			var nextCh *core.Chapter
			for i, ch := range chapters {
				if ch.Number == msg.CurrentChapterNum && i+1 < len(chapters) {
					nextCh = &chapters[i+1]
					break
				}
			}
			if nextCh != nil {
				cmds = append(cmds, m.fetchChapterCmd(msg.NovelID, nextCh.ID, 0, 0, false))
			} else {
				m.statusMsg = "Reached final chapter"
			}
		}

	case reader.PrevChapterMsg:
		chapters, err := m.libraryManager.ListChapters(msg.NovelID)
		if err == nil {
			var prevCh *core.Chapter
			for i, ch := range chapters {
				if ch.Number == msg.CurrentChapterNum && i-1 >= 0 {
					prevCh = &chapters[i-1]
					break
				}
			}
			if prevCh != nil {
				cmds = append(cmds, m.fetchChapterCmd(msg.NovelID, prevCh.ID, 0, 0, false))
			} else {
				m.statusMsg = "Already at first chapter"
			}
		}

	case reader.SwitchToLibraryMsg:
		m.state = ViewLibrary

	case LibraryLoadedMsg:
		m.libraryModel.SetNovels(msg.Novels)

	case OpenNovelMsg:
		m.activeNovelID = msg.NovelID
		cmds = append(cmds, m.openNovelCmd(msg.NovelID))

	case struct{ NovelID string }: // Emitted by library item selection
		m.activeNovelID = msg.NovelID
		cmds = append(cmds, m.openNovelCmd(msg.NovelID))

	case AddNovelProgressMsg:
		m.statusMsg = msg.Status

	case AddNovelSuccessMsg:
		m.statusMsg = fmt.Sprintf("Added %s to library", msg.Novel.Title)
		m.state = ViewLibrary
		cmds = append(cmds, m.loadLibraryCmd())

	case AddNovelErrorMsg:
		m.err = msg.Err
		m.statusMsg = fmt.Sprintf("Error: %v", msg.Err)

	case ChapterLoadedMsg:
		m.readerModel = m.readerModel.SetChapter(
			m.activeNovelID,
			msg.Chapter,
			msg.ParagraphIdx,
			msg.ScrollOffset,
			msg.IsNovelComplete,
			m.userSettings.LineWidth,
			m.userSettings.AutoSaveEvery,
		).SetSize(m.width, m.height)
		m.state = ViewReader

	case ChapterErrorMsg:
		m.err = msg.Err
		m.statusMsg = fmt.Sprintf("Error loading chapter: %v", msg.Err)

	case SaveProgressMsg:
		cmds = append(cmds, m.saveProgressCmd(msg.NovelID, msg.ChapterID, msg.ParagraphIdx, msg.ScrollOffset, msg.TotalParas, msg.AdditionalReadSec))

	case reader.SaveProgressMsg:
		cmds = append(cmds, m.saveProgressCmd(msg.NovelID, msg.ChapterID, msg.ParagraphIdx, msg.ScrollOffset, msg.TotalParas, msg.AdditionalReadSec))

	case tea.KeyMsg:
		switch msg.String() {
		case "s":
			if m.state == ViewLibrary {
				m.state = ViewSearch
				m.refreshSearchPlugin()
			}
		case "c":
			if m.state == ViewReader {
				cmds = append(cmds, m.openChapterListCmd(m.activeNovelID))
			}
		case ",", "S":
			if m.state == ViewLibrary {
				m.state = ViewSettings
			}
		case "q", "esc":
			if m.state == ViewSettings {
				m.state = ViewLibrary
			}
		case "ctrl+c":
			if m.state == ViewReader {
				nID, cID, pIdx, sOff, tParas, rSec := m.readerModel.GetCurrentState()
				_ = m.progressMgr.SaveReadingState(nID, cID, pIdx, sOff, tParas, rSec)
			}
			return m, tea.Quit
		}
	}

	// Delegate update to sub-model based on active ViewState
	switch m.state {
	case ViewLibrary:
		var cmd tea.Cmd
		m.libraryModel, cmd = m.libraryModel.Update(msg)
		cmds = append(cmds, cmd)

	case ViewReader:
		var cmd tea.Cmd
		m.readerModel, cmd = m.readerModel.Update(msg)
		cmds = append(cmds, cmd)

	case ViewSearch:
		var cmd tea.Cmd
		m.searchModel, cmd = m.searchModel.Update(msg)
		cmds = append(cmds, cmd)

	case ViewChapterList:
		var cmd tea.Cmd
		m.chapterListModel, cmd = m.chapterListModel.Update(msg)
		cmds = append(cmds, cmd)

	case ViewSettings:
		var cmd tea.Cmd
		m.settingsModel, cmd = m.settingsModel.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *AppModel) refreshSearchPlugin() {
	if m.registry != nil && m.searchModel.Plugin() == nil {
		if plugins := m.registry.List(); len(plugins) > 0 {
			m.searchModel = m.searchModel.SetPlugin(plugins[0])
		}
	}
}

func (m AppModel) View() string {
	switch m.state {
	case ViewLibrary:
		return m.libraryModel.View()
	case ViewReader:
		return m.readerModel.View()
	case ViewSearch:
		return m.searchModel.View()
	case ViewChapterList:
		return m.chapterListModel.View()
	case ViewSettings:
		return m.settingsModel.View()
	default:
		return "Unknown view"
	}
}
