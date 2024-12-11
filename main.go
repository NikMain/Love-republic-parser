package main

import (
 "bytes"
 "encoding/json"
 "fmt"
 "log"
 "net/http"
 "os"
 "strconv"
 "time"
)

// Структура для данных о категории
type CategoryData struct {
 Status int json:"status"
 Data   struct {
  Variables struct {
   SectionID   int    json:"sectionId"
   SectionName string json:"sectionName"
  } json:"variables"
 } json:"data"
}

// Структура для данных о товарах
type ProductsData struct {
 Status int json:"status"
 Data   struct {
  Pagination struct {
   Total int json:"total"
   Pages int json:"pages"
  } json:"pagination"
  Items []struct {
   ID int json:"id"
  } json:"items"
 } json:"data"
}

// Кастомный тип для обработки цены (float64 или string)
type Price float64

func (p *Price) UnmarshalJSON(b []byte) error {
 var raw interface{}
 if err := json.Unmarshal(b, &raw); err != nil {
  return err
 }
 switch v := raw.(type) {
 case float64:
  *p = Price(v)
 case string:
  parsed, err := strconv.ParseFloat(v, 64)
  if err != nil {
   return err
  }
  *p = Price(parsed)
 default:
  return fmt.Errorf("unexpected type for price: %T", v)
 }
 return nil
}

// Структура для данных о товаре
type ProductDetail struct {
 Status int json:"status"
 Data   struct {
  ID          int     json:"id"
  Name        string  json:"name"
  Price       Price   json:"price"
  Description string  json:"description"
 } json:"data"
 Errors []string json:"errors"
}

func main() {
 // Пример URL для запросов
 categoryURL := "https://example.com/categories"
 productsURL := "https://example.com/products"
 productDetailURL := "https://example.com/product"

 // HTTP клиент с таймаутом
 client := &http.Client{
  Timeout: 10 * time.Second,
 }

 // Получение категории
 category, err := fetchCategory(client, categoryURL)
 if err != nil {
  log.Fatalf("Ошибка при получении категории: %v", err)
 }
 log.Printf("Категория: %+v", category)

 // Получение товаров
 products, err := fetchProducts(client, productsURL)
 if err != nil {
  log.Fatalf("Ошибка при получении товаров: %v", err)
 }
 log.Printf("Найдено товаров: %d", len(products.Data.Items))

 // Получение деталей по каждому товару
 for _, item := range products.Data.Items {
  detail, err := fetchProductDetail(client, fmt.Sprintf("%s/%d", productDetailURL, item.ID))
  if err != nil {
   log.Printf("Ошибка при получении товара %d: %v", item.ID, err)
   continue
  }
  log.Printf("Детали товара %d: %+v", item.ID, detail)
 }
}

// fetchCategory выполняет запрос категории
func fetchCategory(client *http.Client, url string) (*CategoryData, error) {
 resp, err := client.Get(url)
 if err != nil {
  return nil, fmt.Errorf("ошибка запроса: %w", err)
 }
 defer resp.Body.Close()

 if resp.StatusCode != http.StatusOK {
  return nil, fmt.Errorf("неожиданный статус: %d", resp.StatusCode)
 }

 var data CategoryData
 if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
  return nil, fmt.Errorf("ошибка декодирования JSON: %w", err)
 }
 return &data, nil
}

// fetchProducts выполняет запрос товаров
func fetchProducts(client *http.Client, url string) (*ProductsData, error) {
 resp, err := client.Get(url)
 if err != nil {
  return nil, fmt.Errorf("ошибка запроса: %w", err)
 }
 defer resp.Body.Close()

 if resp.StatusCode != http.StatusOK {
  return nil, fmt.Errorf("неожиданный статус: %d", resp.StatusCode)
 }

 var data ProductsData
 if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
  return nil, fmt.Errorf("ошибка декодирования JSON: %w", err)
 }
 return &data, nil
}

// fetchProductDetail выполняет запрос деталей товара
func fetchProductDetail(client *http.Client, url string) (*ProductDetail, error) {
 resp, err := client.Get(url)
 if err != nil {
  return nil, fmt.Errorf("ошибка запроса: %w", err)
 }
 defer resp.Body.Close()

 if resp.StatusCode != http.StatusOK {
  return nil, fmt.Errorf("неожиданный статус: %d", resp.StatusCode)
 }

 var data ProductDetail
 if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
  return nil, fmt.Errorf("ошибка декодирования JSON: %w", err)
 }
 return &data, nil
}