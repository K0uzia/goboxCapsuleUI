package main

import (
	"fmt"
	"os/exec"
	"strings"

	dimension "gobox/bin"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	cursor  int
	choices []string
	output  string
}

func main() {
	m := model{
		choices: []string{"MAJ.sh", "CLEAN.sh", "SHRED.sh", "TEST_HW.sh", "PRINT.sh"},
	}
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
	}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case "d":
			dimension.Run()
		case "down":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}
		case "enter":
			choice := m.choices[m.cursor]
			out, _ := exec.Command("bash", "core/bin/"+choice).CombinedOutput()
			m.output = string(out)
		}
	}
	return m, nil
}

func (m model) View() string {
	s := "Select script to run:\n\n"
	for i, choice := range m.choices {
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		s += fmt.Sprintf("%s %s\n", cursor, choice)
	}
	s += "\nPress q to quit.\n"
	if m.output != "" {
		s += "\n--- Output ---\n" + strings.TrimSpace(m.output) + "\n"
	}
	return s
}
