package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gocolly/colly"
	"github.com/zserge/lorca"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var html = `
<!DOCTYPE html>
<html>
    <head>
		<title>Power Monitor</title>
        <style>
            html,
            body {
            padding: 0;
            margin: 0;
            font-family: -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, Oxygen,
                Ubuntu, Cantarell, Fira Sans, Droid Sans, Helvetica Neue, sans-serif;
            }

            * {
            box-sizing: border-box;
            }

            .container {
                min-height: 100vh;
            }

            .value {
                height: 50vh;
                color: white;
                font-size: 3rem;
                width: 100%;
                display: flex;
                align-items: center;
                justify-content: center;
                background-color: black;
            }
        </style>
    </head>
    <body>
        <div class="container">
            <div id="priceBox" class="value">
                <p>
                    <b id="priceValue">0.0</b> c/kWh
                </p>
            </div>

            <div id="co2Box" class="value">
                <p>
                    <b id="co2Value">0.0</b> gCO2eq/kWh
                </p>
            </div>
        </div>
    </body>
</html>
`

func getAveragePrice() float64 {
	var prices []float64

	c := colly.NewCollector()

	c.OnHTML("#priceList > li", func(element *colly.HTMLElement) {
		numericString := element.Text[1:]
		price, err := strconv.ParseFloat(numericString, 64)
		if err != nil {
			log.Fatalln("Float parse error: ", err)
		}
		prices = append(prices, price)
	})

	err := c.Visit("https://www.em6live.co.nz/")
	if err != nil {
		log.Fatalln("Colly visit error: ", err)
	}

	var sum float64 = 0
	for _, p := range prices {
		sum = sum + p
	}

	return sum / float64(len(prices))
}

func getAverageCarbonOutput() float64 {
	currentTimestamp := time.Now().UnixNano() / 1e6
	path := "/v3/state"
	token := "kUp26@Zg4fv$9Pm" // found in their bundle.js

	signature := sha256.Sum256([]byte(token + path + strconv.FormatInt(currentTimestamp, 10)))

	httpClient := &http.Client{}

	req, err := http.NewRequest("GET", "https://api.electricitymap.org/v3/state", nil)
	if err != nil {
		log.Fatalln("HTTP request creation error: ", err)
	}

	req.Header.Add("Origin", "https://www.electricitymap.org")
	req.Header.Add("Referer", "https://www.electricitymap.org/")
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.88 Safari/537.36")
	req.Header.Add("x-request-timestamp", strconv.FormatInt(currentTimestamp, 10))
	req.Header.Add("x-signature", hex.EncodeToString(signature[:]))

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Fatalln("HTTP do request error: ", err)
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln("HTTP body read error: ", err)
	}

	var data interface{}
	jsonErr := json.Unmarshal(body, &data)
	if jsonErr != nil {
		log.Fatalln("JSON unmarshal error: ", jsonErr)
	}

	// hacky as hell
	countries := data.(map[string]interface{})["data"].(map[string]interface{})["countries"].(map[string]interface{})

	nzn := countries["NZ-NZN"].(map[string]interface{})["co2intensity"].(float64)
	nzs := countries["NZ-NZS"].(map[string]interface{})["co2intensity"].(float64)

	return (nzn + nzs) / 2
}

func getPriceColor(price float64) string {
	if price < 70 {
		return "green"
	} else if price >= 70 && price <= 150 {
		return "orange"
	} else {
		return "red"
	}
}

func getCo2Color(grams float64) string {
	if grams < 100 {
		return "green"
	} else if grams >= 100 && grams <= 250 {
		return "orange"
	} else {
		return "red"
	}
}

func setPrice(ui *lorca.UI, value float64) {
	js := `
	document.getElementById("priceBox").style.backgroundColor = "%s";
	document.getElementById("priceValue").innerHTML = "%s";
	`
	interpolatedJs := fmt.Sprintf(js, getPriceColor(value), fmt.Sprintf("$%.2f", value / 1000))

	(*ui).Eval(interpolatedJs)
}

func setCo2(ui *lorca.UI, value float64) {
	js := `
	document.getElementById("co2Box").style.backgroundColor = "%s";
	document.getElementById("co2Value").innerHTML = "%s";
	`
	interpolatedJs := fmt.Sprintf(js, getCo2Color(value), fmt.Sprintf("%.2f", value))

	(*ui).Eval(interpolatedJs)
}

func main() {
	println(lorca.LocateChrome())

	ui, err := lorca.New("", "", 480, 320)
	if err != nil {
		log.Fatalln("Lorca initialisation error:",err)
	}
	defer ui.Close()

	ui.SetBounds(lorca.Bounds{
		WindowState: lorca.WindowStateFullscreen,
	})

	ui.Load("data:text/html," + url.PathEscape(html))

	// Perform initial load
	setPrice(&ui, getAveragePrice())
	setCo2(&ui, getAverageCarbonOutput())

	go func() {
		for range time.Tick(time.Minute * 10) {
			setPrice(&ui, getAveragePrice())
			setCo2(&ui, getAverageCarbonOutput())
		}
	}()

	<-ui.Done()
}
