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
	Line          int
}
type deck struct {
	Cards         []card
	Size          int
	f             *os.File
	NewCards      int
	FinishedCards map[string]card
	FileFullPath  string
}

func (d *deck) updateDeckFile() {
	fInfo, _ := d.f.Stat()
	sb := strings.Builder{}
	sb.Grow(int(fInfo.Size()))

	sb.WriteString(fmt.Sprintf("NewCards: %d\n", d.NewCards))

	d.f.Seek(0, 0)
	csvReader := csv.NewReader(d.f)
	csvReader.TrimLeadingSpace = true
	r, err := csvReader.Read() // Skip config row

	for r, err = csvReader.Read(); r != nil && err != io.EOF; r, err = csvReader.Read() {
		c, found := d.FinishedCards[r[0]]
		if found {
			sb.WriteString(fmt.Sprintf("\"%s\",\"%s\",%d,%s\n", c.Front, c.Back, c.DayMultiplier, c.DueDate.Format(time.DateOnly)))
		} else {
			r[0] = "\"" + r[0] + "\""
			r[1] = "\"" + r[1] + "\""
			sb.WriteString(strings.Join(r, ","))
			sb.WriteByte('\n')
		}
	}
	d.f.Close()
	os.Remove(d.FileFullPath)
	newFile, _ := os.Create(d.FileFullPath)
	newFile.WriteString(sb.String())
	newFile.Close()
}

func (d *deck) updateFinishedCards(c card) {
	d.FinishedCards[c.Front] = c
}

func (d *deck) changeCard(m model, correct bool) (model, tea.Cmd) {
	if correct {
		m.Solved++
		// Update good card
		m.Deck.Cards[m.CardIndex].DueDate = time.Now().AddDate(0, 0, m.Deck.Cards[m.CardIndex].DayMultiplier)
		m.Deck.Cards[m.CardIndex].DayMultiplier *= 2
		d.updateFinishedCards(m.Deck.Cards[m.CardIndex])
		l := len(m.Deck.Cards)
		if l == 1 {
			return m, tea.Quit
		}
		m.Deck.Cards[m.CardIndex] = m.Deck.Cards[l-1]
		m.Deck.Cards = m.Deck.Cards[:l-1]
	} else if m.CurrentCard.DayMultiplier > 2 {
		// Update bad card
		m.Deck.Cards[m.CardIndex].DayMultiplier /= 2
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
	csvReader.TrimLeadingSpace = true
	// csvReader.LazyQuotes = true
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

	d.NewCards = cardsPerDay
	var newCards []card

	// Parse cards
	for r, err = csvReader.Read(); r != nil && err != io.EOF; r, err = csvReader.Read() {
		line, _ := csvReader.FieldPos(0)
		rl := len(r)
		if rl < 2 {
			return errors.New("malformed deck: cards must have at least 2 fields separated by commas")
		}
		if rl == 3 {
			return errors.New("malformed deck: missing day multiplier / review date")
		}

		// If only 2 fields provided, it's a new card
		if rl == 2 {
			c := newCard(r[0], r[1], 2, time.Now(), line)
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
			continue
		}

		c := newCard(r[0], r[1], dayMult, dd, line)
		d.Cards = append(d.Cards, c)
	}

	// Choose random new cards to show on this session
	n := len(newCards)
	for n > 0 && cardsPerDay > 0 {
		i := rand.Intn(n)
		d.Cards = append(d.Cards, newCards[i])
		newCards[i] = newCards[n-1]
		newCards = newCards[:n-1]
		n--
		cardsPerDay--
	}
	d.Size = len(d.Cards)
	return nil
}

func newCard(f, b string, d int, dd time.Time, l int) card {
	return card{
		Front:         f,
		Back:          b,
		DayMultiplier: d,
		DueDate:       dd,
		Line:          l,
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
	d.FileFullPath = confDeck
	if err != nil {
		f, err = os.Open(path)
		d.FileFullPath = path
		if err != nil {
			f.Close()
			return nil, err
		}
	}

	err = d.loadCards(f)
	if err != nil {
		f.Close()
		return nil, err
	}

	d.f = f
	d.FinishedCards = make(map[string]card)
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
