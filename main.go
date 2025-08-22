package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
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
)

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
)

type model struct {
	state              state
	list               list.Model
	fileList           list.Model
	passwordInput      textinput.Model
	titleInput         textinput.Model
	choice             string
	fileChoice         string
	quitting           bool
	width              int
	height             int
	titleInputError    bool
	passwordInputError bool
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

	return model{
		state:  stateMainMenu,
		list:   l,
		width:  80,
		height: 24,
	}
}

// Создание списка файлов БД с красными placeholder'ами при ошибках
func createFileList() list.Model {
	var files []list.Item
	config := ReadConfigFile()

	data, err := ReadDBsFolder(config.DBsFolder)
	if err != nil {
		fmt.Print(err)
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

	return fileList
}

// Создание поля ввода пароля с возможностью установки красного placeholder'а
func createPasswordInputWithError(errorState bool) textinput.Model {
	input := textinput.New()
	if errorState {
		input.Placeholder = "Enter password" // Оставляем обычный placeholder, стилизуем через границу
	} else {
		input.Placeholder = "Enter password"
	}
	input.Focus()
	input.CharLimit = 156
	input.Width = 30
	input.EchoMode = textinput.EchoPassword
	input.EchoCharacter = '*'
	return input
}

// Создание поля ввода заголовка с возможностью установки красного placeholder'а
func createTitleInputWithError(errorState bool) textinput.Model {
	input := textinput.New()
	if errorState {
		input.Placeholder = "Enter database title" // Оставляем обычный placeholder, стилизуем через границу
	} else {
		input.Placeholder = "Enter database title"
	}
	input.Focus()
	input.CharLimit = 100
	input.Width = 30
	return input
}

// Обработка глобальных клавиш
func (m *model) handleGlobalKeys(keypress string) (tea.Model, tea.Cmd) {
	if m.state == stateAddDbForm || m.state == statePasswordInput {
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
	m.titleInputError = false
	m.passwordInputError = false
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
	}
	return *m
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
		m.titleInput = createTitleInputWithError(false)
		m.passwordInput = createPasswordInputWithError(false)
		m.titleInput.Focus()
		m.passwordInput.Blur()
		m.titleInputError = false
		m.passwordInputError = false
		m.state = stateAddDbForm

	case "Open db":
		m.fileList = createFileList()
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
	m.passwordInput = createPasswordInputWithError(false)
	m.passwordInputError = false
	m.state = statePasswordInput

	return *m, nil
}

// Обработка Enter при вводе пароля
func (m *model) handlePasswordEnter() (tea.Model, tea.Cmd) {
	if m.passwordInput.Value() == "" {
		m.passwordInputError = true
		// Пересоздаем input с красной границей
		m.passwordInput = createPasswordInputWithError(true)
		if m.state == stateAddDbForm {
			m.passwordInput.Focus()
		}
		return m, nil
	}

	// Здесь можно обработать введенный пароль
	fmt.Printf("File: %s, Password: %s\n", m.fileChoice, m.passwordInput.Value())
	m.quitting = true
	return *m, tea.Quit
}

// Обработка Enter в форме добавления БД
func (m *model) handleAddDbFormEnter() (tea.Model, tea.Cmd) {
	titleEmpty := m.titleInput.Value() == ""
	passwordEmpty := m.passwordInput.Value() == ""

	m.titleInputError = titleEmpty
	m.passwordInputError = passwordEmpty

	// Если есть пустые поля, пересоздаем inputs с красными границами
	if titleEmpty || passwordEmpty {
		if titleEmpty {
			m.titleInput = createTitleInputWithError(true)
			m.titleInput.Blur()
		}
		if passwordEmpty {
			m.passwordInput = createPasswordInputWithError(true)
			m.passwordInput.Blur()
		}
		// Устанавливаем фокус на первое пустое поле
		if titleEmpty {
			m.titleInput.Focus()
		} else {
			m.passwordInput.Focus()
		}
		return m, nil
	}

	// Все данные введены корректно, продолжаем
	config := ReadConfigFile()

	err := CreatePasswordFile(m.titleInput.Value(), config.DBsFolder, m.passwordInput.Value())
	if err != nil {
		// Можно добавить обработку ошибки создания файла здесь
		return m, nil
	}

	// Возвращаемся в главное меню после добавления
	m.state = stateMainMenu
	m.choice = ""
	m.titleInput = textinput.Model{}
	m.passwordInput = textinput.Model{}
	m.titleInputError = false
	m.passwordInputError = false
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
		formContent := fmt.Sprintf(
			"Selected file: %s\n\n%s\n\n%s",
			m.fileChoice,
			passwordField,
			"(press enter to submit)",
		)
		styledForm := formStyle.Render(formContent) +
			"\n\n" + "(Enter to submit, Esc to cancel)"
		content = m.centerContent(styledForm)

	case stateAddDbForm:
		titleField := m.renderInputWithError(m.titleInput, m.titleInputError, "Title")
		passwordField := m.renderInputWithError(m.passwordInput, m.passwordInputError, "Password")

		formContent := fmt.Sprintf(
			"Add New Database\n\n%s\n\n%s\n\n%s",
			titleField,
			passwordField,
			"(Tab to switch fields, Enter to submit, Esc to cancel)",
		)
		styledForm := formStyle.Render(formContent)
		content = m.centerContent(styledForm)

	case stateKeyBindings:
		bindingsContent := bindingsStyle.Render(getKeyBindingsText())
		content = m.centerContent(bindingsContent)
	}

	if m.quitting {
		var quitContent string
		if m.choice != "" && m.fileChoice != "" {
			quitContent = quitTextStyle.Render(fmt.Sprintf("Selected: %s -> %s", m.choice, m.fileChoice))
		} else {
			quitContent = quitTextStyle.Render("Not hungry? That's cool.")
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

Main Menu:
  ↑/↓          - Navigate items
  Enter        - Select item

File Selection:
  ↑/↓          - Navigate files
  Enter        - Select file

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
