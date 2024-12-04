package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

type ProductData struct {
	Title       string   `json:"title"`
	Price       string   `json:"price"`
	Colors      string   `json:"colors"`
	Sizes       []string `json:"sizes"`
	Article     string   `json:"article"`
	Description string   `json:"description"`
	Composition string   `json:"composition"`
	Care        string   `json:"care"`
	Images      []string `json:"images"`
}

func main() {
	var wg sync.WaitGroup
	productURLs := make(chan string, 100)
	productDataChan := make(chan ProductData, 100)
	products := []ProductData{}

	// Запускаем горутины для парсинга товаров
	for i := 0; i < 10; i++ { // Уменьшаем количество горутин до 10
		wg.Add(1)
		go func() {
			defer wg.Done()
			for productURL := range productURLs {
				fmt.Println("Parsing product:", productURL)
				productData, err := parseProductPage("https://loverepublic.ru" + productURL)
				if err != nil {
					log.Printf("error parsing product %s: %v\n", productURL, err)
					continue
				}
				productDataChan <- productData
			}
		}()
	}

	// Запускаем горутину для записи данных в JSON
	go func() {
		ticker := time.NewTicker(1 * time.Minute) // Периодическая запись каждую минуту
		defer ticker.Stop()

		for {
			select {
			case productData, ok := <-productDataChan:
				if !ok {
					// Канал закрыт, завершаем работу
					saveProducts(products)
					return
				}
				products = append(products, productData)
			case <-ticker.C:
				// Периодическая запись данных в файл
				saveProducts(products)
			}
		}
	}()

	// Перебираем все страницы
	for pageNum := 1; pageNum <= 143; pageNum++ {
		url := fmt.Sprintf("https://loverepublic.ru/catalog/odezhda/?page=%d", pageNum)
		fmt.Println("Parsing page:", url)
		doc, err := fetchDocumentWithRetry(url, 3, 2*time.Second) // Добавляем механизм повторных попыток
		if err != nil {
			log.Printf("error parsing page %d: %v\n", pageNum, err)
			continue
		}

		doc.Find(".catalog-item-link").Each(func(i int, s *goquery.Selection) {
			productURL, exists := s.Attr("href")
			if exists {
				productURLs <- productURL
			}
		})
	}

	close(productURLs)
	wg.Wait()
	close(productDataChan)

	// Ожидание сигнала остановки
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		fmt.Println("Received interrupt signal, saving data and exiting...")
		saveProducts(products)
		os.Exit(0)
	}()
}

func saveProducts(products []ProductData) {
	filePath := "loverepublic_products.json"
	fmt.Println("Saving JSON file to:", filePath)
	file, err := os.Create(filePath)
	if err != nil {
		log.Fatalf("error creating JSON file: %v\n", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(products); err != nil {
		log.Fatalf("error encoding JSON: %v\n", err)
	}

	log.Println("JSON file saved successfully at", filePath)
}

func parseProductPage(url string) (ProductData, error) {
	var html string
	var err error

	// Повторные попытки для обработки ошибок соединения
	for attempt := 0; attempt < 3; attempt++ {
		html, err = fetchHTMLWithChromedp(url)
		if err == nil {
			break
		}
		log.Printf("Attempt %d failed: %v. Retrying...", attempt+1, err)
		time.Sleep(2 * time.Second) // Небольшая задержка перед повторной попыткой
	}

	if err != nil {
		return ProductData{}, fmt.Errorf("error running chromedp after 3 attempts: %v", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ProductData{}, fmt.Errorf("error parsing product page HTML: %v", err)
	}

	title := removeDigits(doc.Find(".catalog-element__title").Text())
	price := strings.ReplaceAll(doc.Find(".item-prices__price").Text(), " ", "")
	price = strings.ReplaceAll(price, "\u00A0", "")
	colors := strings.Join(doc.Find(".catalog-element__color a").Map(func(i int, s *goquery.Selection) string {
		color, _ := s.Attr("aria-label")
		return color
	}), ", ")
	article := doc.Find(".description-definition").First().Text()
	description := strings.TrimSpace(strings.TrimPrefix(doc.Find("[data-v-b033c331]").Text(), "Описание: "))
	composition := doc.Find(".description-definition").Eq(1).Text()
	care := doc.Find(".description-definition").Eq(2).Text()
	images := doc.Find(".swiper-slide img").Map(func(i int, s *goquery.Selection) string {
		imgURL, _ := s.Attr("src")
		return imgURL
	})

	// Извлекаем размеры и удаляем дубликаты
	sizeMap := make(map[string]bool)
	doc.Find(".sku-select__list li").Each(func(i int, s *goquery.Selection) {
		size := strings.TrimSpace(s.Text())
		sizeMap[size] = true
	})
	var sizes []string
	for size := range sizeMap {
		sizes = append(sizes, size)
	}

	fmt.Println("Product data extracted")
	return ProductData{
		Title:       title,
		Price:       price,
		Colors:      colors,
		Sizes:       sizes,
		Article:     article,
		Description: description,
		Composition: composition,
		Care:        care,
		Images:      images,
	}, nil
}

func fetchHTMLWithChromedp(url string) (string, error) {
	ctx, cancel := chromedp.NewExecAllocator(context.Background(),
		append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.Flag("headless", true))...)
	defer cancel()

	ctx, cancel = chromedp.NewContext(ctx)
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 60*time.Second) // Увеличиваем время ожидания до 60 секунд
	defer cancel()

	var html string
	err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.Sleep(5*time.Second), // Увеличиваем время ожидания загрузки страницы
		chromedp.OuterHTML(`html`, &html),
	)
	if err != nil {
		return "", fmt.Errorf("error running chromedp: %v", err)
	}

	return html, nil
}

func fetchDocumentWithRetry(url string, maxRetries int, delay time.Duration) (*goquery.Document, error) {
	var doc *goquery.Document
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		doc, err = fetchDocument(url)
		if err == nil {
			return doc, nil
		}
		log.Printf("Attempt %d failed: %v. Retrying...", attempt+1, err)
		time.Sleep(delay) // Небольшая задержка перед повторной попыткой
	}

	return nil, fmt.Errorf("error fetching URL %s after %d attempts: %v", url, maxRetries, err)
}

func fetchDocument(url string) (*goquery.Document, error) {
	client := &http.Client{
		Timeout: 30 * time.Second, // Увеличиваем время ожидания для HTTP-запросов
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching URL %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d for URL %s", resp.StatusCode, url)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing document from URL %s: %v", url, err)
	}

	return doc, nil
}

func removeDigits(s string) string {
	re := regexp.MustCompile(`\d`)
	return re.ReplaceAllString(s, "")
}