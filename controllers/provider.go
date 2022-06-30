package controllers

import (
	"context"
	"fmt"
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
// time for record: 2.21 sec
// concurrency pattern: generator
// source: http://www.golangpatterns.info/concurrency/generators
// improvement: from 2.7 sec per call to potential parallelism (2.21 sec per 500 calls)
func (p Provider) FillCSV(c *gin.Context) {

	requestBody := struct {
		From int `json:"from"`
		To   int `json:"to"`
	}{1, 10}

	if err := c.Bind(&requestBody); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	ctxWithTimeout, cancel := context.WithTimeout(c, 10*time.Minute)
	defer cancel()

	if err := p.Fetch(ctxWithTimeout, requestBody.From, requestBody.To); err != nil {
		c.Status(http.StatusInternalServerError)
		fmt.Printf("[FillCSV][error:%s]", err.Error())
		return
	}

	c.Status(http.StatusOK)
}

// RefreshCache feeds the csv data and save in redis
// time for 40 records: 18.19 seconds
// concurrency pattern: fan-in-fan-out
// improvement: with 3 workers, 40 records : 5.55 seconds
func (p Provider) RefreshCache(c *gin.Context) {
	if err := p.Refresh(c); err != nil {
		c.Status(http.StatusInternalServerError)
		fmt.Printf("[RefreshCache][error:%s]", err.Error())
		return
	}

	c.Status(http.StatusOK)
}

//GetPokemons return all pokemons in cache
func (p Provider) GetPokemons(c *gin.Context) {
	pokemons, err := p.getter.GetPokemons(c)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		fmt.Printf("[GetPokemons][error:%s]", err.Error())
		return
	}

	c.JSON(http.StatusOK, pokemons)
}
