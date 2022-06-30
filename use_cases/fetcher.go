package use_cases

import (
	"context"
	"strings"
	"sync"

	"concurrency-challenge/models"
)

type api interface {
	FetchPokemon(id int) (models.Pokemon, error)
}

type writer interface {
	Write(pokemons []models.Pokemon) error
}

type Fetcher struct {
	api     api
	storage writer
}

func NewFetcher(api api, storage writer) Fetcher {
	return Fetcher{api, storage}
}

type errorSignal struct {
	hasError bool
	error    error
}

func (f Fetcher) Fetch(ctx context.Context, from, to int) error {
	var pokemons []models.Pokemon
	errSignal := make(chan errorSignal)

	g := generator(from, to, f.api, errSignal)

	for {
		select {
		case pokemonIncoming := <-g:
			pokemons = append(pokemons, pokemonIncoming)
		case signal := <-errSignal:
			if signal.hasError {
				return signal.error
			}
			return f.storage.Write(pokemons)
		case <-ctx.Done():
			return ctx.Err()
		}
	}

}

// source: http://www.golangpatterns.info/concurrency/generators
// improve: from 2.7 sec per call to potential parallelism (2.21 sec per 500 calls)
func generator(from, to int, fn api, errSignal chan errorSignal) <-chan models.Pokemon {
	ch := make(chan models.Pokemon)
	wg := sync.WaitGroup{}

	for id := from; id <= to; id++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			pokemon, err := fn.FetchPokemon(id)
			if err != nil {
				errSignal <- errorSignal{
					hasError: true,
					error:    err,
				}
				return
			}

			var flatAbilities []string
			for _, t := range pokemon.Abilities {
				flatAbilities = append(flatAbilities, t.Ability.URL)
			}
			pokemon.FlatAbilityURLs = strings.Join(flatAbilities, "|")
			ch <- pokemon
		}(id)
	}

	go func() {
		wg.Wait()
		close(ch)

		errSignal <- errorSignal{
			hasError: false,
			error:    nil,
		}
	}()

	return ch
}
