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
	titleStyle        = lipgloss.NewStyle().MarginLeft(2)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle         = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)
	bindingsStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)

	// Стили для центрирования
	listStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	// Стиль для центрирования контента
	centerStyle = lipgloss.NewStyle().
			Align(lipgloss.Center).
			AlignVertical(lipgloss.Center)

	// Стили для формы
	formStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	// Стили для таблицы - УПРОЩЕННЫЕ И ЦЕНТРИРОВАННЫЕ
	tableStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(1, 1)

	tableTitleStyle = lipgloss.NewStyle().
			Align(lipgloss.Center).
			Bold(true).
			MarginBottom(0)

	// Стили для центрирования всего блока таблицы
	tableContainerStyle = lipgloss.NewStyle().
				Align(lipgloss.Center)

	// Стили для центрирования кнопок
	buttonsStyle = lipgloss.NewStyle().
			Align(lipgloss.Center).
			MarginTop(0).
			MarginBottom(1)

	// Стили для полей ввода
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

	// Стили для кнопок
	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)

	activeButtonStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("170")).
				Padding(0, 1)

	// Константы для колонок таблицы
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
	activeButton         int // 0 - Add, 1 - Delete
	errorMessage         string
}

// Создание стилизованной таблицы
func createStyledTable(columns []table.Column, rows []table.Row) table.Model {
	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
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

// Обновление данных таблицы
func (m *model) updateTable() {
	m.table.SetRows(m.dbData)
}

// Инициализация модели с динамической высотой списка
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
	l.Title = "What do you want for dinner?"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(false)
	l.Styles.Title = titleStyle
	l.Styles.PaginationStyle = paginationStyle
	l.Styles.HelpStyle = helpStyle

	// Инициализация таблицы
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

// Создание списка файлов БД
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

	listHeight := len(files) + 10
	const defaultWidth = 30

	fileList := list.New(files, itemDelegate{}, defaultWidth, listHeight)
	fileList.Title = "Select a database file"
	fileList.SetShowStatusBar(true)
	fileList.SetFilteringEnabled(false)
	fileList.Styles.Title = titleStyle
	fileList.Styles.PaginationStyle = paginationStyle
	fileList.Styles.HelpStyle = helpStyle

	return fileList, nil
}

// Создание поля ввода пароля
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

// Создание поля ввода заголовка
func createTitleInput() textinput.Model {
	input := textinput.New()
	input.Placeholder = "Enter database title"
	input.Focus()
	input.CharLimit = 100
	input.Width = 30
	return input
}

// Создание поля ввода для заголовка записи в БД
func createDbTitleInput() textinput.Model {
	input := textinput.New()
	input.Placeholder = "Enter record title"
	input.Focus()
	input.CharLimit = 100
	input.Width = 30
	return input
}

// Создание поля ввода для пароля записи в БД
func createDbPasswordInput() textinput.Model {
	input := textinput.New()
	input.Placeholder = "Enter record password"
	input.Focus()
	input.CharLimit = 156
	input.Width = 30
	input.EchoMode = textinput.EchoNormal
	return input
}

// Обработка глобальных клавиш
func (m *model) handleGlobalKeys(keypress string) (tea.Model, tea.Cmd) {
	if m.state == stateAddDbForm || m.state == statePasswordInput || m.state == stateAddRecordForm {
		return nil, nil
	}

	switch keypress {
	case "q", "ctrl+c":
		m.quitting = true
		return *m, tea.Quit

	case "m":
		return m.resetToMainMenu(), nil

	case "b":
		return m.goBack(), nil

	case "e":
		if m.state == stateError {
			m.state = stateMainMenu
			m.errorMessage = ""
			return *m, nil
		}
	}

	return nil, nil
}

// Сброс к главному меню
func (m *model) resetToMainMenu() model {
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
	return *m
}

// Возврат назад
func (m *model) goBack() model {
	switch m.state {
	case stateFileList:
		m.state = stateMainMenu
		m.choice = ""
	case statePasswordInput:
		m.state = stateFileList
		m.fileChoice = ""
		m.passwordInput = textinput.Model{}
		m.passwordInputError = false
	case stateKeyBindings:
		m.state = stateMainMenu
		m.choice = ""
	case stateAddDbForm:
		m.state = stateMainMenu
		m.choice = ""
		m.titleInput = textinput.Model{}
		m.passwordInput = textinput.Model{}
		m.titleInputError = false
		m.passwordInputError = false
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
	return *m
}

// Установка сообщения об ошибке
func (m *model) setError(message string) {
	m.state = stateError
	m.errorMessage = message
}

// Обработка Enter в главном меню
func (m *model) handleMainMenuEnter() (tea.Model, tea.Cmd) {
	i, ok := m.list.SelectedItem().(item)
	if !ok {
		return *m, nil
	}

	m.choice = string(i)

	switch m.choice {
	case "Add db":
		m.titleInput = createTitleInput()
		m.passwordInput = createPasswordInput()
		m.titleInput.Focus()
		m.passwordInput.Blur()
		m.titleInputError = false
		m.passwordInputError = false
		m.state = stateAddDbForm

	case "Open db":
		fileList, err := createFileList()
		if err != nil {
			m.setError(fmt.Sprintf("Failed to create file list: %v", err))
			return *m, nil
		}
		m.fileList = fileList
		m.state = stateFileList

	case "Key bindings":
		m.state = stateKeyBindings
	}

	return *m, nil
}

// Обработка Enter в списке файлов
func (m *model) handleFileListEnter() (tea.Model, tea.Cmd) {
	i, ok := m.fileList.SelectedItem().(item)
	if !ok {
		return *m, nil
	}

	m.fileChoice = string(i)
	m.passwordInput = createPasswordInput()
	m.passwordInputError = false
	m.state = statePasswordInput

	return *m, nil
}

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

	return *m, nil
}

// Обработка Enter в форме добавления БД
func (m *model) handleAddDbFormEnter() (tea.Model, tea.Cmd) {
	titleEmpty := m.titleInput.Value() == ""
	passwordEmpty := m.passwordInput.Value() == ""

	m.titleInputError = titleEmpty
	m.passwordInputError = passwordEmpty

	// Если есть пустые поля, не продолжаем обработку
	if titleEmpty || passwordEmpty {
		return m, nil
	}

	// Все данные введены корректно, продолжаем
	config := ReadConfigFile()

	err := CreatePasswordFile(m.titleInput.Value(), config.DBsFolder, m.passwordInput.Value())
	if err != nil {
		m.setError(fmt.Sprintf("Failed to create password file: %v", err))
		return m, nil
	}

	// Возвращаемся в главное меню после добавления
	m.state = stateMainMenu
	m.choice = ""
	m.titleInput = textinput.Model{}
	m.passwordInput = textinput.Model{}
	m.titleInputError = false
	m.passwordInputError = false
	m.errorMessage = ""
	return *m, nil
}

// Обработка Enter в форме добавления записи
func (m *model) handleAddRecordEnter() (tea.Model, tea.Cmd) {
	titleEmpty := m.dbTitleInput.Value() == ""
	passwordEmpty := m.dbPasswordInput.Value() == ""

	m.dbTitleInputError = titleEmpty
	m.dbPasswordInputError = passwordEmpty

	// Если есть пустые поля, не продолжаем обработку
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
		m.setError(fmt.Sprintf("Failed to add record to password file: %v", err))
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

	return *m, nil
}

// Обработка Enter в биндах клавиш
func (m *model) handleBindingsEnter() (tea.Model, tea.Cmd) {
	m.state = stateMainMenu
	m.choice = ""
	return *m, nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	config := ReadConfigFile()
	if windowMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = windowMsg.Width
		m.height = windowMsg.Height
		m.list.SetWidth(m.width)
		if m.fileList.Width() > 0 {
			m.fileList.SetWidth(m.width)
		}
		return m, nil
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if model, cmd := m.handleGlobalKeys(keyMsg.String()); cmd != nil || m.quitting {
			return model, cmd
		}

		// Обработка клавиш в режиме просмотра БД
		if m.state == stateDbView {
			switch keyMsg.String() {
			case "a":
				// Добавить новую запись
				m.dbTitleInput = createDbTitleInput()
				m.dbPasswordInput = createDbPasswordInput()
				m.dbTitleInput.Focus()
				m.dbPasswordInput.Blur()
				m.dbTitleInputError = false
				m.dbPasswordInputError = false
				m.state = stateAddRecordForm
				return m, nil
			case "d":
				// Удалить выбранную запись
				if len(m.dbData) > 0 {
					selectedIndex := m.table.Cursor()
					if selectedIndex < len(m.dbData) {
						// Удаляем выбранную запись
						err := RemoveFromPasswordFile(config.DBsFolder, m.fileChoice, selectedIndex)
						if err != nil {
							m.setError(fmt.Sprintf("Failed to remove record: %v", err))
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

						data, err := ReadPasswordFile(m.fileChoice, key)
						if err != nil {
							m.setError(fmt.Sprintf("Failed to read password file: %v", err))
							return m, nil
						}

						m.dbData = data
						m.updateTable()
						m.errorMessage = ""
					}
				}
				return m, nil
			case "left":
				// Переключение между кнопками
				m.activeButton--
				if m.activeButton < 0 {
					m.activeButton = 1
				}
				return m, nil
			case "right":
				// Переключение между кнопками
				m.activeButton++
				if m.activeButton > 1 {
					m.activeButton = 0
				}
				return m, nil
			case "enter":
				// Действие по выбранной кнопке
				switch m.activeButton {
				case 0: // Add
					m.dbTitleInput = createDbTitleInput()
					m.dbPasswordInput = createDbPasswordInput()
					m.dbTitleInput.Focus()
					m.dbPasswordInput.Blur()
					m.dbTitleInputError = false
					m.dbPasswordInputError = false
					m.state = stateAddRecordForm
				case 1: // Delete
					if len(m.dbData) > 0 {
						selectedIndex := m.table.Cursor()
						if selectedIndex < len(m.dbData) {
							// Удаляем выбранную запись
							err := RemoveFromPasswordFile(config.DBsFolder, m.fileChoice, selectedIndex)
							if err != nil {
								m.setError(fmt.Sprintf("Failed to remove record: %v", err))
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

							data, err := ReadPasswordFile(m.fileChoice, key)
							if err != nil {
								m.setError(fmt.Sprintf("Failed to read password file: %v", err))
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
		}

		if m.state == stateAddRecordForm {
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
		}

		if m.state == stateAddDbForm {
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
		}

		if m.state == statePasswordInput {
			switch keyMsg.String() {
			case "esc":
				return m.resetToMainMenu(), nil

			case "enter":
				return m.handlePasswordEnter()
			}
		}

		if keyMsg.String() == "enter" {
			switch m.state {
			case stateMainMenu:
				return m.handleMainMenuEnter()
			case stateFileList:
				return m.handleFileListEnter()
			case statePasswordInput:
				return m.handlePasswordEnter()
			case stateKeyBindings:
				return m.handleBindingsEnter()
			}
		}
	}

	var cmd tea.Cmd
	switch m.state {
	case stateMainMenu:
		m.list, cmd = m.list.Update(msg)
	case stateFileList:
		m.fileList, cmd = m.fileList.Update(msg)
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

// Центрирование контента
func (m model) centerContent(content string) string {
	centeredStyle := centerStyle.
		Width(m.width).
		Height(m.height)

	return centeredStyle.Render(content)
}

// Рендеринг поля ввода с учетом ошибок
func (m model) renderInputWithError(input textinput.Model, hasError bool, label string) string {
	// Применяем стили к полю ввода в зависимости от состояния
	var inputStyle lipgloss.Style
	if hasError {
		inputStyle = errorInputFieldStyle
	} else if input.Focused() {
		inputStyle = focusedInputFieldStyle
	} else {
		inputStyle = inputFieldStyle
	}

	// Рендерим поле ввода
	inputView := inputStyle.Render(input.View())

	// Стилизуем label
	styledLabel := labelStyle.Render(label + ":")

	// Комбинируем label и input в одной строке
	inputRow := lipgloss.JoinHorizontal(lipgloss.Left, styledLabel, inputView)

	return inputRow
}

// Рендеринг кнопок - ЦЕНТРИРОВАННЫЙ
func (m model) renderButtons() string {
	buttons := []string{
		"Add",
		"Delete",
	}

	var renderedButtons []string
	for i, button := range buttons {
		if i == m.activeButton {
			renderedButtons = append(renderedButtons, activeButtonStyle.Render(button))
		} else {
			renderedButtons = append(renderedButtons, buttonStyle.Render(button))
		}
	}

	// Соединяем кнопки по горизонтали
	buttonsRow := lipgloss.JoinHorizontal(lipgloss.Left, renderedButtons...)
	return buttonsRow
}

func (m model) View() string {
	var content string

	switch m.state {
	case stateMainMenu:
		listContent := listStyle.Render(m.list.View()) +
			"\n\n" + "(Use ↑/↓ to navigate, Enter to select)"
		content = m.centerContent(listContent)

	case stateFileList:
		listContent := listStyle.Render(m.fileList.View()) +
			"\n\n" + "(b: back to menu, m: main menu)"
		content = m.centerContent(listContent)

	case statePasswordInput:
		passwordField := m.renderInputWithError(m.passwordInput, m.passwordInputError, "Password")
		var errorContent string
		if m.errorMessage != "" {
			errorContent = "\n" + errorMessageStyle.Render(m.errorMessage)
		}
		formContent := fmt.Sprintf(
			"Selected file: %s\n\n%s%s\n\n%s",
			m.fileChoice,
			passwordField,
			errorContent,
			"(press enter to submit)",
		)
		styledForm := formStyle.Render(formContent) +
			"\n\n" + "(Enter to submit, Esc to cancel)"
		content = m.centerContent(styledForm)

	case stateAddDbForm:
		titleField := m.renderInputWithError(m.titleInput, m.titleInputError, "Title")
		passwordField := m.renderInputWithError(m.passwordInput, m.passwordInputError, "Password")
		var errorContent string
		if m.errorMessage != "" {
			errorContent = "\n" + errorMessageStyle.Render(m.errorMessage)
		}

		formContent := fmt.Sprintf(
			"Add New Database\n\n%s\n\n%s%s\n\n%s",
			titleField,
			passwordField,
			errorContent,
			"(Tab to switch fields, Enter to submit, Esc to cancel)",
		)
		styledForm := formStyle.Render(formContent)
		content = m.centerContent(styledForm)

	case stateDbView:
		// Создаем заголовок таблицы с названием файла
		tableTitle := tableTitleStyle.Render(m.fileChoice)

		// Получаем контент таблицы
		tableContent := m.table.View()

		// Применяем стиль к таблице
		tableWithStyle := tableStyle.Render(tableContent)

		// Центрируем всю таблицу со стилем
		centeredTable := tableContainerStyle.Render(tableWithStyle)

		// Рендерим центрированные кнопки
		buttons := buttonsStyle.Render(m.renderButtons())

		// Добавляем сообщение об ошибке если есть
		var errorContent string
		if m.errorMessage != "" {
			errorContent = "\n" + errorMessageStyle.Render(m.errorMessage)
		}

		// Комбинируем все элементы
		viewContent := fmt.Sprintf(
			"%s\n%s\n%s%s\n%s",
			tableTitle,
			centeredTable,
			buttons,
			errorContent,
			"(↑/↓ to navigate table, ←/→ to select action, Enter to execute, a/d for quick actions, b to go back)",
		)

		// Центрируем весь блок
		content = m.centerContent(viewContent)

	case stateAddRecordForm:
		titleField := m.renderInputWithError(m.dbTitleInput, m.dbTitleInputError, "Title")
		passwordField := m.renderInputWithError(m.dbPasswordInput, m.dbPasswordInputError, "Password")
		var errorContent string
		if m.errorMessage != "" {
			errorContent = "\n" + errorMessageStyle.Render(m.errorMessage)
		}

		formContent := fmt.Sprintf(
			"Add New Record\n\n%s\n\n%s%s\n\n%s",
			titleField,
			passwordField,
			errorContent,
			"(Tab to switch fields, Enter to submit, Esc to cancel)",
		)
		styledForm := formStyle.Render(formContent)
		content = m.centerContent(styledForm)

	case stateKeyBindings:
		bindingsContent := bindingsStyle.Render(getKeyBindingsText())
		content = m.centerContent(bindingsContent)

	case stateError:
		errorContent := errorMessageStyle.Render(fmt.Sprintf("Error: %s\n\nPress 'e' to return to main menu", m.errorMessage))
		content = m.centerContent(errorContent)
	}

	if m.quitting {
		var quitContent string
		if m.choice != "" && m.fileChoice != "" {
		} else {
			// quitContent = quitTextStyle.Render("Not hungry? That's cool.")
		}
		content = m.centerContent(quitContent)
	}

	return content
}

// Текст с биндами клавиш
func getKeyBindingsText() string {
	return `
Key Bindings:

Global:
  q, Ctrl+C    - Quit the application
  m            - Return to main menu
  b            - Go back to previous screen
  Enter        - Confirm selection
  e            - Return to main menu from error screen

Main Menu:
  ↑/↓          - Navigate items
  Enter        - Select item

File Selection:
  ↑/↓          - Navigate files
  Enter        - Select file

Database View:
  ↑/↓          - Navigate table rows
  ←/→          - Select action (Add/Delete)
  Enter        - Execute selected action
  a            - Add new record (quick)
  d            - Delete selected record (quick)
  b            - Go back to file selection

Add Record Form:
  Tab          - Switch between fields
  Enter        - Submit form
  Esc          - Cancel and return to database view

Add Database Form:
  Tab          - Switch between fields
  Enter        - Submit form
  Esc          - Cancel and return to main menu

Password Input:
  Any chars    - Type password (hidden)
  Enter        - Submit password

Key Bindings Screen:
  Enter        - Return to main menu

Press 'b' or 'm' to return to main menu, or 'Enter' to go back
`
}

func main() {
	m := initialModel()

	if _, err := tea.NewProgram(m).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
