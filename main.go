package main

import (
	"fmt"
	"github.com/gocolly/colly"
	"log"
	"strconv"
)

func getAveragePrice() float32 {
	var prices []float32

	c := colly.NewCollector()

	c.OnHTML("#priceList > li", func(element *colly.HTMLElement) {
		numericString := element.Text[1:]
		price, err := strconv.ParseFloat(numericString, 32)
		if err != nil {
			log.Fatalln(err)
		}
		price32 := float32(price)
		prices = append(prices, price32)
	})

	err := c.Visit("https://www.em6live.co.nz/")
	if err != nil {
		log.Fatalln(err)
	}

	var sum float32 = 0
	for _, p := range prices {
		sum = sum + p
	}

	return sum / float32(len(prices))
}

func main() {
	//ui, err := lorca.New("https://google.com", "", 480, 320)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//<-ui.Done()

	fmt.Printf("%.2f", getAveragePrice())
}
