package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

type card struct {
	Front         string
	Back          string
	DayMultiplier int
	DueDate       time.Time
}
type deck struct {
	Cards []card
	Size  int
}

func (d *deck) changeCard(m model, correct bool) (model, tea.Cmd) {
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

func (d *deck) loadCards(f *os.File) error {
	csvReader := csv.NewReader(f)
	r, err := csvReader.Read()
	if err != nil {
		return err
	}

	// Parse file config
	conf := strings.Split(r[0], ":")
	conf[0] = strings.TrimSpace(conf[0])
	if len(conf) != 2 || conf[0] != "NewCards" {
		return errors.New("missing first row config in deck: deck's first row must contain \"NewCards: N\", where N is the number of new cards to study per day")
	}
	cardsPerDay, err := strconv.Atoi(strings.TrimSpace(conf[1]))
	if err != nil || cardsPerDay < 0 {
		return errors.New("invalid NewCards day specified on deck config")
	}

	var newCards []card

	// Parse cards
	for r, err = csvReader.Read(); r != nil && err != io.EOF; r, err = csvReader.Read() {
		rl := len(r)
		if rl < 2 {
			return errors.New("malformed deck: cards must have at least 2 fields separated by commas")
		}
		if rl == 3 {
			return errors.New("malformed deck: missing day multiplier / review date")
		}

		// If only 2 fields provided, it's a new card
		if rl == 2 {
			c := newCard(r[0], r[1], 2, time.Now())
			newCards = append(newCards, c)
			continue
		}

		// Parse rest of cards
		dayMult, err := strconv.Atoi(strings.TrimSpace(r[2]))
		if err != nil {
			return errors.New("malfomed deck: invalid day multiplier")
		}

		dd, err := time.Parse(time.DateOnly, strings.TrimSpace(r[3]))
		if err != nil {
			return errors.New("malformed deck: unsupported date format")
		}

		// Check if the card is due, it not, skip
		if dd.Compare(time.Now()) > 0 {
			fmt.Printf("CARDS %s, %s IGNORED BECAUSE NOT DUE YET\n", r[0], r[1])
			continue
		}

		c := card{
			Front:         r[0],
			Back:          r[1],
			DayMultiplier: dayMult,
			DueDate:       dd,
		}
		fmt.Printf("CARDS %s, %s ADDED BECAUSE IT IS DUE\n", r[0], r[1])
		d.Cards = append(d.Cards, c)
	}

	// Choose random new cards to show on this session
	n := len(newCards)
	for n > 0 && cardsPerDay > 0 {
		i := rand.Intn(n)
		d.Cards = append(d.Cards, newCards[i])
		fmt.Printf("CARDS %s, %s ADDED RANDOMLY BECAUSE IT'S NEW\n", newCards[i].Front, newCards[i].Back)
		newCards[i] = newCards[n-1]
		newCards = newCards[:n-1]
		n--
		cardsPerDay--
	}

	d.Size = len(d.Cards)
	return nil
}

func newCard(f, b string, d int, dd time.Time) card {
	return card{
		Front:         f,
		Back:          b,
		DayMultiplier: d,
		DueDate:       dd,
	}
}

func newDeck(path string) (*deck, error) {
	// We first look for a deck in .config/termsrs
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	confDeck := fmt.Sprintf("%s/.config/termsrs/%s.srs", homeDir, path)

	var f *os.File
	var d deck
	f, err = os.Open(confDeck)
	if err != nil {
		f, err = os.Open(path)
		if err != nil {
			return nil, err
		}
	}
	defer f.Close()

	err = d.loadCards(f)
	if err != nil {
		return nil, err
	}

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
