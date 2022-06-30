package use_cases

import (
	"context"
	"strings"
	"sync"

	"concurrency-challenge/models"
)

const (
	numWorkers = 10
)

type reader interface {
	Read() <-chan models.IncomingPokemon
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

	stages := r.save(
		r.feed(
			r.read(ctx),
		),
	)

	return <-stages
}

func (r Refresher) read(ctx context.Context) (context.Context, <-chan models.IncomingPokemon) {
	return ctx, r.Read()
}

func (r Refresher) feed(ctx context.Context, queue <-chan models.IncomingPokemon) (context.Context, <-chan models.IncomingPokemon) {
	out := make(chan models.IncomingPokemon)

	go func() {
		defer close(out)
		// fan out pattern
		workers := make([]<-chan models.IncomingPokemon, numWorkers)
		for i := range workers {
			workers[i] = r.feedPokemonWithAbility(queue)
		}

		// fan in pattern
		for incomingPokemon := range fanIn(workers...) {
			if incomingPokemon.Error != nil {
				out <- models.IncomingPokemon{
					Pokemon: models.Pokemon{},
					Error:   incomingPokemon.Error,
				}
				return
			}

			out <- models.IncomingPokemon{
				Pokemon: incomingPokemon.Pokemon,
				Error:   nil,
			}
		}
	}()

	return ctx, out
}

func (r Refresher) save(ctx context.Context, queue <-chan models.IncomingPokemon) <-chan error {
	out := make(chan error)

	go func() {
		defer close(out)
		var batch []models.Pokemon
		const batchSize = 2
		for pokemonIncoming := range queue {

			if pokemonIncoming.Error != nil {
				out <- pokemonIncoming.Error
				return
			}

			if len(batch) < batchSize {
				batch = append(batch, pokemonIncoming.Pokemon)
			} else {
				saveErr := r.Save(ctx, batch)
				if saveErr != nil {
					out <- saveErr
					return
				}
				batch = []models.Pokemon{pokemonIncoming.Pokemon}
			}
		}
	}()

	return out
}

func (r Refresher) feedPokemonWithAbility(queue <-chan models.IncomingPokemon) <-chan models.IncomingPokemon {
	out := make(chan models.IncomingPokemon)
	go func() {
		defer close(out)
		for ip := range queue {

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
