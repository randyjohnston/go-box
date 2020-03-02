package main

import(
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

var accessToken string

var baseURL string = "https://api.box.com/2.0/"
var method string = "GET"
var URI string = "folders/102941292100/items"

var requestsPerSecond int = 15 // rps
var numWorkers int = 8
var accessTokenTimeToLive int = 55 // minutes
var numJobs int = 1000 // number of requests to process

func main() {
	jobs := make(chan int, numJobs)
	results := make(chan []interface{}, numJobs)

	accessToken = auth()

	for i := 1; i <= numWorkers; i++ {
		go worker(jobs, results)
	}

	go keepTokenRefreshed()

	for j := 1; j <= numJobs; j++ {
	 	jobs <- j
	}

	close(jobs)

	for k := 1; k <= numJobs; k++ {
		fmt.Println(<- results)
		fmt.Println("Processed job ", k, "of", numJobs)
	}	
}

func keepTokenRefreshed() {
	refreshCycle := time.Tick(time.Duration(accessTokenTimeToLive) * time.Minute)
	for {
		<- refreshCycle
		accessToken = auth()
	}
}

func worker(jobs  <-chan int, results chan<- []interface{}) {
	rate := time.Minute / time.Duration(float64(requestsPerSecond)/float64(numWorkers)*60)
	fmt.Println("Rate per worker:", rate)
	limiter := time.Tick(rate)
	for range jobs {
		<-limiter  
		results <- execute()
	}
}

func execute() []interface{} {
	client := &http.Client{}	
	req, err := http.NewRequest(method, baseURL + URI, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Add("Authorization", "Bearer " + accessToken)
	resp, err := client.Do(req)

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)

	entries := result["entries"].([]interface{})
	return entries
 }