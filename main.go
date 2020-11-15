package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
)

type Movie struct {
	IMDBRating       float64 `json:"imdb_rating"`
	Title            string  `json:"title"`
	MovieReleaseYEar int     `json:"movie_release_year"`
	Summary          string  `json:"summary"`
	Durating         string  `json:"duration"`
	Genre            string  `json:"genre"`
}

var wg sync.WaitGroup

type Doc struct {
	doc *goquery.Document
}

func Usage() {
	fmt.Println("./imdb_chart_fetcher <URL> <COUNT>")
}

func Trim(content string) string {
	return strings.TrimSpace(content)
}

func NewDocument(url string) *Doc {
	doc, err := goquery.NewDocument(url)
	if err != nil {
		log.Fatalf("[ERROR] charUrl document creation failed! %v", err)
		os.Exit(1)
	}
	return &Doc{
		doc: doc,
	}
}

func ParseURL(charUrl string) *url.URL {
	urlProps, err := url.Parse(charUrl)
	if err != nil {
		log.Fatalf("Error: URL parse failed! %v", err)
		os.Exit(1)
	}
	return urlProps
}

func MovieLinks(docs *Doc, url string) []string {
	var movieLinks []string
	urlProps := ParseURL(url)
	docs.doc.Find(".titleColumn a").Each(func(index int, item *goquery.Selection) {
		linkTag := item
		link, _ := linkTag.Attr("href")
		movieLinks = append(movieLinks, urlProps.Scheme+"://"+urlProps.Host+link)
	})
	return movieLinks
}

func Details(docs *Doc) *Movie {
	titleWithYear := Trim(docs.doc.Find("div .title_wrapper h1").Contents().Text())
	titleList := strings.Split(titleWithYear, "(")

	rating, err := strconv.ParseFloat(Trim(docs.doc.Find("div div strong span").Contents().Text()), 100)
	if err != nil {
		rating = float64(0)
	}

	summary := Trim(docs.doc.Find("div .summary_text").Contents().Text())

	duration := Trim(docs.doc.Find("div .subtext time").Contents().Text())

	var genreList []string
	genreTags := docs.doc.Find("div .subtext a")
	count := len(genreTags.Nodes) - 1
	genreTags.Each(func(index int, item *goquery.Selection) {
		linkTag := item
		if count != index {
			genreList = append(genreList, linkTag.Text())
		}
	})

	genre := strings.Join(genreList, ", ")

	title := Trim(titleList[0])
	year, err := strconv.Atoi(Trim(titleList[1][:len(titleList[1])-1]))
	if err != nil {
		year = 1900
	}
	return &Movie{
		Title:            title,
		MovieReleaseYEar: year,
		IMDBRating:       rating,
		Summary:          summary,
		Durating:         duration,
		Genre:            genre,
	}
}

func docRoutine(c chan *Doc, movieLink string) {
	defer wg.Done()
	doc := NewDocument(movieLink)
	c <- doc
}

func MovieDetails() {

	chartLink := os.Args[1]
	doc := NewDocument(chartLink)
	movieLinks := MovieLinks(doc, chartLink)

	itemsCount, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal("Error: Enter integer count.")
		os.Exit(1)
	}

	ch := make(chan *Doc, itemsCount)

	for index, movieLink := range movieLinks {
		if index+1 > itemsCount {
			break
		}
		wg.Add(1)
		go docRoutine(ch, movieLink)
	}

	wg.Wait()
	close(ch)

	var movies []Movie

	for doc := range ch {
		mov := Details(doc)
		movies = append(movies, *mov)
	}

	j, err := json.Marshal(movies)
	if err != nil {
		log.Fatal("Error converting to json!")
	}
	fmt.Println(string(j))
}

func main() {
	if len(os.Args) != 3 {
		Usage()
		os.Exit(1)
	}
	MovieDetails()
}
