package scraper

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"sort"

	"github.com/kuronosu/jkanime_scraper/utils"
)

type Episode struct {
	Number string `json:"number"`
	Title  string `json:"title"`
	Image  string `json:"image"` // https://cdn.jkanime.net/assets/images/animes/video/image_thumb/{Image}
}

func fetchEpisodes(urls []string) []Episode {
	ch := make(chan []Episode, len(urls))
	episodes := []Episode{}

	for _, url := range urls {
		go func(url string) {
			ch <- fetchEpisodesPage(url)
		}(url)
	}

	c := 0
	for {
		r := <-ch
		c++
		episodes = append(episodes, r...)
		if c == len(urls) {
			sort.Slice(episodes, func(i, j int) bool {
				return utils.ConvertToInt(episodes[i].Number, 0) < utils.ConvertToInt(episodes[j].Number, 0)
			})
			return episodes
		}
	}
}

func fetchEpisodesPage(url string) []Episode {
	response, err := http.Get(url)
	if err != nil {
		log.Println(utils.WARNING(), url, err)
		return []Episode{}
	}
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	var episodes []Episode
	json.Unmarshal(responseData, &episodes)
	return episodes
}
