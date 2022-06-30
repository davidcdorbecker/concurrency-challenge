package use_cases

import (
	"context"
	"strings"
	"sync"

	"concurrency-challenge/models"
)

const (
	numWorkers = 3
)

type reader interface {
	Read() (<-chan models.IncomingPokemon, error)
}

type saver interface {
	Save(context.Context, []models.Pokemon) error
}

type fetcher interface {
	FetchAbility(string) (models.Ability, error)
}

type Refresher struct {
	reader
	saver
	fetcher
}

func NewRefresher(reader reader, saver saver, fetcher fetcher) Refresher {
	return Refresher{reader, saver, fetcher}
}

func (r Refresher) Refresh(ctx context.Context) error {
	pokemonIncoming, err := r.Read()
	if err != nil {
		return err
	}

	var pokemons []models.Pokemon

	// fan out pattern
	workers := make([]<-chan models.IncomingPokemon, numWorkers)
	for i := range workers {
		workers[i] = r.feedPokemonWithAbility(pokemonIncoming)
	}

	// fan in pattern
	for incomingPokemon := range fanIn(workers...) {
		if incomingPokemon.Error != nil {
			return incomingPokemon.Error
		}

		pokemons = append(pokemons, incomingPokemon.Pokemon)
	}

	if err := r.Save(ctx, pokemons); err != nil {
		return err
	}

	return nil
}

func (r Refresher) feedPokemonWithAbility(incomingPokemon <-chan models.IncomingPokemon) <-chan models.IncomingPokemon {
	out := make(chan models.IncomingPokemon)
	go func() {
		for ip := range incomingPokemon {

			if ip.Error != nil {
				out <- ip
				return
			}

			urls := strings.Split(ip.Pokemon.FlatAbilityURLs, "|")
			var abilities []string
			for _, url := range urls {
				ability, err := r.FetchAbility(url)
				if err != nil {
					out <- models.IncomingPokemon{
						Pokemon: models.Pokemon{},
						Error:   err,
					}
					return
				}

				for _, ee := range ability.EffectEntries {
					abilities = append(abilities, ee.Effect)
				}
			}

			ip.Pokemon.EffectEntries = abilities
			out <- models.IncomingPokemon{
				Pokemon: ip.Pokemon,
				Error:   nil,
			}
		}

		close(out)
	}()

	return out
}

func fanIn(chans ...<-chan models.IncomingPokemon) <-chan models.IncomingPokemon {
	out := make(chan models.IncomingPokemon)
	wg := &sync.WaitGroup{}
	wg.Add(len(chans))

	for _, c := range chans {
		go func(incomingPokemon <-chan models.IncomingPokemon) {
			for ip := range incomingPokemon {
				out <- ip
			}
			wg.Done()
		}(c)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
