package repositories

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"concurrency-challenge/models"
)

type LocalStorage struct{}

const filePath = "resources/pokemons.csv"

func (l LocalStorage) Write(pokemons []models.Pokemon) error {
	file, fErr := os.Create(filePath)
	defer file.Close()
	if fErr != nil {
		return fErr
	}

	w := csv.NewWriter(file)
	records := buildRecords(pokemons)
	if err := w.WriteAll(records); err != nil {
		return err
	}

	return nil
}

func buildRecords(pokemons []models.Pokemon) [][]string {
	headers := []string{"id", "name", "height", "weight", "flat_abilities"}
	records := [][]string{headers}
	for _, p := range pokemons {
		record := fmt.Sprintf("%d,%s,%d,%d,%s",
			p.ID,
			p.Name,
			p.Height,
			p.Weight,
			p.FlatAbilityURLs)
		records = append(records, strings.Split(record, ","))
	}

	return records
}

func parseCSVData(record []string) models.IncomingPokemon {

	id, err := strconv.Atoi(record[0])
	if err != nil {
		return models.IncomingPokemon{
			Pokemon: models.Pokemon{},
			Error:   err,
		}
	}

	height, err := strconv.Atoi(record[2])
	if err != nil {
		return models.IncomingPokemon{
			Pokemon: models.Pokemon{},
			Error:   err,
		}
	}

	weight, err := strconv.Atoi(record[3])
	if err != nil {
		return models.IncomingPokemon{
			Pokemon: models.Pokemon{},
			Error:   err,
		}
	}

	return models.IncomingPokemon{
		Pokemon: models.Pokemon{
			ID:              id,
			Name:            record[1],
			Height:          height,
			Weight:          weight,
			Abilities:       nil,
			FlatAbilityURLs: record[4],
			EffectEntries:   nil,
		},
		Error: err,
	}
}

func (l LocalStorage) Read() (<-chan models.IncomingPokemon, error) {
	file, fErr := os.Open(filePath)
	if fErr != nil {
		return nil, fErr
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)

	out := make(chan models.IncomingPokemon)
	go func() {
		reader := csv.NewReader(file)
		_, err := reader.Read()
		if err != nil {
			out <- models.IncomingPokemon{
				Pokemon: models.Pokemon{},
				Error:   err,
			}
			return
		}

		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				out <- models.IncomingPokemon{
					Pokemon: models.Pokemon{},
					Error:   err,
				}
				return
			}

			out <- parseCSVData(row)
		}
		close(out)
		wg.Done()
	}()

	go func() {
		wg.Wait()
		file.Close()
	}()

	return out, nil
}
