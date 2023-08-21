package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"math/rand"
	"os"
)

var (
	focusedModelStyle = lipgloss.NewStyle().
		Width(25).
		Height(10).
		Align(lipgloss.Center, lipgloss.Center).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("69"))
)

func handleArgs() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: termsrs [name of deck]\n")
		os.Exit(1)
	}
	arg := os.Args[1]
	if arg == "--h" || arg == "--help" || arg == "-help" || arg == "-h" {
		fmt.Print("Usage: termsrs [name of deck]\n\n\n")
		fmt.Println("termsrs first tries to open the decks in .config/termsrs without the .srs ending\nExample: 'termsrs" +
			" spanish_vocab' will try to open a spanish_vocab.srs file located in .config/termsrs/\n\nIf not found, it will treat it as a path and will try to open the file, which means" +
			" that you can also supply a valid path directly.\nExample: 'termsrs ~/Documents/latin_declinations.srs")
		os.Exit(0)
	}
	if arg == "--l" || arg == "--list" || arg == "-l" || arg == "-list" {
		err := showDecks()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}

type model struct {
	Deck         *deck
	CurrentCard  card
	CardIndex    int
	Solved       int
	ShowSolution bool
}

func main() {
	handleArgs()
	m := initialModel()
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
	}
	m.Deck.updateDeckFile()
	fmt.Println("Done!")
}

func initialModel() model {
	d, err := newDeck(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	if len(d.Cards) == 0 {
		fmt.Println("no cards to review in deck")
		os.Exit(0)
	}

	i := rand.Intn(len(d.Cards))
	return model{
		ShowSolution: false,
		Deck:         d,
		CardIndex:    i,
		// Consider removing this and simply indexing the cards by the card index
		CurrentCard: d.Cards[i],
		Solved:      0,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) View() string {
	// Might be worth using a string builder
	var s string
	var helper string
	var displayCard string

	if m.ShowSolution {
		displayCard = m.CurrentCard.Back
		s = "\nBack\n\n"
		helper = "\n\nPress Y if you got it correct, N if not\n\n"
	} else {
		displayCard = m.CurrentCard.Front
		s = "\nFront\n\n"
		helper = "\n\nPress spacebar to reveal the card\n\n"
	}
	s += lipgloss.JoinHorizontal(lipgloss.Top, focusedModelStyle.Render(fmt.Sprintf("%s", displayCard)))
	c := m.Deck.Size
	p := float32(m.Solved) / float32(c) * 100

	s += fmt.Sprintf("\n\nSOLVED: %d/%d  %.0f%%\n", m.Solved, c, p)
	s += helper
	return s
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.String()
		if k == "ctrl+c" || k == "q" {
			return m, tea.Quit
		}

		if m.ShowSolution {
			if k == "y" {
				m.ShowSolution = false
				return m.Deck.changeCard(m, true)
			} else if k == "n" {
				m.ShowSolution = false
				return m.Deck.changeCard(m, false)
			}
		} else if k == " " {
			m.ShowSolution = true
		}
	}
	return m, nil
}
