package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	BaseURL              = "https://loverepublic.ru"
	APIEndpoint          = BaseURL + "/api/catalog"
	ProductsPerPage      = 12
	RequestTimeout       = 30 * time.Second
	DelayBetweenRequests = 1 * time.Second
	OutputFileName       = "loverepublic_products.json"
)

// Структуры для JSON
type CategoryResponse struct {
	Status int `json:"status"`
	Data   struct {
		Variables struct {
			SectionId   int    `json:"sectionId"`
			SectionCode string `json:"sectionCode"`
			SectionName string `json:"sectionName"`
		} `json:"variables"`
	} `json:"data"`
}

type ProductsResponse struct {
	Status int `json:"status"`
	Data   struct {
		Items      []Product `json:"items"`
		Pagination struct {
			Total int `json:"total"`
			Pages int `json:"pages"`
			Limit int `json:"limit"`
		} `json:"pagination"`
	} `json:"data"`
}

type Product struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Article     string `json:"article"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Color       Color  `json:"color"`
	Price       Price  `json:"price"`
	SKU         []SKU  `json:"sku"`
	Images      Images `json:"images"`
	Properties  struct {
		Composition ProductProperty `json:"cml2Sostav"`
		Care       ProductProperty `json:"cml2Uhod"`
	} `json:"properties"`
}

type Color struct {
	ColorName   string `json:"colorName"`
	ColorCommon string `json:"colorCommon"`
	ColorHex    string `json:"colorHex"`
}

type Price struct {
	Value           int    `json:"value"`
	DiscountValue   int    `json:"discountValue"`
	DiscountPercent int    `json:"discountPercent"`
	Currency        string `json:"currency"`
}

type SKU struct {
	ID       int    `json:"id"`
	Size     string `json:"size"`
	Quantity int    `json:"quantity"`
	Stock    []struct {
		StoreID  int `json:"storeId"`
		Quantity int `json:"quantity"`
	} `json:"stock"`
}

type Images struct {
	Original []string `json:"original"`
	Detail   []string `json:"detail"`
}

// Добавляем новые структуры перед существующими
type ColorVariant struct {
	ID     int    `json:"id"`
	Color  Color  `json:"color"`
	Price  Price  `json:"price"`
	SKU    []SKU  `json:"sku"`
	Images Images `json:"images"`
}

type MergedProduct struct {
	Article     string         `json:"article"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	URL         string         `json:"url"`
	Composition string         `json:"composition"`
	Care        string         `json:"care"`
	Variants    []ColorVariant `json:"variants"`
}

// Добавляем новую структуру для свойств
type ProductProperty struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HTTP клиент с таймаутом
var client = &http.Client{
	Timeout: RequestTimeout,
}

// Получение данных категории
func getCategory(categoryURL string) (*CategoryResponse, error) {
	url := fmt.Sprintf("%s?url=%s", APIEndpoint, categoryURL)

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ошибка при запросе категории: %w", err)
	}
	defer resp.Body.Close()

	var category CategoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&category); err != nil {
		return nil, fmt.Errorf("ошибка при декодировании ответа категории: %w", err)
	}

	return &category, nil
}

// Получение списка продуктов
func getProducts(sectionID int, page int) (*ProductsResponse, error) {
	body := map[string]interface{}{
		"uri":    fmt.Sprintf("/catalog/sections/%d/products/", sectionID),
		"city":   "Москва",
		"order":  "DESC",
		"sort":   "SORT",
		"offset": (page - 1) * ProductsPerPage,
		"limit":  ProductsPerPage,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании тела запроса: %w", err)
	}

	req, err := http.NewRequest("POST", APIEndpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка при выполнении запроса: %w", err)
	}
	defer resp.Body.Close()

	var products ProductsResponse
	if err := json.NewDecoder(resp.Body).Decode(&products); err != nil {
		return nil, fmt.Errorf("ошибка при декодировании ответа продуктов: %w", err)
	}

	// После получения ответа, добавляем URL для каждого продукта
	for i := range products.Data.Items {
		// Формируем URL в соответствии с форматом сайта
		products.Data.Items[i].URL = fmt.Sprintf("%s/catalog/odezhda/%d/", 
			BaseURL, products.Data.Items[i].ID)
	}

	return &products, nil
}

// Очистка описания продукта
func cleanDescription(desc string) string {
	desc = html.UnescapeString(desc)

	// Удаляем HTML-теги и заменяем переносы строк
	replacements := map[string]string{
		"<ul>":  "",
		"</ul>": "",
		"<li>":  "",
		"</li>": " ",
		"\\n":   " ",
		"\\r":   " ",
		"\n":    " ",
		"\r":    " ",
	}

	for old, new := range replacements {
		desc = strings.ReplaceAll(desc, old, new)
	}

	// Нормализация пробелов
	for strings.Contains(desc, "  ") {
		desc = strings.ReplaceAll(desc, "  ", " ")
	}

	return strings.TrimSpace(desc)
}

// Подготовка файла для записи
func prepareOutputFile() (*os.File, error) {
	// Удаляем старый файл если существует
	if _, err := os.Stat(OutputFileName); err == nil {
		if err := os.Remove(OutputFileName); err != nil {
			return nil, fmt.Errorf("ошибка при удалении старого файла: %w", err)
		}
		fmt.Println("Старый файл удален")
	}

	file, err := os.Create(OutputFileName)
	if err != nil {
		return nil, fmt.Errorf("ошибка при создании файла: %w", err)
	}

	return file, nil
}

func main() {
	// Подготовка файла
	file, err := prepareOutputFile()
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Получение данных категории
	category, err := getCategory(BaseURL + "/catalog/odezhda/")
	if err != nil {
		log.Fatal(err)
	}

	// Получение первой страницы для определения общего количества
	firstPage, err := getProducts(category.Data.Variables.SectionId, 1)
	if err != nil {
		log.Fatal(err)
	}

	totalPages := firstPage.Data.Pagination.Pages
	var allProducts []Product

	// Обработка всех страниц
	for page := 1; page <= totalPages; page++ {
		products, err := getProducts(category.Data.Variables.SectionId, page)
		if err != nil {
			log.Printf("Ошибка при получении страницы %d: %v", page, err)
			continue
		}

		// Очистка описаний
		for i := range products.Data.Items {
			products.Data.Items[i].Description = cleanDescription(products.Data.Items[i].Description)
		}

		allProducts = append(allProducts, products.Data.Items...)
		fmt.Printf("Обработана страница %d из %d (всего товаров: %d)\n",
			page, totalPages, len(allProducts))

		time.Sleep(DelayBetweenRequests)
	}

	// Группировка продуктов по артикулу
	mergedProducts := make(map[string]*MergedProduct)

	// При группировке продуктов добавляем информацию о составе и уходе
	for _, product := range allProducts {
		if merged, exists := mergedProducts[product.Article]; exists {
			variant := ColorVariant{
				ID:     product.ID,
				Color:  product.Color,
				Price:  product.Price,
				SKU:    product.SKU,
				Images: product.Images,
			}
			merged.Variants = append(merged.Variants, variant)
		} else {
			mergedProducts[product.Article] = &MergedProduct{
				Article:     product.Article,
				Name:        product.Name,
				Description: product.Description,
				URL:         fmt.Sprintf("%s/catalog/odezhda/%d/", BaseURL, product.ID),
				Composition: product.Properties.Composition.Value,
				Care:        product.Properties.Care.Value,
				Variants: []ColorVariant{{
					ID:     product.ID,
					Color:  product.Color,
					Price:  product.Price,
					SKU:    product.SKU,
					Images: product.Images,
				}},
			}
		}
	}

	// Создаем финальный слайс продуктов
	var finalProducts []MergedProduct
	for _, product := range mergedProducts {
		finalProducts = append(finalProducts, *product)
	}

	// Записываем результат в файл
	jsonData, err := json.MarshalIndent(finalProducts, "", "    ")
	if err != nil {
		log.Fatal("Ошибка при создании JSON:", err)
	}

	if err := os.WriteFile(OutputFileName, jsonData, 0644); err != nil {
		log.Fatal("Ошибка при записи файла:", err)
	}

	fmt.Printf("Парсинг завершен. Сохранено %d уникальных товаров в %s\n",
		len(finalProducts), OutputFileName)
}
