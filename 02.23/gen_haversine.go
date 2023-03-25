package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math/rand"
)

const name = "data_10000000_flex.json"
const count = 10000000

type Item struct {
	X0 float32 `json:"x0"`
	Y0 float32 `json:"y0"`
	X1 float32 `json:"x1"`
	Y1 float32 `json:"y1"`
}

type Result struct {
	Pairs []Item `json:"pairs"`
}

func main() {
	items := make([]Item, count)

	for i := 0; i < count; i++ {
		items[i].X0 = rand.Float32()
		items[i].Y0 = rand.Float32()
		items[i].X1 = rand.Float32()
		items[i].Y1 = rand.Float32()
	}

	b, err := json.Marshal(Result{items})
	if err != nil {
		log.Fatal(err)
	}

	if err := ioutil.WriteFile(name, b, 0644); err != nil {
		log.Fatal(err)
	}
}
