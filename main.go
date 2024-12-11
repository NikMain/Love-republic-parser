package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Структура для данных о категории
type CategoryData struct {
	Status int `json:"status"`
	Data struct {
		Variables struct {
			SectionId   int    `json:"sectionId"`
			SectionName string `json:"sectionName"`
			SectionCode string `json:"sectionCode"`
		} `json:"variables"`
	} `json:"data"`
	Errors   []interface{} `json:"errors"`
	Messages []interface{} `json:"messages"`
}

// Структура для данных о товарах
type ProductsData struct {
	Status int `json:"status"`
	Data struct {
		Pagination struct {
			Total int `json:"total"`
			Pages int `json:"pages"`
		} `json:"pagination"`
		Items []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"items"`
	} `json:"data"`
	Errors   []interface{} `json:"errors"`
	Messages []interface{} `json:"messages"`
}

// Структура для данных о товаре
type ProductDetails struct {
	Status int `json:"status"`
	Data struct {
		URL string `json:"url"`
	} `json:"data"`
	Errors   []interface{} `json:"errors"`
	Messages []interface{} `json:"messages"`
}

// Функция для создания JSON запроса
func toJSON(data map[string]interface{}) (*http.Request, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании JSON данных: %w", err)
	}
	req, err := http.NewRequest("POST", "https://loverepublic.ru/api/catalog", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании HTTP запроса: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func main() {
	categoryURL := "https://loverepublic.ru/catalog/odezhda/" // Замените на нужный URL

	// 1. Получение информации о категории
	categoryResponse, err := http.Get(fmt.Sprintf("https://loverepublic.ru/api/catalog?url=%s", categoryURL))
	if err != nil {
		log.Fatalf("Ошибка при получении информации о категории: %v", err)
	}
	defer categoryResponse.Body.Close()

	var categoryData CategoryData
	if err := json.NewDecoder(categoryResponse.Body).Decode(&categoryData); err != nil {
		log.Fatalf("Ошибка декодирования JSON для категории: %v", err)
	}

	if categoryData.Status != 200 {
		log.Fatalf("Ошибка API при получении категории: Статус %d, Ошибки: %v, Сообщения: %v", categoryData.Status, categoryData.Errors, categoryData.Messages)
	}

	sectionID := categoryData.Data.Variables.SectionId
	sectionName := categoryData.Data.Variables.SectionName
	fmt.Printf("Информация о категории: %s (ID: %d, Код: %s)\n", sectionName, sectionID, categoryData.Data.Variables.SectionCode)

	// 2. Получение информации о товарах
	client := &http.Client{Timeout: 10 * time.Second}
	for page := 0; ; page++ {
		payload := map[string]interface{}{
			"uri":    fmt.Sprintf("/catalog/sections/%d/products/", sectionID),
			"city":   "Москва",
			"order":  "DESC",
			"sort":   "SORT",
			"offset": page * 12,
			"limit":  12,
		}

		req, err := toJSON(payload)
		if err != nil {
			log.Fatalf("Ошибка при создании JSON запроса: %v", err)
		}

		productsResponse, err := client.Do(req)
		if err != nil {
			log.Fatalf("Ошибка при выполнении запроса к API товаров: %v", err)
		}
		defer productsResponse.Body.Close()

		var productsData ProductsData
		if err := json.NewDecoder(productsResponse.Body).Decode(&productsData); err != nil {
			log.Fatalf("Ошибка декодирования JSON для товаров: %v", err)
		}

		if productsData.Status != 200 {
			log.Fatalf("Ошибка API при получении товаров: Статус %d, Ошибки: %v, Сообщения: %v", productsData.Status, productsData.Errors, productsData.Messages)
		}

		if len(productsData.Data.Items) == 0 {
			break // Нет больше товаров на следующих страницах
		}

		fmt.Printf("Страница %d из %d\n", page+1, productsData.Data.Pagination.Pages)

		for _, item := range productsData.Data.Items {
			// 3. Получение информации о каждом товаре
			productResponse, err := client.Get(fmt.Sprintf("https://loverepublic.ru/api/catalog/%d", item.ID))
			if err != nil {
				log.Printf("Ошибка при получении информации о товаре %d: %v", item.ID, err)
				continue
			}
			defer productResponse.Body.Close()

			var productDetails ProductDetails
			if err := json.NewDecoder(productResponse.Body).Decode(&productDetails); err != nil {
				log.Printf("Ошибка декодирования JSON для товара %d: %v", item.ID, err)
				continue
			}

			if productDetails.Status != 200 {
				log.Printf("Ошибка API при получении товара %d: Статус %d, Ошибки: %v, Сообщения: %v", item.ID, productDetails.Status, productDetails.Errors, productDetails.Messages)
				continue
			}

			fmt.Printf("  Товар: %s (ID: %d) - %s\n", item.Name, item.ID, productDetails.Data.URL)
		}
		time.Sleep(2 * time.Second) // Добавлена задержка для предотвращения перегрузки сервера
	}
}