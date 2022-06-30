package controllers

import (
	"context"
	"net/http"
	"time"

	"concurrency-challenge/models"

	"github.com/gin-gonic/gin"
)

type Provider struct {
	fetcher
	refresher
	getter
}

func NewProvider(fetcher fetcher, refresher refresher, getter getter) Provider {
	return Provider{fetcher, refresher, getter}
}

type fetcher interface {
	Fetch(ctx context.Context, from, to int) error
}

type refresher interface {
	Refresh(context.Context) error
}

type getter interface {
	GetPokemons(context.Context) ([]models.Pokemon, error)
}

//FillCSV fill the local CSV to data form PokeAPI. By default will fetch from id 1 to 100 unless there are other information on the body
func (p Provider) FillCSV(c *gin.Context) {

	requestBody := struct {
		From int `json:"from"`
		To   int `json:"to"`
	}{1, 10}

	if err := c.Bind(&requestBody); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(c, 10 * time.Minute)
	defer cancel()

	if err := p.Fetch(ctxWithTimeout, requestBody.From, requestBody.To); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}

//RefreshCache feeds the csv data and save in redis
func (p Provider) RefreshCache(c *gin.Context) {
	if err := p.Refresh(c); err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.Status(http.StatusOK)
}

//GetPokemons return all pokemons in cache
func (p Provider) GetPokemons(c *gin.Context) {
	pokemons, err := p.getter.GetPokemons(c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, pokemons)
}
