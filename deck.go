package main

import (
	"bufio"
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"math/rand"
	"os"
	"strings"
)

type Card struct {
	Front string
	Back  string
}
type Deck struct {
	Cards []Card
	Size  int
}

func (d *Deck) ChangeCard(m Model, correct bool) (Model, tea.Cmd) {
	if correct {
		m.Solved++
		l := len(m.Deck.Cards)
		if l == 1 {
			return m, tea.Quit
		}
		m.Deck.Cards[m.CardIndex] = m.Deck.Cards[l-1]
		m.Deck.Cards = m.Deck.Cards[:l-1]
	}
	l := len(m.Deck.Cards)

	i := rand.Intn(l)

	for i == m.CardIndex && l != 1 {
		i = rand.Intn(l)
	}

	m.CardIndex = i
	m.CurrentCard = m.Deck.Cards[i]
	return m, nil
}

func LoadDeck(path string) (*Deck, error) {
	// We first look for a deck in .config/termsrs
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	confDeck := fmt.Sprintf("%s/.config/termsrs/%s.srs", homeDir, path)

	var f *os.File
	var d Deck
	f, err = os.Open(confDeck)
	if err != nil {
		f, err = os.Open(path)
		if err != nil {
			return nil, err
		}
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		s := strings.Split(sc.Text(), ",")
		if len(s) != 2 {
			return nil, errors.New("incorrect card in deck")
		}
		c := Card{
			Front: strings.TrimSpace(s[0]),
			Back:  strings.TrimSpace(s[1]),
		}
		d.Cards = append(d.Cards, c)
	}
	d.Size = len(d.Cards)

	return &d, nil
}

func showDecks() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir, err := os.ReadDir(homeDir + "/.config/termsrs")
	if err != nil {
		return err
	}
	fmt.Println("List of decks:")
	for _, f := range dir {
		if len(f.Name()) > 5 && f.Name()[len(f.Name())-4:] == ".srs" {
			fmt.Println(f.Name()[:len(f.Name())-4])
		}
	}
	return nil
}
