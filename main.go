package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle        = lipgloss.NewStyle().Align(lipgloss.Center)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	bindingsStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)

	listStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Width(45).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	centerStyle = lipgloss.NewStyle().
			Align(lipgloss.Center).
			AlignVertical(lipgloss.Center)

	formStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	tableStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 1)

	tableTitleStyle = lipgloss.NewStyle().
			Align(lipgloss.Center).
			Bold(true).
			MarginBottom(0)

	tableContainerStyle = lipgloss.NewStyle().Align(lipgloss.Center)

	buttonsStyle = lipgloss.NewStyle().
			Align(lipgloss.Center).
			MarginTop(0).
			MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Width(10).
			MarginRight(1)

	inputFieldStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 0)

	focusedInputFieldStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(lipgloss.Color("170")).
				Padding(0, 0)

	errorInputFieldStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(lipgloss.Color("196")).
				Padding(0, 0)

	errorMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				MarginTop(1).
				MarginBottom(1)

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)

	activeButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("170")).
				Padding(0, 1)

	defaultColumns = []table.Column{
		{Title: "Title", Width: 30},
		{Title: "Password", Width: 30},
	}
)

// Global store struct
type Store struct {
	data map[string]interface{}
	mu   sync.RWMutex
}

// Global instance
var GlobalStore = &Store{
	data: make(map[string]interface{}),
}

// Methods
func (s *Store) Set(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[key] = value
}

func (s *Store) Get(key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	value, exists := s.data[key]
	return value, exists
}

func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

type item string

func (i item) FilterValue() string { return "" }

type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i)
	fn := itemStyle.Render

	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type state int

const (
	stateMainMenu state = iota
	stateFileList
	statePasswordInput
	stateKeyBindings
	stateAddDbForm
	stateDbView
	stateAddRecordForm
	stateError
)

type model struct {
	state                state
	list                 list.Model
	fileList             list.Model
	passwordInput        textinput.Model
	titleInput           textinput.Model
	dbTitleInput         textinput.Model
	dbPasswordInput      textinput.Model
	choice               string
	fileChoice           string
	quitting             bool
	width                int
	height               int
	titleInputError      bool
	passwordInputError   bool
	dbTitleInputError    bool
	dbPasswordInputError bool
	table                table.Model
	dbData               []table.Row
	activeButton         int
	errorMessage         string
}

// Create styled table
func createStyledTable(columns []table.Column, rows []table.Row) table.Model {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(14),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Align(lipgloss.Center)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("170"))

	t.SetStyles(s)
	return t
}

// Update table data
func (m *model) updateTable() {
	m.table.SetRows(m.dbData)
}

// Initialize model with dynamic list height
func initialModel() model {
	items := []list.Item{
		item("Add db"),
		item("Open db"),
		item("Manage dbs"),
		item("Key bindings"),
	}

	listHeight := len(items) + 8
	const defaultWidth = 30

	l := list.New(items, itemDelegate{}, defaultWidth, listHeight)
	l.Title = "Password Manager"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	// Initialize table
	t := createStyledTable(defaultColumns, []table.Row{})

	return model{
		state:  stateMainMenu,
		list:   l,
		width:  80,
		height: 24,
		table:  t,
		dbData: []table.Row{},
	}
}

// Create file list
func createFileList() (list.Model, error) {
	var files []list.Item
	config := ReadConfigFile()

	data, err := ReadDBsFolder(config.DBsFolder)
	if err != nil {
		return list.Model{}, fmt.Errorf("failed to read DBs folder: %v", err)
	}

	for _, filename := range data {
		files = append(files, item(filename))
	}

	// Compact height calculation
	listHeight := min(len(files)+10, 15)
	const defaultWidth = 30

	fileList := list.New(files, itemDelegate{}, defaultWidth, listHeight)
	fileList.Title = "Select file"
	fileList.SetShowStatusBar(true)
	fileList.SetFilteringEnabled(true) // Enable filtering
	fileList.Styles.Title = titleStyle
	fileList.Styles.PaginationStyle = paginationStyle
	fileList.Styles.HelpStyle = helpStyle

	fileList.SetSize(defaultWidth, listHeight)
	return fileList, nil
}

// Create password input field
func createPasswordInput() textinput.Model {
	input := textinput.New()
	input.Placeholder = "Enter password"
	input.Focus()
	input.CharLimit = 156
	input.Width = 30
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '*'
	return input
}

// Create title input field
func createTitleInput() textinput.Model {
	input := textinput.New()
	input.Placeholder = "Enter database title"
	input.Focus()
	input.CharLimit = 100
	input.Width = 30
	return input
}

// Create DB record title input field
func createDbTitleInput() textinput.Model {
	input := textinput.New()
	input.Placeholder = "Enter record title"
	input.Focus()
	input.CharLimit = 100
	input.Width = 30
	return input
}

// Create DB record password input field
func createDbPasswordInput() textinput.Model {
	input := textinput.New()
	input.Placeholder = "Enter record password"
	input.Focus()
	input.CharLimit = 156
	input.Width = 30
	input.EchoMode = textinput.EchoNormal
	return input
}

// Handle global keys
func (m *model) handleGlobalKeys(keypress string) (tea.Model, tea.Cmd) {
	// Allow filtering to work
	if m.state == stateFileList && m.fileList.FilterState() != list.Unfiltered {
		return nil, nil
	}

	if m.state == stateAddDbForm || m.state == statePasswordInput || m.state == stateAddRecordForm {
		return nil, nil
	}

	switch keypress {
	case "q", "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "m":
		return m.resetToMainMenu(), nil
	case "b":
		return m.goBack(), nil
	case "e":
		if m.state == stateError {
			m.state = stateMainMenu
			m.errorMessage = ""
			return m, nil
		}
	}
	return nil, nil
}

// Reset to main menu
func (m *model) resetToMainMenu() *model {
	m.state = stateMainMenu
	m.choice = ""
	m.fileChoice = ""
	m.passwordInput = textinput.Model{}
	m.titleInput = textinput.Model{}
	m.dbTitleInput = textinput.Model{}
	m.dbPasswordInput = textinput.Model{}
	m.titleInputError = false
	m.passwordInputError = false
	m.dbTitleInputError = false
	m.dbPasswordInputError = false
	m.dbData = []table.Row{}
	m.activeButton = 0
	m.errorMessage = ""
	return m
}

// Go back
func (m *model) goBack() *model {
	switch m.state {
	case stateFileList:
		m.state = stateMainMenu
	case statePasswordInput:
		m.state = stateFileList
		m.fileChoice = ""
		m.passwordInput = textinput.Model{}
		m.passwordInputError = false
	case stateKeyBindings:
		m.state = stateMainMenu
	case stateAddDbForm:
		m.state = stateMainMenu
	case stateDbView:
		m.state = stateFileList
		m.fileChoice = ""
		m.dbData = []table.Row{}
	case stateAddRecordForm:
		m.state = stateDbView
		m.dbTitleInput = textinput.Model{}
		m.dbPasswordInput = textinput.Model{}
		m.dbTitleInputError = false
		m.dbPasswordInputError = false
	case stateError:
		m.state = stateMainMenu
		m.errorMessage = ""
	}
	m.choice = ""
	return m
}

// Set error message
func (m *model) setError(message string) {
	m.state = stateError
	m.errorMessage = message
}

// Handle Enter in main menu
func (m *model) handleMainMenuEnter() (tea.Model, tea.Cmd) {
	i, ok := m.list.SelectedItem().(item)
	if !ok {
		return m, nil
	}

	m.choice = string(i)

	switch m.choice {
	case "Add db":
		m.titleInput = createTitleInput()
		m.passwordInput = createPasswordInput()
		m.titleInput.Focus()
		m.passwordInput.Blur()
		m.state = stateAddDbForm
	case "Open db":
		fileList, err := createFileList()
		if err != nil {
			m.setError(fmt.Sprintf("Failed to create file list: %v", err))
			return m, nil
		}
		m.fileList = fileList
		m.state = stateFileList
	case "Key bindings":
		m.state = stateKeyBindings
	}

	return m, nil
}

// Handle Enter in file list
func (m *model) handleFileListEnter() (tea.Model, tea.Cmd) {
	i, ok := m.fileList.SelectedItem().(item)
	if !ok {
		return m, nil
	}

	m.fileChoice = string(i)
	m.passwordInput = createPasswordInput()
	m.passwordInputError = false
	m.state = statePasswordInput

	return m, nil
}

// Handle Enter in password input
func (m *model) handlePasswordEnter() (tea.Model, tea.Cmd) {
	if m.passwordInput.Value() == "" {
		m.passwordInputError = true
		return m, nil
	}

	isOk, salt, err := IsFileHashValid(m.fileChoice, m.passwordInput.Value())
	if err != nil {
		m.setError(fmt.Sprintf("Failed to validate file hash: %v", err))
		return m, nil
	}
	if !isOk {
		m.passwordInputError = true
		m.errorMessage = "Invalid password"
		return m, nil
	}

	key := GenerateKey(m.passwordInput.Value(), salt)

	data, err := ReadPasswordFile(m.fileChoice, key)
	if err != nil {
		m.setError(fmt.Sprintf("Failed to read password file: %v", err))
		return m, nil
	}

	GlobalStore.Set("key", key)
	m.dbData = data
	m.updateTable()
	m.state = stateDbView
	m.activeButton = 0
	m.errorMessage = ""

	return m, nil
}

// Handle Enter in add DB form
func (m *model) handleAddDbFormEnter() (tea.Model, tea.Cmd) {
	titleEmpty := m.titleInput.Value() == ""
	passwordEmpty := m.passwordInput.Value() == ""

	m.titleInputError = titleEmpty
	m.passwordInputError = passwordEmpty

	if titleEmpty || passwordEmpty {
		return m, nil
	}

	config := ReadConfigFile()

	err := CreatePasswordFile(m.titleInput.Value(), config.DBsFolder, m.passwordInput.Value())
	if err != nil {
		m.setError(fmt.Sprintf("Failed to create password file: %v", err))
		return m, nil
	}

	m.state = stateMainMenu
	m.titleInput = textinput.Model{}
	m.passwordInput = textinput.Model{}
	m.titleInputError = false
	m.passwordInputError = false
	m.errorMessage = ""
	return m, nil
}

// Handle Enter in add record form
func (m *model) handleAddRecordEnter() (tea.Model, tea.Cmd) {
	titleEmpty := m.dbTitleInput.Value() == ""
	passwordEmpty := m.dbPasswordInput.Value() == ""

	m.dbTitleInputError = titleEmpty
	m.dbPasswordInputError = passwordEmpty

	if titleEmpty || passwordEmpty {
		return m, nil
	}

	keyInterface, exists := GlobalStore.Get("key")
	if !exists {
		m.setError("Key not found in store")
		return m, nil
	}

	key, ok := keyInterface.([]byte)
	if !ok {
		m.setError("Invalid key type in store")
		return m, nil
	}

	config := ReadConfigFile()

	err := AddToPasswordFile(config.DBsFolder, m.fileChoice, m.dbTitleInput.Value(), m.dbPasswordInput.Value(), key)
	if err != nil {
		m.setError(fmt.Sprintf("Failed to add record: %v", err))
		return m, nil
	}

	newRow := table.Row{m.dbTitleInput.Value(), m.dbPasswordInput.Value()}
	m.dbData = append(m.dbData, newRow)
	m.updateTable()

	m.state = stateDbView
	m.dbTitleInput = textinput.Model{}
	m.dbPasswordInput = textinput.Model{}
	m.dbTitleInputError = false
	m.dbPasswordInputError = false
	m.activeButton = 0
	m.errorMessage = ""

	return m, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// config := ReadConfigFile()

	if windowMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = windowMsg.Width
		m.height = windowMsg.Height
		m.list.SetWidth(m.width)
		if m.fileList.Width() > 0 {
			m.fileList.SetWidth(30)
		}
		return m, nil
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// Handle filtering for fileList
		if m.state == stateFileList && m.fileList.FilterState() != list.Unfiltered {
			var cmd tea.Cmd
			m.fileList, cmd = m.fileList.Update(msg)
			m.fileList.SetWidth(30)
			return m, cmd
		}

		// Global keys
		if model, cmd := m.handleGlobalKeys(keyMsg.String()); cmd != nil || m.quitting {
			return model, cmd
		}

		// Handle keys based on state
		switch m.state {
		case stateMainMenu:
			if keyMsg.String() == "enter" {
				return m.handleMainMenuEnter()
			}
		case stateFileList:
			if keyMsg.String() == "enter" {
				return m.handleFileListEnter()
			}
		case statePasswordInput:
			switch keyMsg.String() {
			case "esc":
				return m.resetToMainMenu(), nil
			case "enter":
				return m.handlePasswordEnter()
			}
		case stateAddDbForm:
			switch keyMsg.String() {
			case "esc":
				return m.resetToMainMenu(), nil
			case "enter":
				return m.handleAddDbFormEnter()
			case "tab":
				if m.titleInput.Focused() {
					m.titleInput.Blur()
					m.passwordInput.Focus()
				} else {
					m.passwordInput.Blur()
					m.titleInput.Focus()
				}
				return m, nil
			}
		case stateDbView:
			return m.handleDbViewKeys(keyMsg.String())
		case stateAddRecordForm:
			switch keyMsg.String() {
			case "esc":
				m.state = stateDbView
				m.dbTitleInput = textinput.Model{}
				m.dbPasswordInput = textinput.Model{}
				m.dbTitleInputError = false
				m.dbPasswordInputError = false
				return m, nil
			case "enter":
				return m.handleAddRecordEnter()
			case "tab":
				if m.dbTitleInput.Focused() {
					m.dbTitleInput.Blur()
					m.dbPasswordInput.Focus()
				} else {
					m.dbPasswordInput.Blur()
					m.dbTitleInput.Focus()
				}
				return m, nil
			}
		case stateKeyBindings:
			if keyMsg.String() == "enter" {
				m.state = stateMainMenu
				m.choice = ""
				return m, nil
			}
		}
	}

	// Update components
	var cmd tea.Cmd
	switch m.state {
	case stateMainMenu:
		m.list, cmd = m.list.Update(msg)
	case stateFileList:
		m.fileList, cmd = m.fileList.Update(msg)
		m.fileList.SetWidth(30)
	case statePasswordInput:
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	case stateAddDbForm:
		m.titleInput, cmd = m.titleInput.Update(msg)
		if cmd != nil {
			return m, cmd
		}
		m.passwordInput, cmd = m.passwordInput.Update(msg)
	case stateDbView:
		m.table, cmd = m.table.Update(msg)
	case stateAddRecordForm:
		m.dbTitleInput, cmd = m.dbTitleInput.Update(msg)
		if cmd != nil {
			return m, cmd
		}
		m.dbPasswordInput, cmd = m.dbPasswordInput.Update(msg)
	}

	return m, cmd
}

// Handle DbView keys
func (m *model) handleDbViewKeys(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "a":
		m.dbTitleInput = createDbTitleInput()
		m.dbPasswordInput = createDbPasswordInput()
		m.dbTitleInput.Focus()
		m.dbPasswordInput.Blur()
		m.state = stateAddRecordForm
		return m, nil
	case "d":
		if len(m.dbData) > 0 {
			selectedIndex := m.table.Cursor()
			if selectedIndex < len(m.dbData) {
				err := RemoveFromPasswordFile(ReadConfigFile().DBsFolder, m.fileChoice, selectedIndex)
				if err != nil {
					m.setError(fmt.Sprintf("Failed to remove record: %v", err))
					return m, nil
				}

				keyInterface, exists := GlobalStore.Get("key")
				if !exists {
					m.setError("Key not found")
					return m, nil
				}

				key, ok := keyInterface.([]byte)
				if !ok {
					m.setError("Invalid key type")
					return m, nil
				}

				data, err := ReadPasswordFile(m.fileChoice, key)
				if err != nil {
					m.setError(fmt.Sprintf("Failed to read file: %v", err))
					return m, nil
				}

				m.dbData = data
				m.updateTable()
				m.errorMessage = ""
			}
		}
		return m, nil
	case "left":
		m.activeButton--
		if m.activeButton < 0 {
			m.activeButton = 1
		}
		return m, nil
	case "right":
		m.activeButton++
		if m.activeButton > 1 {
			m.activeButton = 0
		}
		return m, nil
	case "enter":
		switch m.activeButton {
		case 0: // Add
			m.dbTitleInput = createDbTitleInput()
			m.dbPasswordInput = createDbPasswordInput()
			m.dbTitleInput.Focus()
			m.dbPasswordInput.Blur()
			m.state = stateAddRecordForm
		case 1: // Delete
			if len(m.dbData) > 0 {
				selectedIndex := m.table.Cursor()
				if selectedIndex < len(m.dbData) {
					err := RemoveFromPasswordFile(ReadConfigFile().DBsFolder, m.fileChoice, selectedIndex)
					if err != nil {
						m.setError(fmt.Sprintf("Failed to remove record: %v", err))
						return m, nil
					}

					keyInterface, exists := GlobalStore.Get("key")
					if !exists {
						m.setError("Key not found")
						return m, nil
					}

					key, ok := keyInterface.([]byte)
					if !ok {
						m.setError("Invalid key type")
						return m, nil
					}

					data, err := ReadPasswordFile(m.fileChoice, key)
					if err != nil {
						m.setError(fmt.Sprintf("Failed to read file: %v", err))
						return m, nil
					}

					m.dbData = data
					m.updateTable()
					m.errorMessage = ""
				}
			}
		}
		return m, nil
	}
	return m, nil
}

// Center content
func (m model) centerContent(content string) string {
	centeredStyle := centerStyle.
		Width(m.width).
		Height(m.height)
	return centeredStyle.Render(content)
}

// Render input with error
func (m model) renderInputWithError(input textinput.Model, hasError bool, label string) string {
	var inputStyle lipgloss.Style
	if hasError {
		inputStyle = errorInputFieldStyle
	} else if input.Focused() {
		inputStyle = focusedInputFieldStyle
	} else {
		inputStyle = inputFieldStyle
	}

	inputView := inputStyle.Render(input.View())
	styledLabel := labelStyle.Render(label + ":")
	inputRow := lipgloss.JoinHorizontal(lipgloss.Left, styledLabel, inputView)

	return inputRow
}

// Render buttons
func (m model) renderButtons() string {
	buttons := []string{"Add", "Delete"}
	var renderedButtons []string

	for i, button := range buttons {
		if i == m.activeButton {
			renderedButtons = append(renderedButtons, activeButtonStyle.Render(button))
		} else {
			renderedButtons = append(renderedButtons, buttonStyle.Render(button))
		}
	}

	buttonsRow := lipgloss.JoinHorizontal(lipgloss.Left, renderedButtons...)
	return buttonsRow
}

func (m model) View() string {
	var content string

	switch m.state {
	case stateMainMenu:
		listContent := listStyle.Render(m.list.View()) +
			"\n\n(Use ↑/↓ to navigate, Enter to select)"
		content = m.centerContent(listContent)

	case stateFileList:
		listContent := listStyle.Render(m.fileList.View()) +
			"\n\n(b: back to menu, m: main menu)"
		content = m.centerContent(listContent)

	case statePasswordInput:
		passwordField := m.renderInputWithError(m.passwordInput, m.passwordInputError, "Password")
		var errorContent string
		if m.errorMessage != "" {
			errorContent = "\n" + errorMessageStyle.Render(m.errorMessage)
		}
		formContent := fmt.Sprintf(
			"Selected file: %s\n\n%s%s\n\n(Enter to submit)",
			m.fileChoice,
			passwordField,
			errorContent,
		)
		styledForm := formStyle.Render(formContent) +
			"\n\n(Enter to submit, Esc to cancel)"
		content = m.centerContent(styledForm)

	case stateAddDbForm:
		titleField := m.renderInputWithError(m.titleInput, m.titleInputError, "Title")
		passwordField := m.renderInputWithError(m.passwordInput, m.passwordInputError, "Password")
		var errorContent string
		if m.errorMessage != "" {
			errorContent = "\n" + errorMessageStyle.Render(m.errorMessage)
		}

		formContent := fmt.Sprintf(
			"Add New Database\n\n%s\n\n%s%s\n\n(Tab to switch fields)",
			titleField,
			passwordField,
			errorContent,
		)
		styledForm := formStyle.Render(formContent)
		content = m.centerContent(styledForm)

	case stateDbView:
		tableTitle := tableTitleStyle.Render(m.fileChoice)
		tableContent := m.table.View()
		tableWithStyle := tableStyle.Render(tableContent)
		centeredTable := tableContainerStyle.Render(tableWithStyle)
		buttons := buttonsStyle.Render(m.renderButtons())

		var errorContent string
		if m.errorMessage != "" {
			errorContent = "\n" + errorMessageStyle.Render(m.errorMessage)
		}

		viewContent := fmt.Sprintf(
			"%s\n%s\n%s%s\n(↑/↓ navigate, ←/→ select, Enter execute)",
			tableTitle,
			centeredTable,
			buttons,
			errorContent,
		)
		content = m.centerContent(viewContent)

	case stateAddRecordForm:
		titleField := m.renderInputWithError(m.dbTitleInput, m.dbTitleInputError, "Title")
		passwordField := m.renderInputWithError(m.dbPasswordInput, m.dbPasswordInputError, "Password")
		var errorContent string
		if m.errorMessage != "" {
			errorContent = "\n" + errorMessageStyle.Render(m.errorMessage)
		}

		formContent := fmt.Sprintf(
			"Add New Record\n\n%s\n\n%s%s\n\n(Tab to switch fields)",
			titleField,
			passwordField,
			errorContent,
		)
		styledForm := formStyle.Render(formContent)
		content = m.centerContent(styledForm)

	case stateKeyBindings:
		bindingsContent := bindingsStyle.Render(getKeyBindingsText())
		content = m.centerContent(bindingsContent)

	case stateError:
		errorContent := errorMessageStyle.Render(fmt.Sprintf("Error: %s\n\nPress 'e' to return", m.errorMessage))
		content = m.centerContent(errorContent)
	}

	if m.quitting {
		content = m.centerContent("Goodbye!")
	}

	return content
}

// Key bindings text
func getKeyBindingsText() string {
	return `
Key Bindings:

Global:
  q, Ctrl+C    - Quit
  m            - Main menu
  b            - Go back
  Enter        - Confirm

Main Menu:
  ↑/↓          - Navigate
  Enter        - Select

File Selection:
  ↑/↓          - Navigate
  /            - Filter
  Enter        - Select

Database View:
  ↑/↓          - Navigate rows
  ←/→          - Select action
  Enter        - Execute
  a            - Add record
  d            - Delete record

Forms:
  Tab          - Switch fields
  Enter        - Submit
  Esc          - Cancel
`
}

func main() {
	m := initialModel()

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
