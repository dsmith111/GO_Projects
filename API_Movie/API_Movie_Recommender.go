package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type TasteDive struct {
	Similar TDSimilar `json:"Similar"`
}

type TDSimilar struct {
	Info    []TDSSub `json:"Info"`
	Results []TDSSub `json:"Results"`
}

type TDSSub struct {
	Name string `json:"Name"`
	Type string `json:"Type"`
}

type Omdb struct {
	Ratings []Ratings
}

type Ratings struct {
	Source string
	Value  string
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

func getMoviesTasteDive(name string) (*TasteDive, error) {
	t := &TasteDive{}
	req, err := http.NewRequest("GET", "https://tastedive.com/api/similar", nil)
	if err != nil {
		err = fmt.Errorf("Failed to initialize request!\n%w\n", err)
		return nil, err
	}

	q := req.URL.Query()
	q.Add("q", name)
	q.Add("type", "movie")
	q.Add("limit", "20")
	req.URL.RawQuery = q.Encode()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = fmt.Errorf("Failed to query API!\n%w\n", err)
		return nil, err
	}

	defer resp.Body.Close()
	dc := json.NewDecoder(resp.Body)
	if err := dc.Decode(t); err != nil {
		err = fmt.Errorf("Failed to read JSON!\n%w\n", err)
		return nil, err
	}

	return t, nil

}

func extractMovieTitles(selection *TasteDive) []string {
	names := []string{}

	for _, results := range selection.Similar.Results {
		names = append(names, results.Name)
	}
	return names
}

func getRelatedTitles(movieNames []string) []string {
	history := []string{}

	for _, title := range movieNames {
		tasteDiveResult, err := getMoviesTasteDive(title)
		if err != nil {
			log.Fatalf("Failed to get movies from TD!")
		}

		extractedTitles := extractMovieTitles(tasteDiveResult)

		for _, movie := range extractedTitles {
			if !contains(history, movie) {
				history = append(history, movie)
			}
		}
	}
	return history
}

func getMovieData(name string) (*Omdb, error) {
	omd := &Omdb{}
	req, err := http.NewRequest("GET", "http://www.omdbapi.com/", nil)
	if err != nil {
		err = fmt.Errorf("Failed to initialize request!\n%w\n", err)
		return nil, err
	}

	q := req.URL.Query()
	q.Add("apikey", os.Getenv("OMDB_API_KEY"))
	q.Add("t", name)
	q.Add("r", "json")
	req.URL.RawQuery = q.Encode()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		err = fmt.Errorf("Failed to query API!\n%w\n", err)
		return nil, err
	}

	defer resp.Body.Close()
	dc := json.NewDecoder(resp.Body)
	if err := dc.Decode(omd); err != nil {
		err = fmt.Errorf("Failed to read JSON!\n%w\n", err)
		return nil, err
	}
	return omd, nil
}

func findRaterInOMDB(omdb *Omdb) string {
	for _, object := range omdb.Ratings {
		if object.Source == "Rotten Tomatoes" {
			return object.Value
		}
	}
	return ""
}

func getMovieRating(name string) string {
	movieData, err := getMovieData(name)
	if err != nil {
		fmt.Printf("Error:\t%v\n", err)
		println("Failed to get rating for", name)
		return ""
	}

	rating := findRaterInOMDB(movieData)
	return rating
}

func getSortedRecommendations(movies []string) [][]string {
	related := getRelatedTitles(movies)
	rec := map[string]string{}
	ratingList := [][]string{}
	type kv struct {
		Key   string
		Value int
	}
	var kvList []kv
	ch := make(chan kv)

	for index, movie := range related {
		go func(ch chan kv, index int, movie string) {
			rating := getMovieRating(movie)
			rec[movie] = rating
			ratingVal, err := strconv.Atoi(strings.Split(rating, "%")[0])
			if err != nil {
				println("Error converting rating!", movie, strings.Split(rating, "%"))
				log.Fatalf("Exit 1")
			}
			ch <- kv{movie, ratingVal}

		}(ch, index, movie)

	}
	for range related {
		tempKv := <-ch
		kvList = append(kvList, tempKv)
	}
	sort.Slice(kvList, func(i, j int) bool {
		return kvList[i].Value > kvList[j].Value
	})
	for _, pair := range kvList {
		ratingList = append(ratingList, []string{pair.Key, strconv.Itoa(pair.Value)})
	}

	return ratingList

}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Failed to load environment variabls!")
	}
	var names []string
	names = []string{
		"John wick",
	}
	recs := getSortedRecommendations(names)

	fmt.Printf("Recommendations:\n%+v\n", recs)

}
