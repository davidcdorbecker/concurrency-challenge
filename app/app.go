package app

import "concurrency-challenge/router"

func Start() {
	r := router.Init()
	if err := r.Run(":8080"); err != nil {
		panic(err)
	}
}
