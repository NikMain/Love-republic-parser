package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"sync"
)

// Структура для информации о пагинации
type Pagination struct {
	Total   int `json:"total"`
	Pages   int `json:"pages"`
	Current int `json:"current"`
	Limit   int `json:"limit"`
}

// Структура для цвета
type Color struct {
	Color       string `json:"color"`
	ColorName   string `json:"colorName"`
	ColorCommon string `json:"colorCommon"`
	ColorHex    string `json:"colorHex"`
}

// Структура для свойств товара
type Properties struct {
	Stiker struct {
		Code       string      `json:"code"`
		Name       string      `json:"name"`
		Value      int         `json:"value"`
		CustomValue struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"customValue"`
	} `json:"stiker"`
	Stana struct {
		Code string `json:"code"`
		Name string `json:"name"`
		Value string `json:"value"`
	} `json:"strana"`
	Cml2Sostav struct {
		Code string `json:"code"`
		Name string `json:"name"`
		Value string `json:"value"`
	} `json:"cml2Sostav"`
	Cml2Uhod struct {
		Code string `json:"code"`
		Name string `json:"name"`
		Value string `json:"value"`
	} `json:"cml2Uhod"`
	Season struct {
		Code string `json:"code"`
		Name string `json:"name"`
		Value string `json:"value"`
	} `json:"season"`
	Preorder struct {
		Code string `json:"code"`
		Name string `json:"name"`
		Value string `json:"value"`
	} `json:"preorder"`
	FitWith struct {
		Code string   `json:"code"`
		Name string   `json:"name"`
		Value []int   `json:"value"`
	} `json:"fitWith"`
	Colors struct {
		Code string `json:"code"`
		Name string `json:"name"`
		Value string `json:"value"`
	} `json:"colors"`
}

// Структура для цены товара
type Price struct {
	Value             int    `json:"value"`
	DiscountValue     int    `json:"discountValue"`
	DiscountPercent   int    `json:"discountPercent"`
	Currency          string `json:"currency"`
	FormatValue       string `json:"formatValue"`
	FormatDiscountValue string `json:"formatDiscountValue"`
	AmountOfPayment   int    `json:"amountOfPayment"`
}

// Структура для склада
type Stock struct {
	StoreID  int `json:"storeId"`
	Quantity int `json:"quantity"`
}

// Структура для SKU
type Sku struct {
	ID         int       `json:"id"`
	Size       string    `json:"size"`
	Color      Color     `json:"color"`
	Ean        string    `json:"ean"`
	Properties struct {
		StarayaTsena struct {
			Code string `json:"code"`
			Name string `json:"name"`
			Value string `json:"value"`
		} `json:"starayaTsena"`
	} `json:"properties"`
	Quantity int      `json:"quantity"`
	Stock    []Stock `json:"stock"`
}

// Структура для изображений
type Images struct {
	Thumb    []string `json:"thumb"`
	List     []string `json:"list"`
	Detail   []string `json:"detail"`
	Original []string `json:"original"`
	List150   []string `json:"list150"`
	List375   []string `json:"list375"`
}

// Структура для видео
type Videos struct {
	Order int    `json:"order"`
	Xs    string `json:"xs"`
	Sm    string `json:"sm"`
	Md    string `json:"md"`
	Lg    string `json:"lg"`
}

// Структура для товара
type Item struct {
	ID              int          `json:"id"`
	Name            string       `json:"name"`
	DetailName      string       `json:"detailName"`
	SectionId       int          `json:"sectionId"`
	SectionName     string       `json:"sectionName"`
	SectionPromoName string       `json:"sectionPromoName"`
	Link            string       `json:"link"`
	SectionLink     string       `json:"sectionLink"`
	Description     string       `json:"description"`
	Suite           struct {
		PreviewPicture interface{} `json:"previewPicture"`
		DetailPicture  interface{} `json:"detailPicture"`
	} `json:"suite"`
	Article  string     `json:"article"`
	Meta     struct {
		Title       string `json:"title"`
		Keywords    string `json:"keywords"`
		Description string `json:"description"`
		H1          string `json:"h1"`
	} `json:"meta"`
	IsFullModel bool       `json:"isFullModel"`
	Properties  Properties `json:"properties"`
	Color       Color       `json:"color"`
	Price       Price       `json:"price"`
	IsAvailable bool       `json:"isAvailable"`
	IsAvailableStores bool `json:"isAvailableStores"`
	Sku         []Sku       `json:"sku"`
	Images      Images      `json:"images"`
	Videos      Videos      `json:"videos"`
	Stores      []interface{} `json:"stores"`
}

type Breadcrumb struct {
	Name string `json:"name"`
	Url  string `json:"url"`
}

// Структура для полного ответа getProductsInCategory
type GetProductsResponse struct {
	Status   int           `json:"status"`
	Data     ProductsData  `json:"data"` // Объект, а не массив
	Errors   []interface{} `json:"errors"`
	Messages []interface{} `json:"messages"`
}

type ProductsData struct {
	Items      []Item      `json:"items"`
	Pagination Pagination `json:"pagination"`
}

// Структура для полного ответа getProductDetails
type GetProductDetailsResponse struct {
	ID              int          `json:"id"`
	Name            string       `json:"name"`
	DetailName      string       `json:"detailName"`
	SectionId       int          `json:"sectionId"`
	SectionName     string       `json:"sectionName"`
	SectionPromoName string       `json:"sectionPromoName"`
	Breadcrumbs     []Breadcrumb `json:"breadcrumbs"`
	Link            string       `json:"link"`
	SectionLink     string       `json:"sectionLink"`
	Description     string       `json:"description"`
	Suite           struct {
		PreviewPicture interface{} `json:"previewPicture"`
		DetailPicture  interface{} `json:"detailPicture"`
	} `json:"suite"`
	Article  string     `json:"article"`
	Meta     struct {
		Title       string `json:"title"`
		Keywords    string `json:"keywords"`
		Description string `json:"description"`
		H1          string `json:"h1"`
	} `json:"meta"`
	IsFullModel bool       `json:"isFullModel"`
	Properties  Properties `json:"properties"`
	Color       Color       `json:"color"`
	Price       Price       `json:"price"`
	IsAvailable bool       `json:"isAvailable"`
	IsAvailableStores bool `json:"isAvailableStores"`
	Sku         []Sku       `json:"sku"`
	Images      Images      `json:"images"`
	Videos      Videos      `json:"videos"`
	Stores      []interface{} `json:"stores"`
}

// Функция для получения списка товаров из категории
func getProductsInCategory(sectionID int, offset int, limit int, city string) (*ProductsData, error) {
	values := url.Values{}
	values.Add("url", fmt.Sprintf("/catalog/sections/%d/products/", sectionID)) // Исправлено: "uri" -> "url"
	values.Add("city", city)
	values.Add("order", "DESC")
	values.Add("sort", "SORT")
	values.Add("offset", strconv.Itoa(offset))
	values.Add("limit", strconv.Itoa(limit))

	url := fmt.Sprintf("https://loverepublic.ru/api/catalog?%s", values.Encode())
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var response GetProductsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	if response.Status != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code %d", response.Status)
	}

	return &response.Data, nil
}

// Функция для получения информации о товаре
func getProductDetails(itemID int) (*GetProductDetailsResponse, error) {
	url := fmt.Sprintf("https://loverepublic.ru/api/catalog/%d", itemID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status code %d", resp.StatusCode)
	}

	var response GetProductDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode JSON response: %w", err)
	}

	return &response, nil
}

func main() {
	sectionID := 68
	offset := 0
	limit := 12
	city := "Пермь"

	productsData, err := getProductsInCategory(sectionID, offset, limit, city)
	if err != nil {
		log.Fatal(err)
	}

	if productsData != nil {
		fmt.Printf("Total products: %d, Pages: %d\n", productsData.Pagination.Total, productsData.Pagination.Pages)

		var wg sync.WaitGroup
		for _, item := range productsData.Items {
			wg.Add(1)
			go func(item Item) {
				defer wg.Done()
				details, err := getProductDetails(item.ID)
				if err != nil {
					fmt.Printf("Error getting details for product %d: %v\n", item.ID, err)
					return
				}
				fmt.Printf("Product ID: %d, Name: %s, Details: %+v\n", item.ID, item.Name, details)
			}(item)
		}
		wg.Wait()
	} else {
		fmt.Println("No products found.")
	}
}