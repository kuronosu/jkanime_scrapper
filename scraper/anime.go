package scraper

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/kuronosu/jkanime_scraper/utils"
	"golang.org/x/net/html"
)

type Link struct {
	Text string `json:"text"`
	Url  string `json:"url"`
}

type Related struct {
	Relation string `json:"relation"`
	Animes   []Link `json:"animes"`
}

func (related *Related) addLink(link Link) {
	if related.Animes == nil {
		related.Animes = make([]Link, 0)
	}
	related.Animes = append(related.Animes, link)
}

type Anime struct {
	Url               string            `json:"url"`
	Name              string            `json:"name"`
	Image             string            `json:"image"`
	Synopsis          string            `json:"synopsis"`
	Type              string            `json:"type"`
	Genres            []Link            `json:"genres"`
	Studios           []Link            `json:"studios"`
	Languages         string            `json:"languages"`
	Episodes          int               `json:"episodes"`
	Duration          string            `json:"duration"`
	Aired             string            `json:"aired"`
	Status            string            `json:"status"`
	AlternativeTitles map[string]string `json:"alternative_titles"`
	Likes             int               `json:"likes"`
	Related           []Related         `json:"related"`
	EpisodesList      []Episode         `json:"episodes_list"`
}

type safeAnimeContainer struct {
	mu     sync.Mutex
	animes map[string]Anime
}

func (container *safeAnimeContainer) AddAnime(url string, anime Anime) {
	container.mu.Lock()
	defer container.mu.Unlock()
	container.animes[url] = anime
}

func ScrapeAnimes(urls []string) map[string]Anime {
	c := colly.NewCollector(
		colly.Async(true),
	)
	container := safeAnimeContainer{
		animes: make(map[string]Anime),
	}

	c.OnError(func(r *colly.Response, err error) {
		log.Println(utils.ERROR(), r.Request.URL, err)
		log.Fatal()
	})

	c.OnResponse(func(r *colly.Response) {
		log.Println(utils.VISITED(), r.Request.URL)
	})

	c.OnHTML("section.contenido.spad>div.container", func(e *colly.HTMLElement) {
		episodesPageCount := len(strings.Split(strings.TrimSpace(e.ChildText(".anime__pagination")), "\n"))

		name := strings.TrimSpace(e.ChildText(".anime__details__title"))
		image := e.ChildAttr("div.anime__details__pic", "data-setbg")
		synopsis := strings.TrimSpace(e.ChildText("p[rel=sinopsis]"))
		typ := ""
		genres := make([]Link, 0)
		studios := make([]Link, 0)
		language := ""
		episodes := -1
		duration := ""
		aired := ""
		status := ""
		e.ForEach(".anime__details__widget li", func(i int, li *colly.HTMLElement) {
			if strings.Contains(li.Text, "Tipo:") {
				typ = getTextDetail(li, "Tipo:")
			} else if strings.Contains(li.Text, "Genero:") {
				genres = getLinkArr(li)
			} else if strings.Contains(li.Text, "Studios:") {
				studios = getLinkArr(li)
			} else if strings.Contains(li.Text, "Idiomas:") {
				language = getTextDetail(li, "Idiomas:")
			} else if strings.Contains(li.Text, "Episodios:") {
				episodes = getIntDetail(li, "Episodios:")
			} else if strings.Contains(li.Text, "Duracion:") {
				duration = getTextDetail(li, "Duracion:")
			} else if strings.Contains(li.Text, "Emitido:") {
				aired = getTextDetail(li, "Emitido:")
			} else if strings.Contains(li.Text, "Estado:") {
				status = getTextDetail(li, "Estado:")
			}
		})
		alternativeTitles := getAlternativeTitles(e.ChildText("div#c"), e.ChildTexts("div#c b.t"))
		likes, err := strconv.Atoi(e.ChildText("span.vot"))
		if err != nil {
			log.Println(utils.WARNING(), err)
		}
		related := assembleRelated(e.DOM.Find(".col-lg-6.col-md-6:nth-child(2)").Find("h5#aditional, a").Nodes)
		episodesAjaxUrls := make([]string, 0)
		for i := 1; i <= episodesPageCount; i++ {
			episodesAjaxUrls = append(episodesAjaxUrls,
				fmt.Sprintf("https://jkanime.net/ajax/pagination_episodes/%s/%d/", e.ChildAttr("#guardar-anime", "data-anime"), i),
			)
		}
		episodesList := fetchEpisodes(episodesAjaxUrls)

		container.AddAnime(e.Request.URL.String(), Anime{
			Url:               e.Request.URL.String(),
			Name:              name,
			Image:             image,
			Synopsis:          synopsis,
			Type:              typ,
			Genres:            genres,
			Studios:           studios,
			Languages:         language,
			Episodes:          episodes,
			Duration:          duration,
			Aired:             aired,
			Status:            status,
			AlternativeTitles: alternativeTitles,
			Likes:             likes,
			Related:           related,
			EpisodesList:      episodesList,
		})
	})

	for _, url := range urls {
		c.Visit(url)
	}
	c.Wait()
	return container.animes
}

func getLinkArr(li *colly.HTMLElement) []Link {
	elements := make([]Link, 0)
	li.ForEach("a", func(i int, a *colly.HTMLElement) {
		elements = append(elements, Link{
			Text: strings.TrimSpace(a.Text),
			Url:  strings.TrimSpace(a.Attr("href")),
		})
	})
	return elements
}

func getTextDetail(li *colly.HTMLElement, text string) string {
	return strings.TrimSpace(strings.Replace(li.Text, text, "", 1))
}

func getIntDetail(li *colly.HTMLElement, text string) int {
	tmp := strings.TrimSpace(strings.Replace(li.Text, text, "", 1))
	n, err := strconv.Atoi(tmp)
	if err != nil {
		log.Println(utils.WARNING(), err)
		return -1
	}
	return n
}

func splitAndTrim(s string) []string {
	tmp := strings.Split(s, "\n")
	tmp2 := make([]string, 0)
	for _, v := range tmp {
		tmp2 = append(tmp2, strings.TrimSpace(v))
	}
	return tmp2
}

func getAlternativeTitles(s string, sep []string) map[string]string {
	el := make(map[string]string)
	tmp := splitAndTrim(s)
	if len(tmp)%2 != 0 {
		log.Println(utils.WARNING(), "odd alternate titles")
		return el
	}
	for i := 0; i < len(tmp); i += 2 {
		if len(tmp[i]) == 0 || len(sep[i/2]) == 0 || len(tmp[i+1]) == 0 {
			continue
		}
		if tmp[i] == sep[i/2] {
			el[tmp[i]] = tmp[i+1]
		}
	}
	return el
}

func assembleRelated(nodes []*html.Node) []Related {
	related := make([]Related, 0)
	var tmp *Related = nil
	for _, node := range nodes {
		if node.Type == html.ElementNode && node.Data == "h5" {
			if tmp != nil {
				related = append(related, *tmp)
			}
			tmp = &Related{
				Relation: collectText(node),
				Animes:   make([]Link, 0),
			}
		} else if node.Type == html.ElementNode && node.Data == "a" {
			tmp.addLink(Link{
				Text: collectText(node),
				Url:  getAttribute(node.Attr, "href"),
			})
		}
	}
	if tmp != nil {
		related = append(related, *tmp)
	}
	return related
}

func getAttribute(attrs []html.Attribute, attr string) string {
	for _, _attr := range attrs {
		if _attr.Key == attr {
			return _attr.Val
		}
	}
	return ""
}

func collectText(node *html.Node) string {
	text := &bytes.Buffer{}
	_collectText(node, text)
	return text.String()
}

func _collectText(n *html.Node, buf *bytes.Buffer) {
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		_collectText(c, buf)
	}
}
